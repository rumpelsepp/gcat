package main

import (
	"codeberg.org/rumpelsepp/gcat/lib/proxy"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var (
	proxiesCmd = &cobra.Command{
		Use:   "proxies",
		Short: "Show registered proxy plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			t := table.NewWriter()
			t.SetOutputMirror(cmd.OutOrStdout())
			t.AppendHeader(table.Row{"Scheme", "Description"})
			for _, v := range proxy.Registry.Values() {
				t.AppendRow(table.Row{v.Scheme, v.ShortHelp})
			}
			t.SortBy([]table.SortBy{{Name: "Scheme"}})
			t.Render()

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(proxiesCmd)
}
