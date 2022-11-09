package quic

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/rumpelsepp/gcat/lib/helper"
	"github.com/rumpelsepp/gcat/lib/proxy"
)

var (
	stringOptions = []proxy.ProxyOption[string]{
		{
			Name:        "Hostname",
			Description: "target ip address",
		},
		{
			Name:        "Port",
			Description: "target port",
		},
		{
			Name:        "key_path",
			Description: "path to pem encoded private key",
		},
		{
			Name:        "cert_path",
			Description: "path to pem encoded certificte",
		},
		{
			Name:        "keylog_file",
			Description: "path to sslkeylog file (for debugging)",
		},
		{
			Name:        "fingerprint",
			Description: "pin to this publickey fingerprint (SHA256)",
		},
		{
			Name:        "next_proto",
			Description: "value to use in the ALPN field (https://github.com/quicwg/base-drafts/wiki/ALPN-IDs-used-with-QUIC)",
			Default:     "quic",
		},
	}
	boolOptions = []proxy.ProxyOption[bool]{
		{
			Name:        "skip_verify",
			Description: "skip certificate verification",
			Default:     false,
		},
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

func makeVerifier(fingerprint string) (func([][]byte, [][]*x509.Certificate) error, error) {
	expectedDigest, err := hex.DecodeString(fingerprint)
	if err != nil {
		return nil, err
	}
	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		for _, rawCert := range rawCerts {
			digest := sha256.Sum256(rawCert)
			if bytes.Equal(expectedDigest, digest[:]) {
				return nil
			}
		}
		return fmt.Errorf("peer is not trusted")
	}, nil
}

func parseOptions(prox *proxy.Proxy) (*tls.Config, *quic.Config, error) {
	var (
		err         error
		verifier    func([][]byte, [][]*x509.Certificate) error
		keyPath     = prox.GetStringOption("key_path")
		certPath    = prox.GetStringOption("cert_path")
		keylogFile  = prox.GetStringOption("keylog_file")
		fingerprint = prox.GetStringOption("fingerprint")
		skipVerify  = prox.GetBoolOption("skip_verify")
		clientAuth  = tls.NoClientCert
	)

	var cert tls.Certificate
	if keyPath == "" || certPath == "" {
		cert, err = helper.GenTLSCertificate()
		digest := sha256.Sum256(cert.Certificate[0])
		fmt.Printf("generated cert: %s\n", hex.EncodeToString(digest[:]))
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

	if fingerprint != "" {
		skipVerify = true
		verifier, err = makeVerifier(fingerprint)
		if err != nil {
			return nil, nil, err
		}
		clientAuth = tls.RequireAnyClientCert
	}

	var (
		tlsConfig = &tls.Config{
			Certificates:          []tls.Certificate{cert},
			InsecureSkipVerify:    skipVerify,
			NextProtos:            []string{prox.GetStringOption("next_proto")},
			KeyLogWriter:          keylogWriter,
			VerifyPeerCertificate: verifier,
			ClientAuth:            clientAuth,
		}
		quicConfig = &quic.Config{
			EnableDatagrams: prox.GetBoolOption("enable_datagrams"),
			KeepAlivePeriod: time.Duration(prox.GetIntOption("keepalive_period", 10)) * time.Second,
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
