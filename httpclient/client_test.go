package httpclient

import (
	"net/http"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CachedHTTPClient", func() {
	var (
		cache      *memcacheImpl
		httpclient *MockHTTPRequestDoer
		client     *CachedHTTPClient
		req, _     = http.NewRequest("GET", "http://example.com", nil)
		resp       = &http.Response{StatusCode: 404, Request: req}
	)
	setupMockClient := func(rtt time.Duration, times int) {
		httpclient.EXPECT().Do(req).DoAndReturn(func(*http.Request) (*http.Response, error) {
			time.Sleep(rtt)
			return resp, nil
		}).Times(times)
	}

	BeforeEach(func() {
		cache = NewMemcacheImpl(simpleGetCacheKey)
		httpclient = NewMockHTTPRequestDoer(mockCtrl)
		client = NewCachedHTTPClient(cache, httpclient)
	})

	It("request to same url at same time would be cached", func() {
		setupMockClient(time.Millisecond*50, 1)

		var wg sync.WaitGroup
		wg.Add(2)
		for i := 0; i < 2; i++ {
			go func() {
				defer wg.Done()
				resp, err := client.Do(req)
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(404))
			}()
		}
		wg.Wait()
	})

	It("request to same url subsequently would not use cache item", func() {
		setupMockClient(time.Millisecond*50, 3)

		for i := 0; i < 3; i++ {
			resp, err := client.Do(req)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(404))
			Expect(cache.items).To(HaveLen(0))
		}
	})

	Context("push", func() {
		It("push response would resolve later incoming requests", func() {
			setupMockClient(time.Millisecond*50, 0)

			client.ReceivePush(resp)
			resp, err := client.Do(req)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(404))
			Expect(cache.items).To(HaveLen(0))
		})

		It("push response would resolve early unresolved requests", func() {
			setupMockClient(time.Hour, 1)

			var wg sync.WaitGroup
			wg.Add(3)
			for i := 0; i < 3; i++ {
				go func() {
					defer wg.Done()
					resp, err := client.Do(req)
					Expect(err).To(BeNil())
					Expect(resp.StatusCode).To(Equal(404))
				}()
			}
			go func() {
				time.Sleep(time.Millisecond * 50)
				client.ReceivePush(resp)
			}()
			wg.Wait()
			Expect(cache.items).To(HaveLen(0))
		})
	})
})
