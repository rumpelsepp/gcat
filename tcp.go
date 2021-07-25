package gcat

import (
	"io"
	"net"
)

type ProxyTCP struct {
	Network string
	Address string
}

func (p *ProxyTCP) Dial() (io.ReadWriteCloser, error) {
	return net.Dial(p.Network, p.Address) // TODO: use interal dialer to make use of timeouts and shit
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
