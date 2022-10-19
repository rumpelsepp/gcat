package quic

import (
	"crypto/tls"
	"io"
	"net"
	"os"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/rumpelsepp/gcat/lib/helper"
	"github.com/rumpelsepp/gcat/lib/proxy"
)

var helpArgs = []proxy.ProxyHelpArg{
	{
		Name:        "Host",
		Type:        "string",
		Explanation: "target ip address",
	},
	{
		Name:        "Port",
		Type:        "int",
		Explanation: "target port",
	},
	{
		Name:        "use_datagrams",
		Type:        "bool",
		Explanation: "use unreliable datagrams (RFC9221)",
		Default:     "false",
	},
	{
		Name:        "next_proto",
		Type:        "string",
		Explanation: "value to use in the ALPN field (https://github.com/quicwg/base-drafts/wiki/ALPN-IDs-used-with-QUIC)",
		Default:     "quic",
	},
	{
		Name:        "key_path",
		Type:        "string",
		Explanation: "path to pem encoded private key",
	},
	{
		Name:        "cert_path",
		Type:        "string",
		Explanation: "path to pem encoded certificte",
	},
	{
		Name:        "keylog_file",
		Type:        "string",
		Explanation: "path to sslkeylog file (for debugging)",
	},
	{
		Name:        "keepalive_period",
		Type:        "int",
		Explanation: "keepalive interval in seconds",
	},
}

func parseOptions(addr *proxy.ProxyAddr) (*tls.Config, *quic.Config, error) {
	var (
		err                error
		insecureSkipVerify = true
		nextProto          = addr.GetStringOption("next_proto", "quic")
		keyPath            = addr.GetStringOption("key_path", "")
		certPath           = addr.GetStringOption("cert_path", "")
		keylogFile         = addr.GetStringOption("keylog_file", "")
	)

	enableDatagrams, err := addr.GetBoolOption("use_datagrams", false)
	if err != nil {
		return nil, nil, err
	}

	keepAlivePeriod, err := addr.GetIntOption("keep_alive_period", 10, 10)
	if err != nil {
		return nil, nil, err
	}

	var cert tls.Certificate
	if keyPath == "" || certPath == "" {
		cert, err = helper.GenTLSCertificate()
	} else {
		cert, err = tls.LoadX509KeyPair(certPath, keyPath)
	}
	if err != nil {
		return nil, nil, err
	}

	if keylogFile == "" {
		keylogFile = os.Getenv("SSLKEYLOGFILE")
	}

	var keylogWriter io.Writer = nil
	if keylogFile != "" {
		f, err := os.Create(keylogFile)
		if err != nil {
			return nil, nil, err
		}
		keylogWriter = f
	}

	var (
		tlsConfig = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: insecureSkipVerify,
			NextProtos:         []string{nextProto},
			KeyLogWriter:       keylogWriter,
		}
		quicConfig = &quic.Config{
			EnableDatagrams: enableDatagrams,
			KeepAlivePeriod: time.Duration(keepAlivePeriod) * time.Second,
		}
	)

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
