package quic

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/quic-go/quic-go"
	"github.com/rumpelsepp/gcat/lib/proxy"
	gtls "github.com/rumpelsepp/gcat/lib/proxy/tls"
)

type QUICListener struct {
	listener   *quic.Listener
	quicConfig *quic.Config
	tlsConfig  *tls.Config
}

func (p *QUICListener) IsListening() bool {
	if p.listener == nil {
		return false
	}
	return true
}

func (p *QUICListener) Listen(desc *proxy.ProxyDescription) error {
	if p.IsListening() {
		return proxy.ErrProxyBusy
	}

	tlsConfig, quicConfig, err := parseOptions(desc)
	if err != nil {
		return err
	}

	p.tlsConfig = tlsConfig
	p.quicConfig = quicConfig

	packetConn, err := net.ListenPacket("udp", desc.TargetHost())
	if err != nil {
		return err
	}

	quicLn, err := quic.Listen(packetConn, tlsConfig, quicConfig)
	fmt.Println("listening")
	if err != nil {
		return err
	}

	p.listener = quicLn

	return nil
}

func (p *QUICListener) Close() error {
	return p.listener.Close()
}

func (p *QUICListener) Accept() (net.Conn, error) {
	if !p.IsListening() {
		return nil, proxy.ErrProxyNotInitialized
	}

	ctx := context.Background()

	conn, err := p.listener.Accept(ctx)
	if err != nil {
		return nil, err
	}

	if p.quicConfig.EnableDatagrams {
		return &datagramWrapper{
			ctx:  ctx,
			conn: conn,
		}, nil
	}

	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		return nil, err
	}

	return &streamWrapper{
		conn:   conn,
		stream: stream,
	}, nil
}

func init() {
	proxy.Registry.Add(proxy.ProxyDescription{
		Scheme:      "quic-listen",
		Description: "spawn quic server",
		Examples: []string{
			"$ gcat proxy quic-listen://localhost:1234 -",
		},
		SupportsMultiple: true,
		Listener:         &QUICListener{},
		StringOptions:    gtls.StringOptions,
		IntOptions:       intOptions,
		BoolOptions:      append(gtls.BoolOptions, boolOptions...),
	})
}
