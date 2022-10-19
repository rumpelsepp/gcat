package helper

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
)

func GenKeyPEM() ([]byte, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, err
	}

	privPEM := pem.EncodeToMemory(&pem.Block{Type: "ED25519 PRIVATE KEY", Bytes: privDER})

	return privPEM, nil
}

func GenCertificate(priv ed25519.PrivateKey) ([]byte, []byte, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("generate serial number: %s", err)
	}

	template := x509.Certificate{SerialNumber: serialNumber}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}

	privPEM := pem.EncodeToMemory(&pem.Block{Type: "ED25519 PRIVATE KEY", Bytes: privDER})

	return privPEM, certPEM, nil
}

func GenTLSCertificate() (tls.Certificate, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate private key: %w", err)
	}
	keyPEM, certPEM, err := GenCertificate(priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.X509KeyPair(certPEM, keyPEM)
}

func GenKeypairFS() (string, string, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate private key: %w", err)
	}

	privPEM, certPEM, err := GenCertificate(priv)
	if err != nil {
		return "", "", err
	}

	dir, err := os.MkdirTemp("", "gcat-*")
	if err != nil {
		return "", "", err
	}

	certpath := filepath.Join(dir, "cert.pem")
	privpath := filepath.Join(dir, "priv.pem")

	if err := os.WriteFile(certpath, certPEM, 0644); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(privpath, privPEM, 0600); err != nil {
		return "", "", err
	}

	return privpath, certpath, err
}
