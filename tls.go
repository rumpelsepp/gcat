package gcat

import (
	"crypto/tls"
	"io"
	"net"
)

type ProxyTLS struct {
	Network string
	Address string
	Dialer  *tls.Dialer
}

func (p *ProxyTLS) Dial() (io.ReadWriteCloser, error) {
	return p.Dialer.Dial(p.Network, p.Address)
}

type ProxyTLSListener struct {
	Network string
	Address string
	Config  *tls.Config
}

func (p *ProxyTLSListener) Listen() (net.Listener, error) {
	return tls.Listen(p.Network, p.Address, p.Config)
}
