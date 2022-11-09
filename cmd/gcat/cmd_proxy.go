package main

import (
	"fmt"
	"net"

	"github.com/rumpelsepp/gcat/lib/helper"
	"github.com/rumpelsepp/gcat/lib/proxy"
	_ "github.com/rumpelsepp/gcat/lib/proxy/exec"
	_ "github.com/rumpelsepp/gcat/lib/proxy/quic"
	_ "github.com/rumpelsepp/gcat/lib/proxy/stdio"
	_ "github.com/rumpelsepp/gcat/lib/proxy/tcp"
	_ "github.com/rumpelsepp/gcat/lib/proxy/tun"
	_ "github.com/rumpelsepp/gcat/lib/proxy/unix"
	_ "github.com/rumpelsepp/gcat/lib/proxy/websocket"
	_ "github.com/rumpelsepp/gcat/lib/proxy/webtransport"
	"github.com/spf13/cobra"
)

type mainLoop struct {
	proxyLeft  *proxy.Proxy
	proxyRight *proxy.Proxy
}

func CreateLoop(addrLeft, addrRight string) (*mainLoop, error) {
	addrLeftParsed, err := proxy.ParseAddr(addrLeft)
	if err != nil {
		return nil, err
	}

	addrRightParsed, err := proxy.ParseAddr(addrRight)
	if err != nil {
		return nil, err
	}

	proxyLeft, err := proxy.Registry.CreateProxyInstance(addrLeftParsed)
	if err != nil {
		return nil, err
	}

	proxyRight, err := proxy.Registry.CreateProxyInstance(addrRightParsed)
	if err != nil {
		return nil, err
	}

	return &mainLoop{
		proxyLeft:  proxyLeft,
		proxyRight: proxyRight,
	}, nil
}

func (l *mainLoop) CheckMultiple() bool {
	if !l.proxyLeft.SupportsMultiple || !l.proxyRight.SupportsMultiple {
		return false
	}
	return true
}

func (l *mainLoop) Connect() (net.Conn, net.Conn, error) {
	connLeft, err := l.proxyLeft.Connect()
	if err != nil {
		return nil, nil, err
	}

	connRight, err := l.proxyRight.Connect()
	if err != nil {
		return nil, nil, err
	}

	return connLeft, connRight, nil
}

type proxyOptions struct {
	loop     bool
	parallel bool
}

var (
	proxyOpts proxyOptions
	proxyCmd  = &cobra.Command{
		Use:   "proxy [flags] URL1 URL2",
		Short: "Act as a fancy socat like proxy tool",
		Long: `The proxy command needs two arguments which specify the data pipeline.
The arguments are URLs; in some rare cases it might be required to escape
certain parts of the url. For more information to URLs see the "proxies"
command.
`,
		Example: `  Listen on localhost tcp port 1234 and proxy to stdio.

      $ gcat proxy tcp-listen://localhost:1234 -

  Forward TCP traffic from "localhost:8080" to "1.1.1.1:80":

      $ gcat proxy tcp-listen://localhost:1234 tcp://1.1.1.1:80

  Tunnel IP traffic through SSH (https://rumpelsepp.org/blog/vpn-over-ssh/):

      # gcat proxy "tun://192.168.255.1/24" exec:'ssh root@HOST gcat proxy tun://192.168.255.2/24 -'

  SSH Tunnel through Websocket (https://rumpelsepp.org/blog/ssh-through-websocket/):

      $ ssh -o 'ProxyCommand=gcat proxy wss://example.org/ssh/ -' user@example.org`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("provide two urls")
			}

			loop, err := CreateLoop(args[0], args[1])
			if err != nil {
				return err
			}

			if proxyOpts.loop {
				for {
					lConn, rConn, err := loop.Connect()
					if err != nil {
						return err
					}

					helper.BidirectCopy(lConn, rConn)
				}
			}

			if proxyOpts.parallel {
				if !loop.CheckMultiple() {
					return fmt.Errorf("multiple connections not supported by chosen pipeline")
				}

				for {
					lConn, rConn, err := loop.Connect()
					if err != nil {
						return err
					}

					go helper.BidirectCopy(lConn, rConn)
				}
			}

			lConn, rConn, err := loop.Connect()
			if err != nil {
				return err
			}

			helper.BidirectCopy(lConn, rConn)

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(proxyCmd)
	f := proxyCmd.Flags()
	f.BoolVarP(&proxyOpts.loop, "loop", "l", false, "keep the listener running")
	f.BoolVarP(&proxyOpts.parallel, "parallel", "p", false, "serve multiple connections in parallel")
}
