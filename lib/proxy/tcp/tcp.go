package tcp

import (
	"net"

	"github.com/rumpelsepp/gcat/lib/proxy"
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
	return proxy.CreateProxyFromDialer(
		&ProxyTCP{
			Network: "tcp",
			Address: addr.Host,
			Dialer:  net.Dialer{},
		}), nil
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
	return proxy.CreateProxyFromListener(
		&ProxyTCPListener{
			Network: "tcp",
			Address: addr.Host,
		}), nil
}

func init() {
	proxy.Registry.Add(proxy.ProxyEntryPoint{
		Scheme: "tcp",
		Create: CreateTCPProxy,
		Help: proxy.ProxyHelp{
			Description: "connect to a tcp host:port",
			Examples: []string{
				"$ gcat proxy tcp://localhost:1234 -",
			},
			Args: []proxy.ProxyHelpArg{
				{
					Name:        "Host",
					Type:        "string",
					Explanation: "target ip address",
				},
				{
					Name:        "Port",
					Type:        "int",
					Explanation: "target port",
				},
			},
		},
	})

	proxy.Registry.Add(proxy.ProxyEntryPoint{
		Scheme: "tcp-listen",
		Create: CreateTCPListenProxy,
		Help: proxy.ProxyHelp{
			Description: "tcp listen on host:port",
			Examples: []string{
				"$ gcat proxy tcp-listen://localhost:1234 -",
			},
			Args: []proxy.ProxyHelpArg{
				{
					Name:        "Host",
					Type:        "string",
					Explanation: "listening ip address",
				},
				{
					Name:        "Port",
					Type:        "int",
					Explanation: "listening port",
				},
			},
		},
	})
}
