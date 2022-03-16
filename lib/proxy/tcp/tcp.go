package tcp

import (
	"io"
	"net"
)

type ProxyTCP struct {
	Network string
	Address string
	Dialer  net.Dialer
}

func (p *ProxyTCP) Dial() (io.ReadWriteCloser, error) {
	return p.Dialer.Dial(p.Network, p.Address)
}

type ProxyTCPListener struct {
	Network  string
	Address  string
	listener net.Listener
}

func (p *ProxyTCPListener) Listen() (net.Listener, error) {
	if p.listener == nil {
		ln, err := net.Listen(p.Network, p.Address)
		if err != nil {
			return nil, err
		}
		p.listener = ln
		return ln, nil
	}
	return p.listener, nil
}
