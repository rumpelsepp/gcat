package gtls

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/rumpelsepp/gcat/lib/helper"
	"github.com/rumpelsepp/gcat/lib/proxy"
)

var (
	StringOptions = []proxy.ProxyOption[string]{
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
			Description: "value to use in the ALPN field",
			Default:     "quic",
		},
	}
	BoolOptions = []proxy.ProxyOption[bool]{
		{
			Name:        "skip_verify",
			Description: "skip certificate verification",
			Default:     false,
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

func ParseOptions(prox *proxy.Proxy) (*tls.Config, error) {
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
	if prox.IsListener() {
		if keyPath == "" || certPath == "" {
			cert, err = helper.GenTLSCertificate()
			digest := sha256.Sum256(cert.Certificate[0])
			fmt.Printf("generated cert: %s\n", hex.EncodeToString(digest[:]))
		} else {
			cert, err = tls.LoadX509KeyPair(certPath, keyPath)
		}
		if err != nil {
			return nil, err
		}
	}

	if keylogFile == "" {
		keylogFile = os.Getenv("SSLKEYLOGFILE")
	}

	var keylogWriter io.Writer = nil
	if keylogFile != "" {
		f, err := os.Create(keylogFile)
		if err != nil {
			return nil, err
		}
		keylogWriter = f
	}

	if fingerprint != "" {
		skipVerify = true
		verifier, err = makeVerifier(fingerprint)
		if err != nil {
			return nil, err
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
	)

	return tlsConfig, nil
}
