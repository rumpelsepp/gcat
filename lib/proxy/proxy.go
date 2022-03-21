package proxy

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

var (
	ErrProxyNotSupported   = errors.New("operation not supported")
	ErrProxyBusy           = errors.New("proxy is busy")
	ErrProxyNotInitialized = errors.New("proxy is not initialized")
	ErrNotImplemented      = errors.New("method not implemented")
)

type ProxyScheme string

const (
	ProxySchemeExec      ProxyScheme = "exec"
	ProxySchemeSTDIO                 = "stdio"
	ProxySchemeTCP                   = "tcp"
	ProxySchemeTCPListen             = "tcp-listen"
	ProxySchemeTLS                   = "tls"
	ProxySchemeTLSListen             = "tls-listen"
	ProxySchemeTUN                   = "tun"
	ProxySchemeWS                    = "ws"
	ProxySchemeWSListen              = "ws-listen"
)

type ProxyKind string

func (s ProxyScheme) IsListener() bool {
	if strings.Contains(string(s), "listen") {
		return true
	}
	return false
}

const (
	ProxyKindListener        ProxyKind = "listener"
	ProxyKindDialer                    = "dialer"
	ProxyKindReadWriteCloser           = "readwritecloser"
)

type ProxyDialer interface {
	Dial() (net.Conn, error)
}

type ProxyListener interface {
	IsListening() bool
	Listen() error
	Accept() (net.Conn, error)
}

type Proxy struct {
	Listener ProxyListener
	Dialer   ProxyDialer
	Conn     net.Conn
}

func (p *Proxy) Kind() ProxyKind {
	if p.Listener != nil {
		return ProxyKindListener
	}
	if p.Dialer != nil {
		return ProxyKindDialer
	}
	if p.Conn != nil {
		return ProxyKindReadWriteCloser
	}

	panic("BUG: invalid proxy")
}

func (p *Proxy) Connect() (net.Conn, error) {
	switch p.Kind() {

	case ProxyKindDialer:
		return p.Dialer.Dial()

	case ProxyKindListener:
		if !p.Listener.IsListening() {
			if err := p.Listener.Listen(); err != nil {
				return nil, err
			}
		}
		return p.Listener.Accept()

	case ProxyKindReadWriteCloser:
		return p.Conn, nil
	}

	panic("BUG: invalid proxy")
}

func fixupURL(rawURL string) string {
	if rawURL == "-" {
		return "stdio:"
	}

	if strings.HasPrefix(rawURL, "exec:") && !strings.Contains(rawURL, "?") {
		cmdEncoded := url.QueryEscape(strings.TrimPrefix(rawURL, "exec:"))
		return fmt.Sprintf("exec:?cmd=%s", cmdEncoded)
	}

	return rawURL
}

type ProxyAddr struct {
	url.URL
}

func ParseAddr(raw string) (*ProxyAddr, error) {
	u, err := url.Parse(fixupURL(raw))
	if err != nil {
		return nil, err
	}
	return &ProxyAddr{*u}, nil
}

func (a *ProxyAddr) ProxyScheme() ProxyScheme {
	return ProxyScheme(a.Scheme)
}

func (a *ProxyAddr) Network() string {
	return string(a.ProxyScheme())
}

func (a *ProxyAddr) String() string {
	return a.URL.String()
}

type BaseConn struct {
	LocalAddress  *ProxyAddr
	RemoteAddress *ProxyAddr
}

func (c *BaseConn) Read(b []byte) (int, error) {
	return 0, ErrNotImplemented
}

func (c *BaseConn) Write(b []byte) (int, error) {
	return 0, ErrNotImplemented
}

func (c *BaseConn) Close() error {
	return ErrNotImplemented
}

func (c *BaseConn) LocalAddr() net.Addr {
	return c.LocalAddress
}

func (c *BaseConn) RemoteAddr() net.Addr {
	return c.RemoteAddress
}

func (c *BaseConn) SetDeadline(t time.Time) error {
	return ErrNotImplemented
}

func (c *BaseConn) SetReadDeadline(t time.Time) error {
	return ErrNotImplemented
}

func (c *BaseConn) SetWriteDeadline(t time.Time) error {
	return ErrNotImplemented
}

type ProxyEntryPoint struct {
	Create    func(addr *ProxyAddr) (*Proxy, error)
	Scheme    ProxyScheme
	ShortHelp string
}

var ProxyRegistry = make(map[ProxyScheme]ProxyEntryPoint)
