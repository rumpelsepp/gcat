package tcp

import (
	"net"

	"github.com/rumpelsepp/gcat/lib/proxy"
)

type ProxyTCP struct{}

func (p *ProxyTCP) Dial(prox *proxy.Proxy) (net.Conn, error) {
	return net.Dial("tcp", net.JoinHostPort(prox.GetStringOption("Hostname"), prox.GetStringOption("Port")))
}

type ProxyTCPListener struct {
	listener net.Listener
}

func (p *ProxyTCPListener) IsListening() bool {
	if p.listener == nil {
		return false
	}
	return true
}

func (p *ProxyTCPListener) Listen(prox *proxy.Proxy) error {
	if p.IsListening() {
		return proxy.ErrProxyBusy
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(prox.GetStringOption("Hostname"), prox.GetStringOption("Port")))
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

func (p *ProxyTCPListener) Close() error {
	if p.IsListening() {
		return p.listener.Close()
	}
	return nil
}

func init() {
	proxy.Registry.Add(proxy.Proxy{
		Scheme:           "tcp",
		Description:      "connect to a tcp host:port",
		SupportsMultiple: true,
		Examples: []string{
			"$ gcat proxy tcp://localhost:1234 -",
		},
		Dialer: &ProxyTCP{},
		StringOptions: []proxy.ProxyOption[string]{
			{
				Name:        "Hostname",
				Description: "target ip address with port",
				Default:     "localhost",
			},
			{
				Name:        "Port",
				Description: "tcp connect port",
				Default:     "1234",
			},
		},
	})
	proxy.Registry.Add(proxy.Proxy{
		Scheme:           "tcp-listen",
		Description:      "tcp listen on host:port",
		SupportsMultiple: true,
		Examples: []string{
			"$ gcat proxy tcp-listen://localhost:1234 -",
		},
		Listener: &ProxyTCPListener{},
		StringOptions: []proxy.ProxyOption[string]{
			{
				Name:        "Hostname",
				Description: "listening ip address",
				Default:     "localhost",
			},
			{
				Name:        "Port",
				Description: "tcp listening port",
				Default:     "1234",
			},
		},
	})
}
