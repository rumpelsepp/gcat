package main

import (
	"os"

	"codeberg.org/rumpelsepp/socks5"
	"github.com/Fraunhofer-AISEC/penlogger"
	"github.com/spf13/cobra"
)

type serveSOCKS5Command struct {
	opts     *runtimeOptions
	listen   string
	username string
	password string
}

func (c *serveSOCKS5Command) run(cmd *cobra.Command, args []string) error {
	auth := socks5.AuthNoAuthRequired
	if c.username != "" && c.password != "" {
		auth = socks5.AuthUsernamePassword
	}

	srv := socks5.Server{
		Listen:   c.listen,
		Logger:   penlogger.NewLogger("socks5", os.Stderr),
		Auth:     auth,
		Username: c.username,
		Password: c.password,
	}

	return srv.Serve()
}
