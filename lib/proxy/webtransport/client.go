package webtransport

import (
	"context"
	"fmt"
	"net"

	"github.com/quic-go/webtransport-go"
	"github.com/rumpelsepp/gcat/lib/proxy"
)

type streamWrapper struct {
	webtransport.Stream
	*webtransport.Session
}

func (s *streamWrapper) Close() error {
	if err := s.Stream.Close(); err != nil {
		return err
	}
	return s.Session.CloseWithError(1, "sessions closed")
}

func (s *streamWrapper) LocalAddr() net.Addr {
	return s.Session.LocalAddr()
}

type Dialer struct{}

func (d *Dialer) Dial(ctx context.Context, desc *proxy.ProxyDescription) (net.Conn, error) {
	var (
		dialer webtransport.Dialer
		url    = fmt.Sprintf("https://%s/%s", desc.TargetHost(), desc.GetStringOption("Path"))
	)
	_, session, err := dialer.Dial(ctx, url, nil)
	if err != nil {
		return nil, err
	}

	stream, err := session.OpenStream()
	if err != nil {
		return nil, err
	}

	return &streamWrapper{
		Session: session,
		Stream:  stream,
	}, nil
}

func init() {
	proxy.Registry.Add(proxy.ProxyDescription{
		Scheme:           "wt",
		Description:      "dial to a webtransport endpoint",
		SupportsMultiple: true,
		Dialer:           &Dialer{},
		Examples: []string{
			"$ gcat proxy wt://localhost:1234/wt -",
		},
		StringOptions: []proxy.ProxyOption[string]{
			{
				Name:        "Hostname",
				Description: "target address",
			},
			{
				Name:        "Port",
				Description: "target port",
			},
			{
				Name:        "Path",
				Description: "http uri path",
			},
		},
	})
}
