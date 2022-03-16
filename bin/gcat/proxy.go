package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"codeberg.org/rumpelsepp/gcat/lib/proxy"
	gexec "codeberg.org/rumpelsepp/gcat/lib/proxy/exec"
	"codeberg.org/rumpelsepp/gcat/lib/proxy/stdio"
	"codeberg.org/rumpelsepp/gcat/lib/proxy/tcp"
	gtls "codeberg.org/rumpelsepp/gcat/lib/proxy/tls"
	"codeberg.org/rumpelsepp/gcat/lib/proxy/tun"
	"codeberg.org/rumpelsepp/gcat/lib/proxy/websocket"
	"codeberg.org/rumpelsepp/helpers"
	"github.com/spf13/cobra"
)

func createProxy(u *url.URL) (any, error) {
	switch proxy.ProxyScheme(u.Scheme) {

	case proxy.ProxySchemeExec:
		return gexec.CreateProxyExec(u)

	case proxy.ProxySchemeSTDIO:
		return stdio.NewStdioWrapper(), nil

	// TODO: implement dialer
	case proxy.ProxySchemeTCP:
		return &tcp.ProxyTCP{
			Address: u.Host,
			Network: "tcp",
		}, nil

	case proxy.ProxySchemeTCPListen:
		return &tcp.ProxyTCPListener{
			Address: u.Host,
			Network: "tcp",
		}, nil

	// TODO: implement dialer and tls config parsing
	case proxy.ProxySchemeTLS:
		return &gtls.ProxyTLS{
			Address: u.Host,
			Network: "tcp",
		}, nil

	// TODO: implement tls config parsing
	case proxy.ProxySchemeTLSListen:
		config := &tls.Config{}
		return &gtls.ProxyTLSListener{
			Address: u.Host,
			Config:  config,
			Network: "tcp",
		}, nil

	case proxy.ProxySchemeTUN:
		return tun.CreateProxyTUN(u)

	case proxy.ProxySchemeWS:
		return &websocket.ProxyWS{
			Address:   u.Host,
			KeepAlive: 20 * time.Second, // TODO: Make configurable.
			Path:      u.Path,
			Scheme:    proxy.ProxySchemeWS,
		}, nil

	case proxy.ProxySchemeWSListen:
		return &websocket.ProxyWSListener{
			Address: u.Host,
			Path:    u.Path,
		}, nil
	}

	return nil, fmt.Errorf("%w: %s", proxy.ErrProxyNotSupported, u)
}

func connect(node any) (io.ReadWriteCloser, error) {
	switch p := node.(type) {
	case io.ReadWriteCloser:
		return p, nil

	case *stdio.StdioWrapper:
		p.Reopen()
		return p, nil

	case proxy.ProxyDialer:
		conn, err := p.Dial()
		if err != nil {
			return nil, err
		}
		return conn, nil

	case proxy.ProxyListener:
		ln, err := p.Listen()
		if err != nil {
			return nil, err
		}
		conn, err := ln.Accept()
		if err != nil {
			return nil, err
		}
		return conn, err
	}

	panic("BUG: Wrong proxy type")
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
