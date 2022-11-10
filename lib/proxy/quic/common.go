package quic

import (
	"crypto/tls"
	"net"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/rumpelsepp/gcat/lib/proxy"
	gtls "github.com/rumpelsepp/gcat/lib/proxy/tls"
)

var (
	// Shared with the tls proxy.
	boolOptions = []proxy.ProxyOption[bool]{
		{
			Name:        "enable_datagrams",
			Description: "use unreliable datagrams (RFC9221)",
			Default:     false,
		},
	}
	intOptions = []proxy.ProxyOption[int]{
		{
			Name:        "keepalive_period",
			Description: "keepalive interval in seconds",
		},
	}
)

func parseOptions(prox *proxy.Proxy) (*tls.Config, *quic.Config, error) {
	tlsConfig, err := gtls.ParseOptions(prox)
	if err != nil {
		return nil, nil, err
	}

	quicConfig := &quic.Config{
		EnableDatagrams: prox.GetBoolOption("enable_datagrams"),
		KeepAlivePeriod: time.Duration(prox.GetIntOption("keepalive_period", 10)) * time.Second,
	}

	return tlsConfig, quicConfig, nil
}

type connWrapper struct {
	useDatagrams bool
	conn         quic.Connection
	stream       quic.Stream
}

func (w *connWrapper) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

func (w *connWrapper) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

func (w *connWrapper) Read(p []byte) (int, error) {
	if w.useDatagrams {
		dgram, err := w.conn.ReceiveMessage()
		if err != nil {
			return 0, err
		}
		n := copy(p, dgram)
		return n, nil
	}
	return w.stream.Read(p)
}

func (w *connWrapper) Write(p []byte) (int, error) {
	if w.useDatagrams {
		if err := w.conn.SendMessage(p); err != nil {
			return 0, err
		}
		return len(p), nil
	}
	return w.stream.Write(p)
}

func (w *connWrapper) Close() error {
	if w.stream != nil {
		if err := w.stream.Close(); err != nil {
			return err
		}
	}
	return w.conn.CloseWithError(1, "connection closed")
}

func (w *connWrapper) SetDeadline(t time.Time) error {
	return w.stream.SetDeadline(t)
}

func (w *connWrapper) SetReadDeadline(t time.Time) error {
	return w.stream.SetReadDeadline(t)
}

func (w *connWrapper) SetWriteDeadline(t time.Time) error {
	return w.stream.SetWriteDeadline(t)
}
