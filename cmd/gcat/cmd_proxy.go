package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/rumpelsepp/gcat/lib/helper"
	"github.com/rumpelsepp/gcat/lib/proxy"
	_ "github.com/rumpelsepp/gcat/lib/proxy/exec"
	_ "github.com/rumpelsepp/gcat/lib/proxy/quic"
	_ "github.com/rumpelsepp/gcat/lib/proxy/stdio"
	_ "github.com/rumpelsepp/gcat/lib/proxy/tcp"
	_ "github.com/rumpelsepp/gcat/lib/proxy/tun"
	_ "github.com/rumpelsepp/gcat/lib/proxy/websocket"
	"github.com/spf13/cobra"
)

type mainLoop struct {
	proxyLeft  *proxy.Proxy
	proxyRight *proxy.Proxy
	connLeft   net.Conn
	connRight  net.Conn
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

	proxyLeft, err := proxy.Registry.Create(addrLeftParsed)
	if err != nil {
		return nil, err
	}

	proxyRight, err := proxy.Registry.Create(addrRightParsed)
	if err != nil {
		return nil, err
	}

	return &mainLoop{
		proxyLeft:  proxyLeft,
		proxyRight: proxyRight,
	}, nil
}

func (l *mainLoop) Run() error {
	connLeft, err := l.proxyLeft.Connect()
	if err != nil {
		return err
	}
	l.connLeft = connLeft

	connRight, err := l.proxyRight.Connect()
	if err != nil {
		return err
	}
	l.connRight = connRight

	_, _, err = helper.BidirectCopy(l.connLeft, l.connRight)
	if err != nil {
		return err
	}
	return nil
}

func (l *mainLoop) Abort() {
	if l.connLeft != nil {
		l.connLeft.Close()
		l.connLeft = nil
	}
	if l.connRight != nil {
		l.connRight.Close()
		l.connRight = nil
	}
}

type proxyOptions struct {
	loop bool
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

      $ ssh -o 'ProxyCommand=gcat proxy wss://example.org/ssh/' user@example.org`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("provide two urls")
			}

			var (
				addrLeftRaw  = args[0]
				addrRightRaw = args[1]
				done         = make(chan error)
			)

			loop, err := CreateLoop(addrLeftRaw, addrRightRaw)
			if err != nil {
				return err
			}

			go func() {
				if proxyOpts.loop {
					for {
						loop.Run()
						loop.Abort()
					}
				}
				done <- loop.Run()
			}()

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)

			select {
			case <-c:
				loop.Abort()
				os.Exit(128 + int(syscall.SIGINT))
			case err := <-done:
				return err
			}

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(proxyCmd)
	f := proxyCmd.Flags()
	f.BoolVarP(&proxyOpts.loop, "loop", "l", false, "keep the listener running")
}
