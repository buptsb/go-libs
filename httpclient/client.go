package httpclient

/*
users:
  - local proxy, http racing client, race for push + client.Do
  - server proxy, client with cache

status:
  - initilized
  - got response, body not read
  - body fully read
  - ? expired

client.Do()

	getOrCreate
		if has cached entry
			return listener

	if new entry
		go clientDoer()
		return a listerner

	if listener
		resp := <-ln.C
	return resp.Fork()
*/

import "net/http"

type CachedHTTPClient struct {
	cache      *memcacheImpl
	httpclient HTTPRequestDoer
}

func NewCachedHTTPClient(cache *memcacheImpl, httpclient HTTPRequestDoer) *CachedHTTPClient {
	return &CachedHTTPClient{
		cache:      cache,
		httpclient: httpclient,
	}
}

func (cl *CachedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	ci, waiter, ok := cl.cache.TryRegister(req)
	defer waiter.Close()
	if !ok {
		go cl.doRequest(req, ci)
	}
	return waiter.WaitForResolved(req.Context())
}

func (cl *CachedHTTPClient) doRequest(req *http.Request, ci *cacheItem) {
	resp, err := cl.httpclient.Do(req)
	ci.Resolve(resp, err)
}

func (cl *CachedHTTPClient) ReceivePush(resp *http.Response) (ok bool) {
	ci, _ := cl.cache.GetCacheItem(resp.Request)
	return ci.Resolve(resp, nil)
}
