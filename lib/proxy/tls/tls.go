package gtls

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/rumpelsepp/gcat/lib/proxy"
)

type dialer struct{}

func (d *dialer) Dial(ctx context.Context, desc *proxy.ProxyDescription) (net.Conn, error) {
	tlsConfig, err := ParseOptions(desc)
	if err != nil {
		return nil, err
	}

	dialer := net.Dialer{}
	tcpConn, err := dialer.DialContext(ctx, "tcp", desc.TargetHost())
	if err != nil {
		return nil, err
	}

	tlsConn := tls.Client(tcpConn, tlsConfig)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		return nil, err
	}

	return tlsConn, nil
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

func (ln *listener) Listen(prox *proxy.ProxyDescription) error {
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
	proxy.Registry.Add(proxy.ProxyDescription{
		Scheme:      "tls",
		Description: "dial to a tls host",
		Examples: []string{
			"$ gcat tls://google.de:443 -",
		},
		StringOptions: StringOptions,
		BoolOptions:   BoolOptions,
		Dialer:        &dialer{},
	})
	proxy.Registry.Add(proxy.ProxyDescription{
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
