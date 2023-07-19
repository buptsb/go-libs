package buffer

import (
	"io"
)

type RepeatableBuffer interface {
	RepeatableBufferReader
	Write(p []byte) (n int, err error)
}

type RepeatableBufferReader interface {
	Bytes() []byte
	String() string
	Read(p []byte) (n int, err error)
	Fork() *repeatableBufferFork
}

var (
	_ RepeatableBuffer       = (*repeatableBufferImpl)(nil)
	_ RepeatableBufferReader = (*repeatableBufferFork)(nil)
)

type repeatableBufferImpl struct {
	Buffer
}

func NewRepeatableBuffer() *repeatableBufferImpl {
	return &repeatableBufferImpl{}
}

func (rb *repeatableBufferImpl) Fork() *repeatableBufferFork {
	return newRepeatableBufferFork(&rb.Buffer, 0)
}

type repeatableBufferFork struct {
	origin *Buffer
	off    int
}

func newRepeatableBufferFork(buf *Buffer, off int) *repeatableBufferFork {
	return &repeatableBufferFork{
		origin: buf,
		off:    off,
	}
}

func (rbf *repeatableBufferFork) empty() bool {
	return len(rbf.origin.buf) <= rbf.off
}

func (rbf *repeatableBufferFork) Read(p []byte) (n int, err error) {
	rbf.origin.mu.RLock()
	defer rbf.origin.mu.RUnlock()

	if rbf.empty() {
		if len(p) == 0 {
			return 0, nil
		}
		return 0, io.EOF
	}
	n = copy(p, rbf.origin.buf[rbf.off:])
	rbf.off += n
	return n, nil
}

func (rbf *repeatableBufferFork) Bytes() []byte {
	rbf.origin.mu.RLock()
	defer rbf.origin.mu.RUnlock()

	return rbf.origin.buf[rbf.off:]
}

func (rbf *repeatableBufferFork) String() string {
	return string(rbf.Bytes())
}

func (rbf *repeatableBufferFork) Fork() *repeatableBufferFork {
	return newRepeatableBufferFork(rbf.origin, 0)
}
