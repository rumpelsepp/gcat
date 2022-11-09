package proxy

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"text/template"

	markdown "github.com/MichaelMure/go-term-markdown"
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
	Dial(proxy *Proxy) (net.Conn, error)
}

type ProxyListener interface {
	IsListening() bool
	Listen(proxy *Proxy) error
	Accept() (net.Conn, error)
	Close() error
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

func (a *ProxyAddr) GetStringOption(key, fallback string) string {
	// URL elements not in querystring.
	switch key {
	case "Host":
		if a.Host == "" {
			return fallback
		}
		return a.Host
	case "Hostname":
		if a.Hostname() == "" {
			return fallback
		}
		return a.Hostname()
	case "Port":
		if a.Port() == "" {
			return fallback
		}
		return a.Port()
	case "Path":
		if a.Path == "" {
			return fallback
		}
		return a.Path
	}

	qs := a.URL.Query()

	if qs.Has(key) {
		return qs.Get(key)
	}
	return fallback
}

func (a *ProxyAddr) GetBoolOption(key string, fallback bool) (bool, error) {
	qs := a.URL.Query()

	if qs.Has(key) {
		return strconv.ParseBool(qs.Get(key))
	}
	return fallback, nil
}

func (a *ProxyAddr) GetIntOption(key string, base int, fallback int) (int, error) {
	qs := a.URL.Query()

	if qs.Has(key) {
		v, err := strconv.ParseInt(qs.Get(key), 32, base)
		return int(v), err
	}
	return fallback, nil
}

func (a *ProxyAddr) String() string {
	return a.URL.String()
}

type ProxyOptionType interface {
	string | bool | int
}

type ProxyOption[T ProxyOptionType] struct {
	Name        string
	Description string
	Default     T
}

type Proxy struct {
	Scheme           ProxyScheme
	Description      string
	Examples         []string
	SupportsMultiple bool
	SupportsStreams  bool

	StringOptions []ProxyOption[string]
	BoolOptions   []ProxyOption[bool]
	IntOptions    []ProxyOption[int]

	Listener ProxyListener
	Dialer   ProxyDialer
	Conn     net.Conn

	addr *ProxyAddr
}

func (p *Proxy) Instantiate(addr *ProxyAddr) *Proxy {
	if addr.ProxyScheme() != p.Scheme {
		panic(fmt.Sprintf("wrong scheme %s; expected %s", addr.ProxyScheme(), p.Scheme))
	}
	p.addr = addr

	return p
}

func (p *Proxy) Target() *ProxyAddr {
	if p.addr == nil {
		panic("BUG: proxy not instantiated")
	}

	return p.addr
}

func (ep *Proxy) Help() string {
	var (
		builder strings.Builder
		tpl     = template.Must(template.New("help").Parse(`# Proxy Module
## Scheme

` + "`" + `{{ .Scheme }}` + "`" + `
		
## Description

{{ .Description }}
		
## Params

* SupportsMultipleConnections: ` + "`" + `{{ .SupportsMultiple }}` + "`" + `
* SupportsStreams: ` + "`" + `{{ .SupportsStreams }}` + "`" + `

## String Options
{{ if .StringOptions }}
{{ range .StringOptions }}
  * ` + "`" + `{{ .Name }}` + "`" + `{{if .Default}} [default: ` + "`" + `{{ .Default }}` + "`" + `]{{end}}: {{ .Description }}{{end}}
{{ else }}
no arguments
{{end}}
## Int Options
{{ if .IntOptions }}
{{ range .IntOptions }}
  * ` + "`" + `{{ .Name }}` + "`" + `{{if .Default}} [default: ` + "`" + `{{ .Default }}` + "`" + `]{{end}}: {{ .Description }}{{end}}
{{ else }}
no arguments
{{end}}
## Bool Options
{{ if .BoolOptions }}
{{ range .BoolOptions }}
  * ` + "`" + `{{ .Name }}` + "`" + `{{if .Default}} [default: ` + "`" + `{{ .Default }}` + "`" + `]{{end}}: {{ .Description }}{{end}}
{{ else }}
no arguments
{{end}}
{{ if .Examples }}## Examples
{{ range .Examples }}
    {{ . }}{{end}}
{{end}}
`))
	)

	if err := tpl.Execute(&builder, *ep); err != nil {
		panic(err)
	}

	return string(bytes.TrimSpace(markdown.Render(builder.String(), 80, 2)))
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
		return p.Dialer.Dial(p)

	case ProxyKindListener:
		if !p.Listener.IsListening() {
			if err := p.Listener.Listen(p); err != nil {
				return nil, err
			}
		}
		return p.Listener.Accept()

	case ProxyKindReadWriteCloser:
		return p.Conn, nil
	}

	panic("BUG: invalid proxy")
}

func (p *Proxy) GetStringOption(key string) string {
	var (
		found    = false
		fallback string
	)

	for _, opt := range p.StringOptions {
		if key == opt.Name {
			fallback = opt.Default
			found = true
			break
		}
	}
	if found == false {
		panic(fmt.Sprintf("BUG: unknown option: %s", key))
	}

	return p.addr.GetStringOption(key, fallback)
}

func (p *Proxy) GetBoolOption(key string) bool {
	var (
		found    = false
		fallback bool
	)

	for _, opt := range p.BoolOptions {
		if key == opt.Name {
			fallback = opt.Default
			found = true
			break
		}
	}
	if found == false {
		panic(fmt.Sprintf("BUG: unknown option: %s", key))
	}

	val, err := p.addr.GetBoolOption(key, fallback)
	if err != nil {
		panic(err)
	}
	return val
}

func (p *Proxy) GetIntOption(key string, base int) int {
	var (
		found    = false
		fallback int
	)

	for _, opt := range p.IntOptions {
		if key == opt.Name {
			fallback = opt.Default
			found = true
			break
		}
	}
	if found == false {
		panic(fmt.Sprintf("BUG: unknown option: %s", key))
	}

	val, err := p.addr.GetIntOption(key, base, fallback)
	if err != nil {
		panic(err)
	}
	return val
}

type ProxyRegistry struct {
	data map[ProxyScheme]Proxy
}

func (r ProxyRegistry) Keys() []ProxyScheme {
	return maps.Keys(r.data)
}

func (r ProxyRegistry) Values() []Proxy {
	return maps.Values(r.data)
}

func (r ProxyRegistry) Get(key ProxyScheme) (Proxy, error) {
	if v, ok := r.data[key]; ok {
		return v, nil
	}
	return Proxy{}, fmt.Errorf("no such proxy: %s", key)
}

func (r *ProxyRegistry) Add(ep Proxy) {
	if _, ok := r.data[ep.Scheme]; ok {
		panic(fmt.Sprintf("proxy with scheme %s already registered", ep.Scheme))
	}
	r.data[ep.Scheme] = ep
}

func (r *ProxyRegistry) CreateProxyInstance(addr *ProxyAddr) (*Proxy, error) {
	p, err := r.Get(addr.ProxyScheme())
	if err != nil {
		return nil, err
	}

	return p.Instantiate(addr), nil
}

var Registry = ProxyRegistry{data: make(map[ProxyScheme]Proxy)}
