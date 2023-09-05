package proxy

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"text/template"

	markdown "github.com/MichaelMure/go-term-markdown"
)

type ProxyDialer interface {
	Dial(ctx context.Context, desc *ProxyDescription) (net.Conn, error)
}

type ProxyListener interface {
	IsListening() bool
	Listen(desc *ProxyDescription) error
	Accept() (net.Conn, error)
	Close() error
}

type ProxyScheme string

func (s ProxyScheme) IsListener() bool {
	if strings.HasSuffix(string(s), "-listen") {
		return true
	}
	return false
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

type ProxyOptionType interface {
	string | bool | int
}

type ProxyOption[T ProxyOptionType] struct {
	Name        string
	Description string
	Default     T
}

type ProxyDescription struct {
	Scheme           ProxyScheme
	Description      string
	Examples         []string
	SupportsMultiple bool
	SupportsStreams  bool

	StringOptions []ProxyOption[string]
	BoolOptions   []ProxyOption[bool]
	IntOptions    []ProxyOption[int]

	Dialer   ProxyDialer
	Listener ProxyListener

	addr *ProxyAddr
}

func (p *ProxyDescription) IsListener() bool {
	return p.Target().ProxyScheme().IsListener()
}

func (p *ProxyDescription) SetAddr(addr *ProxyAddr) *ProxyDescription {
	if addr.ProxyScheme() != p.Scheme {
		panic(fmt.Sprintf("wrong scheme %s; expected %s", addr.ProxyScheme(), p.Scheme))
	}
	p.addr = addr

	return p
}

func (p *ProxyDescription) Target() *ProxyAddr {
	if p.addr == nil {
		panic("BUG: proxy not instantiated")
	}

	return p.addr
}

func (ep *ProxyDescription) Help() string {
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
func (p *ProxyDescription) Connect(ctx context.Context) (net.Conn, error) {
	// If both p.Dialer and p.Listener are defined, then p.Listener
	// is ignored.
	if dialer := p.Dialer; dialer != nil {
		return dialer.Dial(ctx, p)
	}

	if ln := p.Listener; ln != nil {
		if !ln.IsListening() {
			if err := ln.Listen(p); err != nil {
				return nil, err
			}
		}
		return ln.Accept()
	}

	panic("BUG: invalid proxy")
}

func (p *ProxyDescription) TargetHost() string {
	return net.JoinHostPort(p.GetStringOption("Hostname"), p.GetStringOption("Port"))
}

func (p *ProxyDescription) GetStringOption(key string) string {
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

func (p *ProxyDescription) GetBoolOption(key string) bool {
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

func (p *ProxyDescription) GetIntOption(key string, base int) int {
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
