package main

import (
	"fmt"

	"github.com/rumpelsepp/gcat/lib/server/webdav"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
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
			logger, err := zap.NewDevelopment()
			if err != nil {
				panic(fmt.Sprintf("can't initialize zap logger: %v", err))
			}
			defer logger.Sync()
			srv := webdav.WebDAVServer{
				Logger: logger.Sugar(),
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
