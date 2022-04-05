package main

import (
	"net/http"

	"codeberg.org/rumpelsepp/gcat/lib/helper"
	"github.com/spf13/cobra"
)

type serveHTTPOptions struct {
	path    string
	root    string
	address string
}

var (
	serveHTTPOpts serveHTTPOptions
	serveHTTPCmd = &cobra.Command{
		Use:   "http",
		Short: "Spawn a HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			handler := http.NewServeMux()
			handler.Handle(serveHTTPOpts.path, http.FileServer(http.Dir(serveHTTPOpts.root)))

			server, err := helper.NewHTTPServer(handler, serveHTTPOpts.address, "", nil)
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
	f.StringVarP(&serveHTTPOpts.address, "address", "a", ":8080", "listen address")
	f.StringVarP(&serveHTTPOpts.root, "root", "r", ".", "HTTP root directory")
	f.StringVarP(&serveHTTPOpts.path, "path", "p", "/", "HTTP path")
}
