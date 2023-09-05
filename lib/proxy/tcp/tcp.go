package tcp

import (
	"context"
	"net"

	"github.com/rumpelsepp/gcat/lib/proxy"
)

type dialer struct{}

func (p *dialer) Dial(ctx context.Context, desc *proxy.ProxyDescription) (net.Conn, error) {
	var dialer net.Dialer
	return dialer.DialContext(ctx, "tcp", desc.TargetHost())
}

type listener struct {
	listener net.Listener
}

func (p *listener) IsListening() bool {
	if p.listener == nil {
		return false
	}
	return true
}

func (p *listener) Listen(desc *proxy.ProxyDescription) error {
	if p.IsListening() {
		return proxy.ErrProxyBusy
	}

	ln, err := net.Listen("tcp", desc.TargetHost())
	if err != nil {
		return err
	}
	p.listener = ln
	return nil
}

func (p *listener) Accept() (net.Conn, error) {
	if !p.IsListening() {
		return nil, proxy.ErrProxyNotInitialized
	}
	return p.listener.Accept()
}

func (p *listener) Close() error {
	if p.IsListening() {
		return p.listener.Close()
	}
	return nil
}

func init() {
	proxy.Registry.Add(proxy.ProxyDescription{
		Scheme:           "tcp",
		Description:      "connect to a tcp host:port",
		SupportsMultiple: true,
		Examples: []string{
			"$ gcat proxy tcp://localhost:1234 -",
		},
		Dialer: &dialer{},
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
	proxy.Registry.Add(proxy.ProxyDescription{
		Scheme:           "tcp-listen",
		Description:      "tcp listen on host:port",
		SupportsMultiple: true,
		Examples: []string{
			"$ gcat proxy tcp-listen://localhost:1234 -",
		},
		Listener: &listener{},
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
