package main

import (
	"codeberg.org/rumpelsepp/penlogger"
	"github.com/spf13/cobra"
)

// TODO: find a better name for this. :)
type runtimeOptions struct {
	logger *penlogger.Logger

	keepRunning bool
	verbose     bool
}

func main() {
	var (
		opts         runtimeOptions
		serveFTPCmd  = serveFTPCommand{opts: &opts}
		serveHTTPCmd = serveHTTPCommand{opts: &opts}
		serveSSHCmd  = newServerSSHCommand(&opts)
		proxyCmd     = proxyCommand{opts: &opts}
	)

	var (
		rootCobraCmd = &cobra.Command{
			Use:   "gcat",
			Short: "gcat",
		}
		serveCobraCmd = &cobra.Command{
			Use: "serve",
		}
		proxyCobraCmd = &cobra.Command{
			Use:  "proxy",
			RunE: proxyCmd.run,
		}
		serveFTPCobraCmd = &cobra.Command{
			Use:  "ftp",
			RunE: serveFTPCmd.run,
		}
		serveHTTPCobraCmd = &cobra.Command{
			Use:  "http",
			RunE: serveHTTPCmd.run,
		}
		serveSSHCobraCmd = &cobra.Command{
			Use:  "ssh",
			RunE: serveSSHCmd.run,
		}
	)

	// globals
	globalFlags := rootCobraCmd.PersistentFlags()
	globalFlags.BoolVarP(&opts.verbose, "verbose", "v", false, "Enable verbose logging")

	// proxy
	rootCobraCmd.AddCommand(proxyCobraCmd)
	proxyFlags := proxyCobraCmd.Flags()
	// TODO: Can this live in the proxy cmd struct instead?
	proxyFlags.BoolVarP(&opts.keepRunning, "keep", "k", false, "Keep the listener running")

	// serve
	rootCobraCmd.AddCommand(serveCobraCmd)

	// ftp
	serveCobraCmd.AddCommand(serveFTPCobraCmd)
	ftpFlags := serveFTPCobraCmd.Flags()
	ftpFlags.StringVarP(&serveFTPCmd.root, "root", "r", ".", "FTP root directory")
	ftpFlags.StringVarP(&serveFTPCmd.user, "user", "u", "ftp", "FTP user")
	ftpFlags.StringVarP(&serveFTPCmd.passwd, "passwd", "P", "ftp", "FTP password")
	ftpFlags.Uint16VarP(&serveFTPCmd.port, "port", "p", 2121, "Listen TCP port")

	// http
	serveCobraCmd.AddCommand(serveHTTPCobraCmd)
	httpFlags := serveHTTPCobraCmd.Flags()
	httpFlags.StringVarP(&serveHTTPCmd.address, "address", "a", ":8080", "Listen address")
	httpFlags.StringVarP(&serveHTTPCmd.root, "root", "r", ".", "HTTP root directory")
	httpFlags.StringVarP(&serveHTTPCmd.path, "path", "p", "/", "HTTP path")

	// ssh
	// TODO: Add flags for ssh host keys and such
	serveCobraCmd.AddCommand(serveSSHCobraCmd)
	sshFlags := serveSSHCobraCmd.Flags()
	sshFlags.StringVarP(&serveSSHCmd.address, "listen", "l", ":2222", "SSH listen address")
	sshFlags.StringVarP(&serveSSHCmd.user, "user", "u", "gcat", "SSH user")
	sshFlags.StringVarP(&serveSSHCmd.passwd, "passwd", "p", "gcat", "SSH password")
	sshFlags.StringVarP(&serveSSHCmd.shell, "shell", "s", "/bin/bash", "Shell to use")

	// Wire everything up.
	rootCobraCmd.Execute()
}
