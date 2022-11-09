package unix

import (
	"net"

	"github.com/rumpelsepp/gcat/lib/proxy"
)

type dialer struct{}

func (d *dialer) Dial(prox *proxy.Proxy) (net.Conn, error) {
	return net.Dial("unix", prox.GetStringOption("Path"))
}

type listener struct {
	listener net.Listener
}

func (l *listener) IsListening() bool {
	if l.listener != nil {
		return true
	}
	return false
}

func (l *listener) Listen(prox *proxy.Proxy) error {
	ln, err := net.Listen("unix", prox.GetStringOption("Path"))
	if err != nil {
		return err
	}
	l.listener = ln
	return nil
}

func (l *listener) Accept() (net.Conn, error) {
	return l.listener.Accept()
}

func (l *listener) Close() error {
	return l.listener.Close()
}

func init() {
	proxy.Registry.Add(proxy.Proxy{
		Scheme:           "unix",
		Description:      "connect to a unix domain socket",
		SupportsMultiple: true,
		Examples: []string{
			"$ gcat unix:///tmp.sock -",
		},
		Dialer: &dialer{},
		StringOptions: []proxy.ProxyOption[string]{
			{
				Name:        "Path",
				Description: "path to socket file",
			},
		},
	})
	proxy.Registry.Add(proxy.Proxy{
		Scheme:           "unix-listen",
		Description:      "listen on a unix domain socket",
		SupportsMultiple: true,
		Examples: []string{
			"$ gcat unix-listen:///tmp.sock -",
		},
		Listener: &listener{},
		StringOptions: []proxy.ProxyOption[string]{
			{
				Name:        "Path",
				Description: "path to socket file",
			},
		},
	})
}
