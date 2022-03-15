package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"codeberg.org/rumpelsepp/gcat"
	gexec "codeberg.org/rumpelsepp/gcat/lib/proxy/exec"
	"codeberg.org/rumpelsepp/gcat/lib/proxy/tun"
	"codeberg.org/rumpelsepp/helpers"
	"github.com/spf13/cobra"
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

func (s ProxyScheme) IsListener() bool {
	if strings.Contains(string(s), "listen") {
		return true
	}
	return false
}

func createProxy(u *url.URL) (any, error) {
	switch ProxyScheme(u.Scheme) {

	case ProxySchemeExec:
		return gexec.CreateProxyExec(u)

	case ProxySchemeSTDIO:
		return gcat.NewStdioWrapper(), nil

	// TODO: implement dialer
	case ProxySchemeTCP:
		return &gcat.ProxyTCP{
			Address: u.Host,
			Network: "tcp",
		}, nil

	case ProxySchemeTCPListen:
		return &gcat.ProxyTCPListener{
			Address: u.Host,
			Network: "tcp",
		}, nil

	// TODO: implement dialer and tls config parsing
	case ProxySchemeTLS:
		return &gcat.ProxyTLS{
			Address: u.Host,
			Network: "tcp",
		}, nil

	// TODO: implement tls config parsing
	case ProxySchemeTLSListen:
		config := &tls.Config{}
		return &gcat.ProxyTLSListener{
			Address: u.Host,
			Config:  config,
			Network: "tcp",
		}, nil

	case ProxySchemeTUN:
		return tun.CreateProxyTUN(u)

	case ProxySchemeWS:
		return &gcat.ProxyWS{
			Address:   u.Host,
			KeepAlive: 20 * time.Second, // TODO: Make configurable.
			Path:      u.Path,
			Scheme:    ProxySchemeWS,
		}, nil

	case ProxySchemeWSListen:
		return &gcat.ProxyWSListener{
			Address: u.Host,
			Path:    u.Path,
		}, nil

	default:
		return nil, fmt.Errorf("%w: %s", gcat.ErrNotSupported, u)
	}
}

func connect(proxy any) (io.ReadWriteCloser, error) {
	switch p := proxy.(type) {
	case io.ReadWriteCloser:
		return p, nil

	case *gcat.StdioWrapper:
		p.Reopen()
		return p, nil

	case gcat.ProxyDialer:
		conn, err := p.Dial()
		if err != nil {
			return nil, err
		}
		return conn, nil

	case gcat.ProxyListener:
		ln, err := p.Listen()
		if err != nil {
			return nil, err
		}
		conn, err := ln.Accept()
		if err != nil {
			return nil, err
		}
		return conn, err

	default:
		panic("BUG: Wrong proxy type")
	}
}

func fixupURL(rawURL string) string {
	switch {
	case rawURL == "-":
		return "stdio:"
	case strings.HasPrefix(rawURL, "exec:") && !strings.Contains(rawURL, "?"):
		cmdEncoded := url.QueryEscape(strings.TrimPrefix(rawURL, "exec:"))
		return fmt.Sprintf("exec:?cmd=%s", cmdEncoded)
	}

	return rawURL
}

func mainLoop(left any, right any) {
	connLeft, err := connect(left)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	connRight, err := connect(right)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	_, _, err = helpers.BidirectCopy(connLeft, connRight)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type proxyCommand struct {
	state *runtimeState
}

func (c *proxyCommand) run(cmd *cobra.Command, args []string) error {
	var (
		urlLeftRaw  string
		urlRightRaw string
	)

	if len(args) == 0 || len(args) > 2 {
		return fmt.Errorf("provide one or two urls")
	}

	if len(args) == 1 {
		urlRightRaw = "stdio:"
	} else {
		urlRightRaw = fixupURL(args[1])
	}
	urlLeftRaw = fixupURL(args[0])

	urlLeft, err := url.Parse(urlLeftRaw)
	if err != nil {
		return err
	}

	urlRight, err := url.Parse(urlRightRaw)
	if err != nil {
		return err
	}

	proxyLeft, err := createProxy(urlLeft)
	if err != nil {
		return err
	}

	proxyRight, err := createProxy(urlRight)
	if err != nil {
		return err
	}

	if c.state.keepRunning {
		for {
			mainLoop(proxyLeft, proxyRight)
		}
	} else {
		mainLoop(proxyLeft, proxyRight)
	}
	return nil
}
