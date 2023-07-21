package httpclient

import "net/http"

//go:generate mockgen -destination=mock_http_request_doer.go -package=httpclient github.com/zckevin/go-libs/httpclient HTTPRequestDoer
type HTTPRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}
