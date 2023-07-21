package httpclient

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"net/http/httputil"

	buffer "github.com/zckevin/go-libs/repeatable_buffer"
)

func cloneResponse(origin http.Response) *http.Response {
	// copy resp headers
	b, _ := httputil.DumpResponse(&origin, false)
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(b)), origin.Request)
	if err != nil {
		panic(err)
	}
	if origin.Body != nil {
		if body, ok := origin.Body.(buffer.RepeatableStreamWrapper); ok {
			resp.Body = body.Fork()
		} else {
			panic("origin body must be RepeatableStreamWrapper")
		}
	}
	return resp
}

func wrapResponse(resp *http.Response) *http.Response {
	if resp == nil || resp.Body == nil {
		return resp
	}
	// already wrapped
	if _, ok := resp.Body.(buffer.RepeatableStreamWrapper); ok {
		return resp
	}

	onEof := func(fullBody io.Reader, err error) {
		// close origin body from net/http
		defer resp.Body.Close()
	}
	resp.Body = buffer.NewRepeatableStreamWrapper(resp.Body, onEof)
	return resp
}
