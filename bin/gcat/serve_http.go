package main

import (
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

type serveHTTPCommand struct {
	opts    *runtimeOptions
	path    string
	root    string
	address string
}

func (c *serveHTTPCommand) run(cmd *cobra.Command, args []string) error {
	handler := http.NewServeMux()
	handler.Handle(c.path, http.FileServer(http.Dir(c.root)))
	srv := &http.Server{
		Addr:         c.address,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      handler,
	}

	if err := srv.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
