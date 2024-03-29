package main

import (
	"log/slog"
	"os"

	"github.com/rumpelsepp/gcat/lib/server/webdav"
	"github.com/spf13/cobra"
)

type serveWebDAVOptions struct {
	root    string
	address string
}

var (
	serveWebDAVOpts serveWebDAVOptions
	serveWebDAVCmd  = &cobra.Command{
		Use:   "webdav",
		Short: "spawn a WebDAV server",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := webdav.WebDAVServer{
				Logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
				Root:   serveWebDAVOpts.root,
				Listen: serveWebDAVOpts.address,
			}

			return srv.Run()
		},
	}
)

func init() {
	serveCmd.AddCommand(serveWebDAVCmd)
	f := serveWebDAVCmd.Flags()
	f.StringVarP(&serveWebDAVOpts.address, "listen", "l", "127.0.0.1:8000", "listen on this address:port")
	f.StringVarP(&serveWebDAVOpts.root, "root", "", "", "directory root; default is CWD")
}
