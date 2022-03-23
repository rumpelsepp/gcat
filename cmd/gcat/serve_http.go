package main

import (
	"net/http"

	"codeberg.org/rumpelsepp/gcat/lib/helper"
	"github.com/spf13/cobra"
)

type serveHTTPCommand struct {
	state   *runtimeState
	path    string
	root    string
	address string
}

func (c *serveHTTPCommand) run(cmd *cobra.Command, args []string) error {
	handler := http.NewServeMux()
	handler.Handle(c.path, http.FileServer(http.Dir(c.root)))

	server, err := helper.NewHTTPServer(handler, c.address, "", nil)
	if err != nil {
		return err
	}

	if err := server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
