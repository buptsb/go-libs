package buffer

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
)

type ForkReader interface {
	Read(p []byte) (n int, err error)
	Fork() *streamWrapperFork
}

var (
	_ ForkReader    = (*streamWrapper)(nil)
	_ io.ReadCloser = (*streamWrapperFork)(nil)
)

type simpleBroadcaster struct {
	mu  sync.RWMutex
	chs []chan struct{}
}

func (b *simpleBroadcaster) Notify() {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.chs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (b *simpleBroadcaster) Register() <-chan struct{} {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan struct{})
	b.chs = append(b.chs, ch)
	return ch
}

func (b *simpleBroadcaster) CloseAll() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, c := range b.chs {
		close(c)
	}
	b.chs = nil
}

type streamWrapper struct {
	r     io.Reader
	buf   RepeatableBuffer
	fork  *streamWrapperFork
	onEOF func(io.Reader, error)

	rerr   atomic.Value
	hasErr chan struct{}

	tryReadCh   chan struct{}
	broadcaster simpleBroadcaster
	children    atomic.Int32
}

func NewRepeatableStreamWrapper(r io.Reader, onEOF func(io.Reader, error)) *streamWrapper {
	sw := &streamWrapper{
		r:         r,
		buf:       NewRepeatableBuffer(),
		onEOF:     onEOF,
		hasErr:    make(chan struct{}),
		tryReadCh: make(chan struct{}, 1),
	}
	sw.fork = sw.Fork()
	return sw
}

func (sw *streamWrapper) setError(err error) {
	if sw.rerr.CompareAndSwap(nil, err) {
		close(sw.hasErr)
		sw.broadcaster.CloseAll()
	}
}

func (sw *streamWrapper) doRead() {
	if err := sw.rerr.Load(); err != nil {
		return
	}
	p := make([]byte, 1024)
	n, err := sw.r.Read(p)
	sw.buf.Write(p[:n])
	if err != nil {
		sw.setError(err)
		if sw.onEOF != nil {
			sw.onEOF(sw.buf, sw.rerr.Load().(error))
		}
	}
	sw.broadcaster.Notify()
}

func (sw *streamWrapper) Read(p []byte) (int, error) {
	return sw.fork.Read(p)
}

func (sw *streamWrapper) Fork() *streamWrapperFork {
	sw.children.Add(1)
	return &streamWrapperFork{
		origin:        sw,
		buf:           sw.buf.Fork(),
		canRead:       sw.broadcaster.Register(),
		localClosedCh: make(chan struct{}),
	}
}

func (sw *streamWrapper) cancel() error {
	// fmt.Println("cancel")
	sw.setError(context.Canceled)
	return nil
}

func (sw *streamWrapper) Close() error {
	sw.fork.Close()
	return nil
}

type streamWrapperFork struct {
	origin *streamWrapper

	buf     RepeatableBufferReader
	canRead <-chan struct{}

	localClosed   atomic.Bool
	localClosedCh chan struct{}
}

func (swf *streamWrapperFork) Read(p []byte) (n int, err error) {
	if p == nil {
		return 0, nil
	}
	if swf.localClosed.Load() {
		return 0, io.EOF
	}

	origin := swf.origin
	for {
		n, _ = swf.buf.Read(p)
		if n > 0 {
			return n, nil
		}
		if rerr := origin.rerr.Load(); rerr != nil {
			// fmt.Println("rerr", rerr)
			n2, err := swf.buf.Read(p)
			if err == nil {
				return n2, nil
			}
			return n2, rerr.(error)
		}
		select {
		case <-swf.localClosedCh:
			return 0, io.EOF
		case origin.tryReadCh <- struct{}{}:
			origin.doRead()
			<-origin.tryReadCh
			continue
		case <-swf.canRead:
			continue
		}
	}
}

func (swf *streamWrapperFork) Close() error {
	if ok := swf.localClosed.CompareAndSwap(false, true); ok {
		close(swf.localClosedCh)
		if new := swf.origin.children.Add(-1); new == 0 {
			swf.origin.cancel()
		}
	}
	return nil
}

func (swf *streamWrapperFork) Fork() *streamWrapperFork {
	return swf.origin.Fork()
}
