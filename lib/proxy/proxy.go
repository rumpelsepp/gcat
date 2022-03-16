package proxy

import (
	"errors"
	"io"
	"net"
	"strings"
)

var ErrProxyNotSupported = errors.New("operation not supported")

type ProxyScheme string

const (
	ProxySchemeExec      ProxyScheme = "exec"
	ProxySchemeSTDIO                 = "stdio"
	ProxySchemeTCP                   = "tcp"
	ProxySchemeTCPListen             = "tcp-listen"
	ProxySchemeTLS                   = "tls"
	ProxySchemeTLSListen             = "tls-listen"
	ProxySchemeTUN                   = "tun"
	ProxySchemeWS                    = "ws"
	ProxySchemeWSListen              = "ws-listen"
)

func (s ProxyScheme) IsListener() bool {
	if strings.Contains(string(s), "listen") {
		return true
	}
	return false
}

type ProxyDialer interface {
	Dial() (io.ReadWriteCloser, error)
}

type ProxyListener interface {
	Listen() (net.Listener, error)
}

type Proxy interface {
	Connect() (net.Conn, error)
}
