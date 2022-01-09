package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	gexec "codeberg.org/rumpelsepp/gcat/lib/exec"

	"codeberg.org/rumpelsepp/gcat"
	"codeberg.org/rumpelsepp/helpers"
	"github.com/spf13/cobra"
)

const (
	ProxySchemeExec      = "exec"
	ProxySchemeSTDIO     = "stdio"
	ProxySchemeTCP       = "tcp"
	ProxySchemeTCPListen = "tcp-listen"
	ProxySchemeTLS       = "tls"
	ProxySchemeTLSListen = "tls-listen"
	ProxySchemeTun       = "tun"
	ProxySchemeWS        = "ws"
	// ProxySchemeWSListen         = "ws-listen"
)

func setupProxy(u *url.URL) (interface{}, error) {
	query := u.Query()

	switch u.Scheme {
	case ProxySchemeExec:
		var (
			cmd      = query.Get("cmd")
			cmdParts = strings.Split(cmd, " ")
		)
		return &gexec.ProxyExec{
			Command: exec.Command(cmdParts[0], cmdParts[1:]...),
		}, nil

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

	case ProxySchemeTun:
		var (
			dev  = query.Get("dev")
			ip   = u.Host
			mask = strings.TrimPrefix(u.Path, "/")
			mtu  = query.Get("mtu")
		)

		if ip == "" {
			return nil, fmt.Errorf("invalid ip address specified")
		}
		if mask == "" || strings.Contains(mask, "/") {
			return nil, fmt.Errorf("invalid subnet mask specified: %s", mask)
		}

		tun, err := gcat.CreateTun(dev)
		if err != nil {
			return nil, err
		}

		if err := tun.AddAddressCIDR(fmt.Sprintf("%s/%s", ip, mask)); err != nil {
			return nil, err
		}

		if mtu != "" {
			mtuInt, err := strconv.Atoi(mtu)
			if err != nil {
				return nil, err
			}
			if err := tun.SetMTU(mtuInt); err != nil {
				return nil, err
			}
		}

		if err := tun.SetUP(); err != nil {
			return nil, err
		}

		return tun, nil

	case ProxySchemeWS:
		return &gcat.ProxyWS{
			Address:   u.Host,
			KeepAlive: 20 * time.Second, // TODO: Make configurable.
			Path:      u.Path,
			Scheme:    ProxySchemeWS,
		}, nil

	default:
		return nil, fmt.Errorf("%w: %s", gcat.ErrNotSupported, u)
	}
}

// TODO: Solve this with generics, once they are here.
func connect(proxy interface{}) (io.ReadWriteCloser, error) {
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
	if rawURL == "-" {
		return "stdio:"
	} else if strings.HasPrefix(rawURL, "exec:") && !strings.Contains(rawURL, "?") {
		cmdEncoded := url.QueryEscape(strings.TrimPrefix(rawURL, "exec:"))
		return fmt.Sprintf("exec:?cmd=%s", cmdEncoded)
	} else {
		return rawURL
	}
}

// TODO: Solve this with generics, once they are here.
func mainLoopSingle(left interface{}, right interface{}) {
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

// TODO: Solve this with generics, once they are here.
func mainLoopKeep(left interface{}, right interface{}) {
	for {
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
}

type proxyCommand struct {
	opts *runtimeOptions
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

	proxyLeft, err := setupProxy(urlLeft)
	if err != nil {
		return err
	}

	proxyRight, err := setupProxy(urlRight)
	if err != nil {
		return err
	}

	if c.opts.keepRunning {
		mainLoopKeep(proxyLeft, proxyRight)
	} else {
		mainLoopSingle(proxyLeft, proxyRight)
	}
	return nil
}
