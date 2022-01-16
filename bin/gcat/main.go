package main

import (
	"fmt"
	"runtime/debug"

	"github.com/Fraunhofer-AISEC/penlogger"
	"github.com/spf13/cobra"
)

type runtimeState struct {
	logger *penlogger.Logger

	keepRunning bool
	verbose     bool
}

func main() {
	var (
		state          runtimeState
		serveDOHCmd    = serveDOHCommand{state: &state}
		serveFTPCmd    = serveFTPCommand{state: &state}
		serveHTTPCmd   = serveHTTPCommand{state: &state}
		serveSOCKS5Cmd = serveSOCKS5Command{state: &state}
		serveSSHCmd    = newServerSSHCommand(&state)
		serveWebDAVCmd = serveWebDAVCommand{state: &state}
		proxyCmd       = proxyCommand{state: &state}
	)

	var (
		rootCobraCmd = &cobra.Command{
			Use:   "gcat",
			Short: "gcat",
		}
		versionCobraCmd = &cobra.Command{
			Use:   "version",
			Short: "Show version information and exit",
			RunE: func(cmd *cobra.Command, args []string) error {
				info, ok := debug.ReadBuildInfo()
				if !ok {
					return fmt.Errorf("could not read build info")
				}

				cmd.Printf("Go version: %s\n", info.GoVersion)
				cmd.Printf("Main version: %s\n", info.Main.Version)

				for _, setting := range info.Settings {
					if setting.Value == "" {
						continue
					}
					cmd.Printf("%s: %s\n", setting.Key, setting.Value)
				}
				return nil
			},
		}
		serveCobraCmd = &cobra.Command{
			Use:   "serve",
			Short: "Run a specific service",
		}
		proxyCobraCmd = &cobra.Command{
			Use:  "proxy",
			RunE: proxyCmd.run,
		}
		serveDOHCobraCmd = &cobra.Command{
			Use:   "doh",
			Short: "Spawn a DOH server",
			RunE:  serveDOHCmd.run,
		}
		serveFTPCobraCmd = &cobra.Command{
			Use:   "ftp",
			Short: "Spawn a FTP server",
			RunE:  serveFTPCmd.run,
		}
		serveHTTPCobraCmd = &cobra.Command{
			Use:   "http",
			Short: "Spawn a HTTP server",
			RunE:  serveHTTPCmd.run,
		}
		serveSOCKS5CobraCmd = &cobra.Command{
			Use:   "socks5",
			Short: "Spawn a SOCKS5 server",
			RunE:  serveSOCKS5Cmd.run,
		}
		serveSSHCobraCmd = &cobra.Command{
			Use:   "ssh",
			Short: "Spawn a SSH server with SFTP support",
			RunE:  serveSSHCmd.run,
		}
		serveWebDAVCobraCmd = &cobra.Command{
			Use:   "webdav",
			Short: "Spawn a WebDAV server",
			RunE:  serveWebDAVCmd.run,
		}
	)

	// globals
	globalFlags := rootCobraCmd.PersistentFlags()
	globalFlags.BoolVarP(&state.verbose, "verbose", "v", false, "Enable verbose logging")

	// proxy
	rootCobraCmd.AddCommand(proxyCobraCmd)
	proxyFlags := proxyCobraCmd.Flags()
	// TODO: Can this live in the proxy cmd struct instead?
	proxyFlags.BoolVarP(&state.keepRunning, "keep", "k", false, "Keep the listener running")

	// serve
	rootCobraCmd.AddCommand(serveCobraCmd)

	// version
	rootCobraCmd.AddCommand(versionCobraCmd)

	// doh
	serveCobraCmd.AddCommand(serveDOHCobraCmd)
	dohFlags := serveDOHCobraCmd.Flags()
	dohFlags.StringVarP(&serveDOHCmd.listen, "listen", "l", "127.0.0.1:8053", "Listen on this address:port")
	dohFlags.StringVarP(&serveDOHCmd.path, "path", "p", "/dns-query", "Specify HTTP path")
	dohFlags.StringVarP(&serveDOHCmd.requestLog, "request-log", "r", "", "Request logfile, `-` means stderr")
	dohFlags.StringVarP(&serveDOHCmd.upstream, "upstream", "u", "udp://127.0.0.1:53", "Upstream DNS resolver, concatenate with `|`")
	dohFlags.BoolVarP(&serveDOHCmd.randomTLS, "random-keypair", "R", false, "Autogenerate a TLS keypair")
	dohFlags.StringVarP(&serveDOHCmd.tlsKeyFile, "keyfile", "K", "", "Path to TLS keyfile in PEM format")
	dohFlags.StringVarP(&serveDOHCmd.tlsCertFile, "certfile", "C", "", "Path to TLS certfile in PEM format")

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

	// socks5
	serveCobraCmd.AddCommand(serveSOCKS5CobraCmd)
	socks5Flags := serveSOCKS5CobraCmd.Flags()
	socks5Flags.StringVarP(&serveSOCKS5Cmd.listen, "listen", "l", ":1080", "listen address")
	socks5Flags.StringVarP(&serveSOCKS5Cmd.listen, "username", "u", "", "specify a username")
	socks5Flags.StringVarP(&serveSOCKS5Cmd.listen, "password", "p", "", "specify a password")

	// ssh
	// TODO: Add flags for ssh host keys and such
	serveCobraCmd.AddCommand(serveSSHCobraCmd)
	sshFlags := serveSSHCobraCmd.Flags()
	sshFlags.StringVarP(&serveSSHCmd.address, "listen", "l", ":2222", "SSH listen address")
	sshFlags.StringVarP(&serveSSHCmd.user, "user", "u", "gcat", "SSH user")
	sshFlags.StringVarP(&serveSSHCmd.passwd, "passwd", "p", "gcat", "SSH password")
	sshFlags.StringVarP(&serveSSHCmd.shell, "shell", "s", "/bin/bash", "Shell to use")
	sshFlags.StringVarP(&serveSSHCmd.hostKey, "host-key", "k", "", "Path to host key file, if empty a random key is generated")
	sshFlags.StringVarP(&serveSSHCmd.authorizedKeys, "authorized-keys", "a", "", "Path to authorized_keys file")

	// webdav
	serveCobraCmd.AddCommand(serveWebDAVCobraCmd)
	webdavFlags := serveWebDAVCobraCmd.Flags()
	webdavFlags.StringVarP(&serveWebDAVCmd.address, "listen", "l", "127.0.0.1:8000", "Listen on this address:port")
	webdavFlags.StringVarP(&serveWebDAVCmd.root, "root", "r", "", "Directory root; default is CWD")

	// Wire everything up.
	rootCobraCmd.Execute()
}
