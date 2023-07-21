package httpclient

import (
	"context"
	"math/rand"
	"net/http"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("cachedItem", func() {
	var item *cacheItem

	BeforeEach(func() {
		item = newCacheItem("", nil)
	})

	It("add waiters", func() {
		closeCh := make(chan struct{})
		item.onClose = func() {
			close(closeCh)
		}

		var wg sync.WaitGroup
		wg.Add(10)
		for i := 0; i < 10; i++ {
			go func() {
				defer wg.Done()
				waiter := item.NewWaiter()
				defer waiter.Close()
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(20)))
			}()
		}
		wg.Wait()
		Expect(item.waiters.Load()).To(Equal(int32(0)))
		Expect(closeCh).To(BeClosed())
	})

	It("non resolved response would block wait", func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()

		waiter := item.NewWaiter()
		resp, err := waiter.WaitForResolved(ctx)
		Expect(resp).To(BeNil())
		Expect(err).To(Equal(context.DeadlineExceeded))
	})

	It("can only resolve once", func() {
		resp1, resp2 := &http.Response{StatusCode: 404}, &http.Response{StatusCode: 500}

		ok := item.Resolve(resp1, nil)
		Expect(ok).To(BeTrue())

		ok = item.Resolve(resp2, nil)
		Expect(ok).To(BeFalse())

		waiter := item.NewWaiter()
		resp, err := waiter.WaitForResolved(context.Background())
		Expect(err).To(BeNil())
		Expect(resp.StatusCode).To(Equal(404))
	})
})
