package main

import (
	"github.com/spf13/cobra"
)

type serveOptions struct {
	tlsKeyFile  string
	tlsCertFile string
	path        string
	listen      string
	requestLog  string
}

var (
	serveOpts serveDOHOptions
	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Run a specific service",
		Example: `  $ gcat serve http
  $ gcat serve ssh -k /etc/ssh/ssh_host_ed25519_key -a ~/.ssh/authorized_keys`,
	}
)

func init() {
	sf := serveCmd.PersistentFlags()
	sf.StringVarP(&serveOpts.path, "tls-key", "c", "", "path to tls keyfile in pem format")
	sf.StringVarP(&serveOpts.path, "tls-cert", "k", "", "path to tls certfile in pem format")
	sf.StringVarP(&serveOpts.path, "path", "p", "", "working dir for the server")
	sf.StringVarP(&serveOpts.listen, "listen", "l", "localhost:1234", "listen address and port")
	sf.StringVarP(&serveOpts.requestLog, "request-log", "r", "-", "path to request log; `-` means stdout")

	rootCmd.AddCommand(serveCmd)
}
