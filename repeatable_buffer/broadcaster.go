package buffer

import "sync"

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
