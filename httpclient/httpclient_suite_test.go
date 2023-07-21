package httpclient

import (
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHttpclient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Httpclient Suite")
}

var mockCtrl *gomock.Controller

var _ = BeforeEach(func() {
	mockCtrl = gomock.NewController(GinkgoT())
})

var _ = BeforeSuite(func() {
})

var _ = AfterEach(func() {
	mockCtrl.Finish()
})
