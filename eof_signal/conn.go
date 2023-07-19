package eofsignal

import (
	"net"
	"sync"
)

var (
	_ net.Conn = (*eofSignalConn)(nil)
)

type eofSignalConn struct {
	net.Conn

	mu sync.Mutex
	fn func(error)
}

func NewEOFSignalConn(conn net.Conn, fn func(error)) net.Conn {
	c := &eofSignalConn{
		Conn: conn,
		fn:   fn,
	}
	return c
}

func (c *eofSignalConn) Read(p []byte) (n int, err error) {
	n, err = c.Conn.Read(p)
	if err != nil {
		c.condfn(err)
	}
	return n, err
}

func (c *eofSignalConn) Close() error {
	err := c.Conn.Close()
	c.condfn(err)
	return err
}

func (c *eofSignalConn) condfn(err error) {
	c.mu.Lock()
	fn := c.fn
	if fn == nil {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	fn(err)
}
