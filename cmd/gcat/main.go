package main

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

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

type globalOptions struct {
	verbose bool
}

var (
	gopts   globalOptions
	rootCmd = &cobra.Command{
		Use:   "gcat",
		Short: "gcat -- the swiss army knife for network protocols",
		Long: `gcat is a tool for penetration testers and sysadmins.
Its design is roughly based on "socat" (hence the name).
However, "gcat" provides the following delta to "socat":

  - "serve" command: "gcat" allows starting several different servers for quick usage.
    The "serve" command might be used in penetration tests or quick 'n' dirty lab setups.
    Here is an excerpt for supported protocols: "doh", "ftp", "http", "ssh", "webdav".

  - "proxy" command: it works similar to "socat". Data is proxied between two proxy modules, 
    specified as command line arguments. The "proxy" command uses URLs for its arguments.`,
		Version:      getVersion(),
		SilenceUsage: true,
	}
)

func main() {
	gf := rootCmd.PersistentFlags()
	gf.BoolVarP(&gopts.verbose, "verbose", "v", false, "enable verbose logging")

	rootCmd.Execute()
}
