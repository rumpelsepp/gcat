package gcat

import (
	"crypto/tls"
	"io"
	"net"
)

type ProxyTLS struct {
	Network string
	Address string
	Config  *tls.Config
}

func (p *ProxyTLS) Dial() (io.ReadWriteCloser, error) {
	return tls.Dial(p.Network, p.Address, p.Config) // TODO: use interal dialer to make use of timeouts and shit
}

type ProxyTLSListener struct {
	Network string
	Address string
	Config  *tls.Config
}

func (p *ProxyTLSListener) Listen() (net.Listener, error) {
	return tls.Listen(p.Network, p.Address, p.Config)
}
