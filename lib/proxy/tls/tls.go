package gtls

import (
	"crypto/tls"
	"net"

	"github.com/rumpelsepp/gcat/lib/proxy"
)

type dialer struct{}

func (d *dialer) Dial(prox *proxy.Proxy) (net.Conn, error) {
	tlsConfig, err := ParseOptions(prox)
	if err != nil {
		return nil, err
	}
	return tls.Dial("tcp", prox.TargetHost(), tlsConfig)
}

type listener struct {
	ln net.Listener
}

func (ln *listener) IsListening() bool {
	if ln.ln != nil {
		return true
	}
	return false
}

func (ln *listener) Listen(prox *proxy.Proxy) error {
	tlsConfig, err := ParseOptions(prox)
	if err != nil {
		return err
	}

	tlsListener, err := tls.Listen("tcp", prox.TargetHost(), tlsConfig)
	if err != nil {
		return err
	}

	ln.ln = tlsListener

	return nil
}

func (ln *listener) Accept() (net.Conn, error) {
	return ln.ln.Accept()
}

func (ln *listener) Close() error {
	return ln.ln.Close()
}

func init() {
	proxy.Registry.Add(proxy.Proxy{
		Scheme:      "tls",
		Description: "dial to a tls host",
		Examples: []string{
			"$ gcat tls://google.de:443 -",
		},
		StringOptions: StringOptions,
		BoolOptions:   BoolOptions,
		Dialer:        &dialer{},
	})
	proxy.Registry.Add(proxy.Proxy{
		Scheme:      "tls-listen",
		Description: "spawn a tls listener",
		Examples: []string{
			"$ gcat tls-listen://127.0.0.1:1234 -",
		},
		StringOptions: StringOptions,
		BoolOptions:   BoolOptions,
		Listener:      &listener{},
	})
}
