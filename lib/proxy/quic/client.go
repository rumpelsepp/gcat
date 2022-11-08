package quic

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/lucas-clemente/quic-go"
	"github.com/rumpelsepp/gcat/lib/proxy"
)

type ProxyQuic struct {
	Address    string
	tlsConfig  *tls.Config
	quicConfig *quic.Config
}

func (p *ProxyQuic) Dial() (net.Conn, error) {
	var (
		stream quic.Stream
		err    error
	)

	conn, err := quic.DialAddr(p.Address, p.tlsConfig, p.quicConfig)
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

func CreateQUICProxy(addr *proxy.ProxyAddr) (*proxy.Proxy, error) {
	tlsConfig, quicConfig, err := parseOptions(addr)
	if err != nil {
		return nil, err
	}
	return proxy.CreateProxyFromDialer(
		&ProxyQuic{
			Address:    addr.Host,
			tlsConfig:  tlsConfig,
			quicConfig: quicConfig,
		}), nil
}

func init() {
	proxy.Registry.Add(proxy.Proxy{
		Scheme: "quic",
		Create: CreateQUICProxy,
		Help: proxy.ProxyHelp{
			Description: "connect to a quic host:port",
			Examples: []string{
				"$ gcat proxy quic://localhost:1234 -",
			},
			Args: helpArgs,
		},
	})
}
