package webtransport

import (
	"context"
	"fmt"
	"net"

	"github.com/marten-seemann/webtransport-go"
	"github.com/rumpelsepp/gcat/lib/proxy"
)

type streamWrapper struct {
	webtransport.Stream
	webtransport.Session
}

func (s *streamWrapper) Close() error {
	if err := s.Stream.Close(); err != nil {
		return err
	}
	return s.Session.Close()
}

func (s *streamWrapper) LocalAddr() net.Addr {
	return s.Session.LocalAddr()
}

type Dialer struct{}

func (d *Dialer) Dial(prox *proxy.Proxy) (net.Conn, error) {
	var (
		dialer webtransport.Dialer
		url    = fmt.Sprintf("https://%s/%s", net.JoinHostPort(prox.GetStringOption("Hostname"), prox.GetStringOption("Port")), prox.GetStringOption("Path"))
	)
	_, session, err := dialer.Dial(context.Background(), url, nil)
	if err != nil {
		return nil, err
	}

	stream, err := session.OpenStream()
	if err != nil {
		return nil, err
	}

	return &streamWrapper{
		Session: *session,
		Stream:  stream,
	}, nil
}

func init() {
	proxy.Registry.Add(proxy.Proxy{
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
