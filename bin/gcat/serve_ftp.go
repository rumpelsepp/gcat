package main

import (
	"github.com/spf13/cobra"
	"goftp.io/server/v2"
	"goftp.io/server/v2/driver/file"
)

type serveFTPCommand struct {
	state  *runtimeState
	root   string
	port   uint16
	user   string
	passwd string
}

func (c *serveFTPCommand) run(cmd *cobra.Command, args []string) error {
	driver, err := file.NewDriver(c.root)
	if err != nil {
		return err
	}

	serverOpts := &server.Options{
		Name:   "gcat ftp server",
		Driver: driver,
		Port:   int(c.port),
		Auth: &server.SimpleAuth{
			Name:     c.user,
			Password: c.passwd,
		},
		Perm: server.NewSimplePerm("gcat", "gcat"),
	}

	ftpServer, err := server.NewServer(serverOpts)
	if err != nil {
		return err
	}

	if err := ftpServer.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
