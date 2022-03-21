package main

import (
	"fmt"
	"runtime/debug"
	"strings"

	"codeberg.org/rumpelsepp/gcat/lib/proxy"
	"github.com/Fraunhofer-AISEC/penlogger"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

type runtimeState struct {
	logger *penlogger.Logger

	loop bool
	verbose     bool
}

func getVersion() string {
	var builder strings.Builder

	info, ok := debug.ReadBuildInfo()
	if !ok {
		panic("could not read build info")
	}

	builder.WriteString(fmt.Sprintf("Go version: %s\n", info.GoVersion))
	builder.WriteString(fmt.Sprintf("Main version: %s\n", info.Main.Version))

	for _, setting := range info.Settings {
		if setting.Value == "" {
			continue
		}
		builder.WriteString(fmt.Sprintf("%s: %s\n", setting.Key, setting.Value))
	}
	return strings.TrimSpace(builder.String())
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
			Use:          "gcat",
			Short:        "gcat -- the swiss army knife for network protocols",
			Version:      getVersion(),
			SilenceUsage: true,
		}
		proxiesCobraCmd = &cobra.Command{
			Use:   "proxies",
			Short: "show registered proxy plugins",
			RunE: func(cmd *cobra.Command, args []string) error {
				t := table.NewWriter()
				t.SetOutputMirror(cmd.OutOrStdout())
				t.AppendHeader(table.Row{"Scheme", "Description"})
				for _, v := range proxy.ProxyRegistry {
					t.AppendRow(table.Row{v.Scheme, v.ShortHelp})
				}
				t.SortBy([]table.SortBy{{Name: "Scheme"}})
				t.Render()

				return nil
			},
		}
		serveCobraCmd = &cobra.Command{
			Use:   "serve",
			Short: "Run a specific service",
		}
		proxyCobraCmd = &cobra.Command{
			Use:   "proxy [flags] URL1 URL2",
			Short: "Act as a fancy socat like proxy tool",
			RunE:  proxyCmd.run,
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
	globalFlags.BoolVarP(&state.verbose, "verbose", "v", false, "enable verbose logging")

	// proxy
	rootCobraCmd.AddCommand(proxyCobraCmd)
	proxyFlags := proxyCobraCmd.Flags()
	// TODO: Can this live in the proxy cmd struct instead?
	proxyFlags.BoolVarP(&state.loop, "loop", "l", false, "keep the listener running")

	// proxies
	rootCobraCmd.AddCommand(proxiesCobraCmd)

	// serve
	rootCobraCmd.AddCommand(serveCobraCmd)

	// doh
	serveCobraCmd.AddCommand(serveDOHCobraCmd)
	dohFlags := serveDOHCobraCmd.Flags()
	dohFlags.StringVarP(&serveDOHCmd.listen, "listen", "l", "127.0.0.1:8053", "listen on this address:port")
	dohFlags.StringVarP(&serveDOHCmd.path, "path", "p", "/dns-query", "specify HTTP path")
	dohFlags.StringVarP(&serveDOHCmd.requestLog, "request-log", "r", "", "request logfile, `-` means stderr")
	dohFlags.StringVarP(&serveDOHCmd.upstream, "upstream", "u", "udp://127.0.0.1:53", "upstream DNS resolver, concatenate with `|`")
	dohFlags.BoolVarP(&serveDOHCmd.randomTLS, "random-keypair", "R", false, "autogenerate a TLS keypair")
	dohFlags.StringVarP(&serveDOHCmd.tlsKeyFile, "keyfile", "K", "", "path to TLS keyfile in PEM format")
	dohFlags.StringVarP(&serveDOHCmd.tlsCertFile, "certfile", "C", "", "path to TLS certfile in PEM format")

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
	httpFlags.StringVarP(&serveHTTPCmd.address, "address", "a", ":8080", "listen address")
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
	sshFlags.StringVarP(&serveSSHCmd.shell, "shell", "s", "/bin/bash", "shell to use")
	sshFlags.StringVarP(&serveSSHCmd.hostKey, "host-key", "k", "", "path to host key file, if empty a random key is generated")
	sshFlags.StringVarP(&serveSSHCmd.authorizedKeys, "authorized-keys", "a", "", "path to authorized_keys file")

	// webdav
	serveCobraCmd.AddCommand(serveWebDAVCobraCmd)
	webdavFlags := serveWebDAVCobraCmd.Flags()
	webdavFlags.StringVarP(&serveWebDAVCmd.address, "listen", "l", "127.0.0.1:8000", "listen on this address:port")
	webdavFlags.StringVarP(&serveWebDAVCmd.root, "root", "r", "", "directory root; default is CWD")

	// Wire everything up.
	rootCobraCmd.Execute()
}
