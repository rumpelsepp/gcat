package proxy

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"text/template"
	"time"

	"golang.org/x/exp/maps"
)

var (
	ErrProxyNotSupported   = errors.New("operation not supported")
	ErrProxyBusy           = errors.New("proxy is busy")
	ErrProxyNotInitialized = errors.New("proxy is not initialized")
	ErrNotImplemented      = errors.New("method not implemented")
)

type ProxyScheme string

func (s ProxyScheme) IsListener() bool {
	if strings.Contains(string(s), "listen") {
		return true
	}
	return false
}

type ProxyKind string

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

func CreateProxyFromConn(conn net.Conn) *Proxy {
	return &Proxy{Conn: conn}
}

func CreateProxyFromDialer(d ProxyDialer) *Proxy {
	return &Proxy{Dialer: d}
}

func CreateProxyFromListener(ln ProxyListener) *Proxy {
	return &Proxy{Listener: ln}
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

type ProxyHelpArg struct {
	Name        string
	Type        string
	Explanation string
	Default     string
}

type ProxyHelp struct {
	Description string
	Examples    []string
	Args        []ProxyHelpArg
}

type ProxyEntryPoint struct {
	Create func(addr *ProxyAddr) (*Proxy, error)
	Scheme ProxyScheme
	Help   ProxyHelp
}

func (ep *ProxyEntryPoint) String() string {
	var (
		builder strings.Builder
		tpl     = template.Must(template.New("help").Parse(`# Scheme

{{ .Scheme }}

## Description

{{ .Help.Description }}
{{ if .Help.Args }}
## Arguments
{{ range .Help.Args }}
  * {{ .Name }} ({{ .Type }}){{if .Default}} [default: {{ .Default }}]{{end}}: {{ .Explanation }}{{end}}
{{ else }}
no arguments
{{end}}
{{ if .Help.Examples }}## Examples
{{ range .Help.Examples }}
  * {{ . }}{{end}}
{{end}}
`))
	)

	if err := tpl.Execute(&builder, *ep); err != nil {
		panic(err)
	}

	return strings.TrimSpace(builder.String())
}

type ProxyRegistry struct {
	data map[ProxyScheme]ProxyEntryPoint
}

func (r ProxyRegistry) Keys() []ProxyScheme {
	return maps.Keys(r.data)
}

func (r ProxyRegistry) Values() []ProxyEntryPoint {
	return maps.Values(r.data)
}

func (r ProxyRegistry) Get(key ProxyScheme) (ProxyEntryPoint, error) {
	if v, ok := r.data[key]; ok {
		return v, nil
	}
	return ProxyEntryPoint{}, fmt.Errorf("no such proxy: %s", key)
}

func (r *ProxyRegistry) Add(ep ProxyEntryPoint) {
	if _, ok := r.data[ep.Scheme]; ok {
		panic(fmt.Sprintf("proxy with scheme %s already registered", ep.Scheme))
	}
	r.data[ep.Scheme] = ep
}

func (r *ProxyRegistry) Create(addr *ProxyAddr) (*Proxy, error) {
	ep, ok := r.data[addr.ProxyScheme()]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProxyNotSupported, addr.ProxyScheme())
	}
	return ep.Create(addr)
}

var Registry = ProxyRegistry{data: make(map[ProxyScheme]ProxyEntryPoint)}
