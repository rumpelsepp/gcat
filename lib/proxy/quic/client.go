package quic

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/quic-go/quic-go"
	"github.com/rumpelsepp/gcat/lib/proxy"
	gtls "github.com/rumpelsepp/gcat/lib/proxy/tls"
)

type QUICDialer struct {
	tlsConfig  *tls.Config
	quicConfig *quic.Config
}

func (p *QUICDialer) Dial(ctx context.Context, prox *proxy.ProxyDescription) (net.Conn, error) {
	var (
		stream quic.Stream
		err    error
	)

	if p.quicConfig == nil || p.tlsConfig == nil {
		tlsConfig, quicConfig, err := parseOptions(prox)
		if err != nil {
			return nil, err
		}

		p.tlsConfig = tlsConfig
		p.quicConfig = quicConfig
	}

	conn, err := quic.DialAddr(ctx, prox.TargetHost(), p.tlsConfig, p.quicConfig)
	if err != nil {
		return nil, err
	}

	if !p.quicConfig.EnableDatagrams {
		stream, err = conn.OpenStreamSync(ctx)
		if err != nil {
			return nil, err
		}
	}
	return &streamWrapper{
		conn:   conn,
		stream: stream,
	}, nil
}

func init() {
	proxy.Registry.Add(proxy.ProxyDescription{
		Scheme:      "quic",
		Dialer:      &QUICDialer{},
		Description: "connect to a quic host:port and open one stream",
		Examples: []string{
			"$ gcat proxy quic://localhost:1234 -",
		},
		SupportsMultiple: true,
		StringOptions:    gtls.StringOptions,
		IntOptions:       intOptions,
		BoolOptions:      append(gtls.BoolOptions, boolOptions...),
	},
	)
}
