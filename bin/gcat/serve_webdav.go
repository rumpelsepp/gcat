package main

import (
	"os"

	"codeberg.org/rumpelsepp/gcat/lib/server/webdav"
	"github.com/Fraunhofer-AISEC/penlogger"
	"github.com/spf13/cobra"
)

type serveWebDAVCommand struct {
	state   *runtimeState
	root    string
	address string
}

func (c *serveWebDAVCommand) run(cmd *cobra.Command, args []string) error {
	srv := webdav.WebDAVServer{
		Logger: penlogger.NewLogger("webdav", os.Stderr),
		Root:   c.root,
		Listen: c.address,
	}

	return srv.Run()
}
