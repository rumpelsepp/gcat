package quic

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/lucas-clemente/quic-go"
	"github.com/rumpelsepp/gcat/lib/proxy"
)

type ProxyQuicListener struct {
	Address    string
	tlsConfig  *tls.Config
	quicConfig *quic.Config
	listener   quic.Listener
}

func (p *ProxyQuicListener) IsListening() bool {
	if p.listener == nil {
		return false
	}
	return true
}

func (p *ProxyQuicListener) Listen() error {
	if p.IsListening() {
		return proxy.ErrProxyBusy
	}

	packetConn, err := net.ListenPacket("udp", p.Address)
	if err != nil {
		return err
	}

	quicLn, err := quic.Listen(packetConn, p.tlsConfig, p.quicConfig)
	if err != nil {
		return err
	}
	p.listener = quicLn
	return nil
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

func CreateQUICListenerProxy(addr *proxy.ProxyAddr) (*proxy.Proxy, error) {
	tlsConfig, quicConfig, err := parseOptions(addr)
	if err != nil {
		return nil, err
	}
	return proxy.CreateProxyFromListener(
		&ProxyQuicListener{
			Address:    addr.Host,
			tlsConfig:  tlsConfig,
			quicConfig: quicConfig,
		}), nil
}

func init() {
	proxy.Registry.Add(proxy.ProxyEntryPoint{
		Scheme: "quic-listen",
		Create: CreateQUICListenerProxy,
		Help: proxy.ProxyHelp{
			Description: "spacn quic server",
			Examples: []string{
				"$ gcat proxy quic-listen://localhost:1234 -",
			},
			Args: helpArgs,
		},
	})
}
