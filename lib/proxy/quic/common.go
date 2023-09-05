package quic

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/quic-go/quic-go"
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

func parseOptions(prox *proxy.ProxyDescription) (*tls.Config, *quic.Config, error) {
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

type streamWrapper struct {
	conn   quic.Connection
	stream quic.Stream
}

func (w *streamWrapper) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

func (w *streamWrapper) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

func (w *streamWrapper) Read(p []byte) (int, error) {
	return w.stream.Read(p)
}

func (w *streamWrapper) Write(p []byte) (int, error) {
	return w.stream.Write(p)
}

func (w *streamWrapper) Close() error {
	if w.stream != nil {
		if err := w.stream.Close(); err != nil {
			return err
		}
	}
	return w.conn.CloseWithError(1, "connection closed")
}

func (w *streamWrapper) SetDeadline(t time.Time) error {
	return w.stream.SetDeadline(t)
}

func (w *streamWrapper) SetReadDeadline(t time.Time) error {
	return w.stream.SetReadDeadline(t)
}

func (w *streamWrapper) SetWriteDeadline(t time.Time) error {
	return w.stream.SetWriteDeadline(t)
}

type datagramWrapper struct {
	ctx  context.Context
	conn quic.Connection
}

func (w *datagramWrapper) Read(p []byte) (int, error) {
	dgram, err := w.conn.ReceiveMessage(w.ctx)
	if err != nil {
		return 0, err
	}
	n := copy(p, dgram)
	return n, nil
}

func (w *datagramWrapper) Write(p []byte) (int, error) {
	if err := w.conn.SendMessage(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *datagramWrapper) SetDeadline(t time.Time) error {
	return proxy.ErrNotSupported
}

func (w *datagramWrapper) SetReadDeadline(t time.Time) error {
	return proxy.ErrNotSupported
}

func (w *datagramWrapper) SetWriteDeadline(t time.Time) error {
	return proxy.ErrNotSupported
}

func (w *datagramWrapper) Close() error {
	return w.conn.CloseWithError(1, "connection closed")
}

func (w *datagramWrapper) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

func (w *datagramWrapper) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}
