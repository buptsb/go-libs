package buffer

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"io"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type testRwc struct {
	buf     bytes.Buffer
	canRead chan struct{}

	mu     sync.Mutex
	closed bool
}

func newTestRwc() *testRwc {
	return &testRwc{
		canRead: make(chan struct{}),
	}
}

func (rwc *testRwc) Read(p []byte) (int, error) {
	for {
		n, _ := rwc.buf.Read(p)
		if n > 0 {
			return n, nil
		}
		rwc.mu.Lock()
		closed := rwc.closed
		rwc.mu.Unlock()
		if closed {
			return 0, io.EOF
		}
		<-rwc.canRead
	}
}

func (rwc *testRwc) Write(p []byte) (int, error) {
	defer func() {
		select {
		case rwc.canRead <- struct{}{}:
		default:
		}
	}()
	return rwc.buf.Write(p)
}

func (rwc *testRwc) Close() error {
	rwc.mu.Lock()
	defer rwc.mu.Unlock()

	rwc.closed = true
	close(rwc.canRead)
	return nil
}

var _ = Describe("streamWrapper", func() {
	go func() {
		_ = http.ListenAndServe("localhost:6060", nil)
	}()

	var (
		source  *testRwc
		wrapper *streamWrapper
	)

	BeforeEach(func() {
		source = newTestRwc()
		wrapper = NewRepeatableStreamWrapper(source, nil)
	})

	It("if all children got closed, parent will be canceled", func() {
		fork := wrapper.Fork()
		wrapper.Close()
		Expect(wrapper.hasErr).ShouldNot(BeClosed())
		fork.Close()
		Expect(wrapper.hasErr).Should(BeClosed())

		_, err := wrapper.Fork().Read(make([]byte, 16))
		Expect(err).To(Equal(context.Canceled))
	})

	It("fuzzing test", func() {
		wroteBytes := bytes.NewBuffer(nil)
		N := 10
		t := 5 * time.Second

		randomSleep := func(upperBound time.Duration) {
			t := time.Duration(rand.Int63n(int64(upperBound)))
			time.Sleep(t)
		}
		randomSizeBuffer := func(upperBound int) []byte {
			n := rand.Intn(upperBound)
			// chance to return short buffer
			if n%5 == 0 {
				n = 1
			}
			return make([]byte, n)
		}
		writer := func(wg *sync.WaitGroup) {
			defer wg.Done()
			defer source.Close()
			defer wrapper.Close()
			expireAt := time.Now().Add(t)
			for {
				if time.Now().After(expireAt) {
					return
				}
				randomSleep(time.Millisecond * 10)
				b := randomSizeBuffer(4096 * 2)
				nr, _ := crand.Read(b)
				nw, _ := source.Write(b[:nr])
				wroteBytes.Write(b[:nw])
			}
		}
		reader := func(wg *sync.WaitGroup, resultCh chan []byte) {
			defer wg.Done()
			fork := wrapper.Fork()
			defer fork.Close()
			readBytes := bytes.NewBuffer(nil)
			for {
				b := randomSizeBuffer(4096 * 2)
				n, err := fork.Read(b)
				if err != nil {
					Expect(err).To(Equal(io.EOF))
					resultCh <- readBytes.Bytes()
					return
				}
				readBytes.Write(b[:n])
			}
		}

		resultCh := make(chan []byte, N)
		var wg sync.WaitGroup
		wg.Add(N + 1)

		go writer(&wg)
		for i := 0; i < N; i++ {
			go reader(&wg, resultCh)
		}
		wg.Wait()

		for i := 0; i < N; i++ {
			Expect(<-resultCh).To(Equal(wroteBytes.Bytes()))
		}
		Expect(wrapper.hasErr).Should(BeClosed())
	})
})
