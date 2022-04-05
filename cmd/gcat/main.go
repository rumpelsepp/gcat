package main

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

type globalOptions struct {
	verbose bool
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

var (
	gopts   globalOptions
	rootCmd = &cobra.Command{
		Use:          "gcat",
		Short:        "gcat -- the swiss army knife for network protocols",
		Version:      getVersion(),
		SilenceUsage: true,
	}
		serveCmd = &cobra.Command{
			Use:   "serve",
			Short: "Run a specific service",
		}
)

func main() {
	// globals
	gf := rootCmd.PersistentFlags()
	gf.BoolVarP(&gopts.verbose, "verbose", "v", false, "enable verbose logging")

	// serve
	rootCmd.AddCommand(serveCmd)

	// // ssh
	// // TODO: Add flags for ssh host keys and such
	// serveCobraCmd.AddCommand(serveSSHCobraCmd)
	// sshFlags := serveSSHCobraCmd.Flags()
	// sshFlags.StringVarP(&serveSSHCmd.address, "listen", "l", ":2222", "SSH listen address")
	// sshFlags.StringVarP(&serveSSHCmd.user, "user", "u", "gcat", "SSH user")
	// sshFlags.StringVarP(&serveSSHCmd.passwd, "passwd", "p", "gcat", "SSH password")
	// sshFlags.StringVarP(&serveSSHCmd.shell, "shell", "s", "/bin/bash", "shell to use")
	// sshFlags.StringVarP(&serveSSHCmd.hostKey, "host-key", "k", "", "path to host key file, if empty a random key is generated")
	// sshFlags.StringVarP(&serveSSHCmd.authorizedKeys, "authorized-keys", "a", "", "path to authorized_keys file")

	rootCmd.Execute()
}
