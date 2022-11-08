package proxy

import (
	"net"
	"time"
)

type BaseConn struct {
	LocalAddress  *ProxyAddr
	RemoteAddress *ProxyAddr
}

func (c *BaseConn) Read(b []byte) (int, error) {
	return 0, ErrNotImplemented
}

func (c *BaseConn) Write(b []byte) (int, error) {
	return 0, ErrNotImplemented
}

func (c *BaseConn) Close() error {
	return ErrNotImplemented
}

func (c *BaseConn) LocalAddr() net.Addr {
	return c.LocalAddress
}

func (c *BaseConn) RemoteAddr() net.Addr {
	return c.RemoteAddress
}

func (c *BaseConn) SetDeadline(t time.Time) error {
	return ErrNotImplemented
}

func (c *BaseConn) SetReadDeadline(t time.Time) error {
	return ErrNotImplemented
}

func (c *BaseConn) SetWriteDeadline(t time.Time) error {
	return ErrNotImplemented
}
