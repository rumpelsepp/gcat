package gcat

import (
	"errors"
	"io"
	"net"
)

var ErrNotSupported = errors.New("operation not supported")

type ProxyDialer interface {
	Dial() (io.ReadWriteCloser, error)
}

type ProxyListener interface {
	Listen() (net.Listener, error)
}

type Proxy interface {
	Connect() (net.Conn, error)
}
