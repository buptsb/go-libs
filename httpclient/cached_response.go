package httpclient

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type cacheStatus int

const (
	cacheStatusWaitForResponse cacheStatus = iota
	cacheStatusGotResponse
)

// raii style waiter
type cacheItemWaiter struct {
	once sync.Once
	ci   *cacheItem
}

func (w *cacheItemWaiter) WaitForResolved(ctx context.Context) (*http.Response, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-w.ci.resolved:
	}
	return cloneResponse(*w.ci.resp), w.ci.err
}

func (w *cacheItemWaiter) Close() {
	w.once.Do(func() {
		if w.ci.waiters.Add(-1) == 0 && w.ci.onClose != nil {
			w.ci.onClose()
		}
	})
}

type cacheItem struct {
	key     CacheKey
	onClose func()

	status   atomic.Value
	resolved chan struct{}
	resp     *http.Response
	err      error

	expireAt time.Time
	waiters  atomic.Int32
}

func newCacheItem(key CacheKey, onClose func()) *cacheItem {
	ci := &cacheItem{
		key:      key,
		resolved: make(chan struct{}),
		onClose:  onClose,
	}
	ci.status.Store(cacheStatusWaitForResponse)
	return ci
}

func (ci *cacheItem) Resolve(resp *http.Response, err error) (ok bool) {
	if resp == nil && err == nil {
		panic("resp and err can not both be nil")
	}
	ok = ci.status.CompareAndSwap(cacheStatusWaitForResponse, cacheStatusGotResponse)
	if ok {
		ci.resp, ci.err = wrapResponse(resp), err
		close(ci.resolved)
	}
	return ok
}

func (ci *cacheItem) NewWaiter() *cacheItemWaiter {
	ci.waiters.Add(1)
	return &cacheItemWaiter{ci: ci}
}
