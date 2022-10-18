package main

import (
	"net/http"

	"github.com/rumpelsepp/gcat/lib/helper"
	"github.com/spf13/cobra"
)

type serveHTTPOptions struct {
	path    string
	root    string
	address string
}

var (
	serveHTTPOpts serveHTTPOptions
	serveHTTPCmd  = &cobra.Command{
		Use:   "http",
		Short: "spawn a HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			handler := http.NewServeMux()
			handler.Handle(serveHTTPOpts.path, http.FileServer(http.Dir(serveHTTPOpts.root)))

			server, err := helper.NewHTTPServer(handler, serveOpts.listen, "", nil)
			if err != nil {
				return err
			}

			if err := server.ListenAndServe(); err != nil {
				return err
			}
			return nil
		},
	}
)

func init() {
	serveCmd.AddCommand(serveHTTPCmd)
	f := serveHTTPCmd.Flags()
	f.StringVarP(&serveHTTPOpts.path, "path", "p", "/", "HTTP path")
}
