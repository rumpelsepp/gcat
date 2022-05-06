package main

import (
	"github.com/rumpelsepp/gcat/lib/proxy"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func printRegistry(cmd *cobra.Command) error {
	t := table.NewWriter()
	t.SetOutputMirror(cmd.OutOrStdout())
	t.AppendHeader(table.Row{"Scheme", "Description"})
	for _, v := range proxy.Registry.Values() {
		t.AppendRow(table.Row{v.Scheme, v.ShortHelp})
	}
	t.SortBy([]table.SortBy{{Name: "Scheme"}})
	t.Render()

	return nil
}

var (
	proxiesCmd = &cobra.Command{
		Use:   "proxies [scheme]",
		Short: "Show registered proxy plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			switch len(args) {
			case 0:
				return printRegistry(cmd)
			case 1:
				p, err := proxy.Registry.Get(proxy.ProxyScheme(args[0]))
				if err != nil {
					return err
				}

				cmd.Println(p.Help)
			default:
				cmd.Usage()
				return nil
			}

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(proxiesCmd)
}
