package quic

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/lucas-clemente/quic-go"
	"github.com/rumpelsepp/gcat/lib/proxy"
	gtls "github.com/rumpelsepp/gcat/lib/proxy/tls"
)

type ProxyQuic struct {
	tlsConfig  *tls.Config
	quicConfig *quic.Config
}

func (p *ProxyQuic) Dial(prox *proxy.Proxy) (net.Conn, error) {
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

	conn, err := quic.DialAddr(net.JoinHostPort(prox.GetStringOption("Hostname"), prox.GetStringOption("Port")), p.tlsConfig, p.quicConfig)
	if err != nil {
		return nil, err
	}

	if !p.quicConfig.EnableDatagrams {
		stream, err = conn.OpenStreamSync(context.Background())
		if err != nil {
			return nil, err
		}
	}
	return &connWrapper{
		useDatagrams: p.quicConfig.EnableDatagrams,
		conn:         conn,
		stream:       stream,
	}, nil
}

func init() {
	proxy.Registry.Add(proxy.Proxy{
		Scheme:      "quic",
		Dialer:      &ProxyQuic{},
		Description: "connect to a quic host:port",
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
