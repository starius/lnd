package testconn

import (
	"errors"
	"net"
	"time"
)

var ErrBrokenNetwork = errors.New("testing network failure")

type TestConn struct {
	conn net.Conn

	brokenNetwork func() bool

	// Once network is broken, connection is always broken.
	brokenConn bool
}

func New(conn net.Conn, brokenNetwork func() bool) *TestConn {
	return &TestConn{
		conn:          conn,
		brokenNetwork: brokenNetwork,
	}
}

func (c *TestConn) broken() bool {
	if c.brokenConn {
		return true
	}
	c.brokenConn = c.brokenNetwork()
	return c.brokenConn
}

func (c *TestConn) Read(b []byte) (n int, err error) {
	if c.broken() {
		return 0, ErrBrokenNetwork
	}
	return c.conn.Read(b)
}

func (c *TestConn) Write(b []byte) (n int, err error) {
	if c.broken() {
		return 0, ErrBrokenNetwork
	}
	return c.conn.Write(b)
}

func (c *TestConn) Close() error {
	return c.conn.Close()
}

func (c *TestConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *TestConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *TestConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *TestConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *TestConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
