package main

import (
	"github.com/rumpelsepp/gcat/lib/helper"
	"github.com/rumpelsepp/gcat/lib/server/socks5"
	"github.com/spf13/cobra"
)

type serveSOCKS5Options struct {
	listen   string
	username string
	password string
}

var (
	serveSOCKS5Opts serveSOCKS5Options
	serveSOCKS5Cmd  = &cobra.Command{
		Use:   "socks5",
		Short: "spawn a SOCKS5 server",
		RunE: func(cmd *cobra.Command, args []string) error {
			auth := socks5.AuthNoAuthRequired
			if serveSOCKS5Opts.username != "" && serveSOCKS5Opts.password != "" {
				auth = socks5.AuthUsernamePassword
			}

			srv := socks5.Server{
				Listen:   serveSOCKS5Opts.listen,
				Logger:   helper.GetLogger(),
				Auth:     auth,
				Username: serveSOCKS5Opts.username,
				Password: serveSOCKS5Opts.password,
			}

			return srv.ListenAndServe()
		},
	}
)

func init() {
	serveCmd.AddCommand(serveSOCKS5Cmd)
	f := serveSOCKS5Cmd.Flags()
	f.StringVarP(&serveSOCKS5Opts.listen, "listen", "l", ":1080", "listen address")
	f.StringVarP(&serveSOCKS5Opts.listen, "username", "u", "", "specify a username")
	f.StringVarP(&serveSOCKS5Opts.listen, "password", "p", "", "specify a password")
}
