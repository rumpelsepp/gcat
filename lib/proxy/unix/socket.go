package unix

import (
	"net"

	"github.com/rumpelsepp/gcat/lib/proxy"
)

type unixDialer struct{}

func (d *unixDialer) Dial(prox *proxy.Proxy) (net.Conn, error) {
	return net.Dial("unix", prox.GetStringOption("Path"))
}

type unixListener struct {
	listener net.Listener
}

func (l *unixListener) IsListening() bool {
	if l.listener != nil {
		return true
	}
	return false
}

func (l *unixListener) Listen(prox *proxy.Proxy) error {
	ln, err := net.Listen("unix", prox.GetStringOption("Path"))
	if err != nil {
		return err
	}
	l.listener = ln
	return nil
}

func (l *unixListener) Accept() (net.Conn, error) {
	return l.listener.Accept()
}

func (l *unixListener) Close() error {
	return l.listener.Close()
}

type unixgramDialer struct{}

func (d *unixgramDialer) Dial(prox *proxy.Proxy) (net.Conn, error) {
	return net.Dial("unixgram", prox.GetStringOption("Path"))
}

type unixpacketDialer struct{}

func (d *unixpacketDialer) Dial(prox *proxy.Proxy) (net.Conn, error) {
	return net.Dial("unixpacket", prox.GetStringOption("Path"))
}

type unixpacketListener struct {
	listener net.Listener
}

func (l *unixpacketListener) IsListening() bool {
	if l.listener != nil {
		return true
	}
	return false
}

func (l *unixpacketListener) Listen(prox *proxy.Proxy) error {
	ln, err := net.Listen("unixpacket", prox.GetStringOption("Path"))
	if err != nil {
		return err
	}
	l.listener = ln
	return nil
}

func (l *unixpacketListener) Accept() (net.Conn, error) {
	return l.listener.Accept()
}

func (l *unixpacketListener) Close() error {
	return l.listener.Close()
}

var pathOption = proxy.ProxyOption[string]{
	Name:        "Path",
	Description: "path to socket file",
}

func init() {
	proxy.Registry.Add(proxy.Proxy{
		Scheme:           "unix",
		Description:      "dial unix domain socket (SOCK_STREAM)",
		SupportsMultiple: true,
		Examples: []string{
			"$ gcat unix:///tmp.sock -",
		},
		Dialer:        &unixDialer{},
		StringOptions: []proxy.ProxyOption[string]{pathOption},
	})
	proxy.Registry.Add(proxy.Proxy{
		Scheme:           "unix-listen",
		Description:      "listen unix domain socket (SOCK_STREAM)",
		SupportsMultiple: true,
		Examples: []string{
			"$ gcat unix-listen:///tmp.sock -",
		},
		Listener:      &unixListener{},
		StringOptions: []proxy.ProxyOption[string]{pathOption},
	})
	proxy.Registry.Add(proxy.Proxy{
		Scheme:           "unixgram",
		Description:      "dial unix domain socket (SOCK_DGRAM)",
		SupportsMultiple: true,
		Examples: []string{
			"$ gcat unixgram:///tmp.sock -",
		},
		Dialer:        &unixgramDialer{},
		StringOptions: []proxy.ProxyOption[string]{pathOption},
	})
	proxy.Registry.Add(proxy.Proxy{
		Scheme:           "unixpacket",
		Description:      "dial unix domain socket (SOCK_SEQPACKET)",
		SupportsMultiple: true,
		Examples: []string{
			"$ gcat unixpacket:///tmp.sock -",
		},
		Dialer:        &unixpacketDialer{},
		StringOptions: []proxy.ProxyOption[string]{pathOption},
	})
	proxy.Registry.Add(proxy.Proxy{
		Scheme:           "unixpacket-listen",
		Description:      "listen unix domain socket (SOCK_SEQPACKET)",
		SupportsMultiple: true,
		Examples: []string{
			"$ gcat unixpacket-listen:///tmp.sock -",
		},
		Listener:      &unixpacketListener{},
		StringOptions: []proxy.ProxyOption[string]{pathOption},
	})
}
