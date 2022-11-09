package quic

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/lucas-clemente/quic-go"
	"github.com/rumpelsepp/gcat/lib/proxy"
)

type ProxyQuicListener struct {
	listener   quic.Listener
	quicConfig *quic.Config
	tlsConfig  *tls.Config
}

func (p *ProxyQuicListener) IsListening() bool {
	if p.listener == nil {
		return false
	}
	return true
}

func (p *ProxyQuicListener) Listen(prox *proxy.Proxy) error {
	if p.IsListening() {
		return proxy.ErrProxyBusy
	}

	tlsConfig, quicConfig, err := parseOptions(prox)
	if err != nil {
		return err
	}

	p.tlsConfig = tlsConfig
	p.quicConfig = quicConfig

	packetConn, err := net.ListenPacket("udp", net.JoinHostPort(prox.GetStringOption("Hostname"), prox.GetStringOption("Port")))
	if err != nil {
		return err
	}

	quicLn, err := quic.Listen(packetConn, tlsConfig, quicConfig)
	if err != nil {
		return err
	}

	p.listener = quicLn

	return nil
}

func (p *ProxyQuicListener) Close() error {
	return p.listener.Close()
}

func (p *ProxyQuicListener) Accept() (net.Conn, error) {
	if !p.IsListening() {
		return nil, proxy.ErrProxyNotInitialized
	}

	ctx := context.Background()

	conn, err := p.listener.Accept(ctx)
	if err != nil {
		return nil, err
	}

	var stream quic.Stream
	if !p.quicConfig.EnableDatagrams {
		stream, err = conn.AcceptStream(ctx)
		if err != nil {
			return nil, err
		}
	}

	return &connWrapper{
		conn:         conn,
		stream:       stream,
		useDatagrams: p.quicConfig.EnableDatagrams,
	}, nil
}

func init() {
	proxy.Registry.Add(proxy.Proxy{
		Scheme:      "quic-listen",
		Description: "spacn quic server",
		Examples: []string{
			"$ gcat proxy quic-listen://localhost:1234 -",
		},
		SupportsMultiple: true,
		Listener:         &ProxyQuicListener{},
		StringOptions:    stringOptions,
		IntOptions:       intOptions,
		BoolOptions:      boolOptions,
	})
}
