package buffer

import (
	"crypto/rand"
	"fmt"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func RandomString(length int) string {
	b := make([]byte, length+2)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[2 : length+2]
}

var _ = Describe("RepeatableBuffer", func() {
	readall := func(r io.Reader) string {
		buf, err := io.ReadAll(r)
		Expect(err).ToNot(HaveOccurred())
		return string(buf)
	}

	It("origin and fork has independent offset internal state", func() {
		origin := NewRepeatableBuffer()
		origin.Write([]byte("hello"))
		fork := origin.Fork()

		Expect("hello").To(Equal(readall(origin)))
		Expect("hello").To(Equal(readall(fork)))
	})

	It("fork would be sync with origin", func() {
		origin := NewRepeatableBuffer()
		origin.Write([]byte("hello"))
		fork := origin.Fork()

		// write long string to force buffer grow multiple times
		longStrA := RandomString(1024)
		origin.Write([]byte(longStrA))

		// change buffer's offset using Read
		Expect("hello" + longStrA).To(Equal(readall(origin)))

		longStrB := RandomString(1024)
		origin.Write([]byte(longStrB))

		Expect(longStrB).To(Equal(readall(origin)))
		Expect("hello" + longStrA + longStrB).To(Equal(readall(fork)))
	})

	// run test with `ginkgo --race`
	It("buffer and forks should not race with each other", func() {
		N := 10
		origin := NewRepeatableBuffer()

		go func() {
			for i := 0; i < N; i++ {
				origin.Write([]byte(RandomString(i)))
			}
		}()
		for i := 0; i < N; i++ {
			go func() {
				fork := origin.Fork()
				readall(fork)
			}()
		}
	})
})
