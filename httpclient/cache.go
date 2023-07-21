package httpclient

import (
	"net/http"
	"sync"
)

type CacheKey = string

func simpleGetCacheKey(req *http.Request) CacheKey {
	return req.Method + req.URL.String()
}

type memcacheImpl struct {
	mu    sync.Mutex
	items map[CacheKey]*cacheItem

	getCacheKey func(req *http.Request) CacheKey
}

func NewMemcacheImpl(getCacheKey func(req *http.Request) CacheKey) *memcacheImpl {
	return &memcacheImpl{
		items:       make(map[CacheKey]*cacheItem),
		getCacheKey: getCacheKey,
	}
}

func (c *memcacheImpl) getCacheItem(req *http.Request) (item *cacheItem, ok bool) {
	key := c.getCacheKey(req)

	if item, ok := c.items[key]; ok {
		return item, true
	}
	item = newCacheItem(key, func() {
		c.DeleteItem(key)
	})
	c.items[key] = item
	return item, false
}

func (c *memcacheImpl) GetCacheItem(req *http.Request) (item *cacheItem, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.getCacheItem(req)
}

func (c *memcacheImpl) TryRegister(req *http.Request) (item *cacheItem, waiter *cacheItemWaiter, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ci, ok := c.getCacheItem(req)
	return ci, ci.NewWaiter(), ok
}

func (c *memcacheImpl) DeleteItem(key CacheKey) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}
