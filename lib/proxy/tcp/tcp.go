package tcp

import (
	"net"

	"codeberg.org/rumpelsepp/gcat/lib/proxy"
)

type ProxyTCP struct {
	Network string
	Address string
	Dialer  net.Dialer
}

func (p *ProxyTCP) Dial() (net.Conn, error) {
	return p.Dialer.Dial(p.Network, p.Address)
}

func CreateTCPProxy(addr *proxy.ProxyAddr) (*proxy.Proxy, error) {
	return &proxy.Proxy{
		Dialer: &ProxyTCP{
			Network: "tcp",
			Address: addr.Host,
			Dialer:  net.Dialer{},
		},
	}, nil
}

type ProxyTCPListener struct {
	Network  string
	Address  string
	listener net.Listener
}

func (p *ProxyTCPListener) IsListening() bool {
	if p.listener == nil {
		return false
	}
	return true
}

func (p *ProxyTCPListener) Listen() error {
	if p.IsListening() {
		return proxy.ErrProxyBusy
	}

	ln, err := net.Listen(p.Network, p.Address)
	if err != nil {
		return err
	}
	p.listener = ln
	return nil
}

func (p *ProxyTCPListener) Accept() (net.Conn, error) {
	if !p.IsListening() {
		return nil, proxy.ErrProxyNotInitialized
	}
	return p.listener.Accept()
}

func CreateTCPListenProxy(addr *proxy.ProxyAddr) (*proxy.Proxy, error) {
	return &proxy.Proxy{
		Listener: &ProxyTCPListener{
			Network: "tcp",
			Address: addr.Host,
		},
	}, nil
}

func init() {
	scheme := proxy.ProxyScheme("tcp")

	proxy.ProxyRegistry[scheme] = proxy.ProxyEntryPoint{
		Scheme:    scheme,
		Create:    CreateTCPProxy,
		ShortHelp: "connect to a tcp host:port",
	}

	scheme = proxy.ProxyScheme("tcp-listen")
	proxy.ProxyRegistry[scheme] = proxy.ProxyEntryPoint{
		Scheme:    scheme,
		Create:    CreateTCPListenProxy,
		ShortHelp: "tcp listen on host:port",
	}
}
