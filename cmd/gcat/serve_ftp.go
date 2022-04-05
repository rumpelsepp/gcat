package main

import (
	"github.com/spf13/cobra"
	"goftp.io/server/v2"
	"goftp.io/server/v2/driver/file"
)

type serveFTPOptions struct {
	root   string
	port   uint16
	user   string
	passwd string
}

var (
	serveFTPOpts serveFTPOptions
	serveFTPCmd  = &cobra.Command{
		Use:   "ftp",
		Short: "Spawn a FTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			driver, err := file.NewDriver(serveFTPOpts.root)
			if err != nil {
				return err
			}

			serverOpts := &server.Options{
				Name:   "gcat ftp server",
				Driver: driver,
				Port:   int(serveFTPOpts.port),
				Auth: &server.SimpleAuth{
					Name:     serveFTPOpts.user,
					Password: serveFTPOpts.passwd,
				},
				Perm: server.NewSimplePerm("gcat", "gcat"),
			}

			ftpServer, err := server.NewServer(serverOpts)
			if err != nil {
				return err
			}

			if err := ftpServer.ListenAndServe(); err != nil {
				return err
			}
			return nil
		},
	}
)

func init() {
	serveCmd.AddCommand(serveFTPCmd)
	f := serveFTPCmd.Flags()
	f.StringVarP(&serveFTPOpts.root, "root", "r", ".", "FTP root directory")
	f.StringVarP(&serveFTPOpts.user, "user", "u", "ftp", "FTP user")
	f.StringVarP(&serveFTPOpts.passwd, "passwd", "P", "ftp", "FTP password")
	f.Uint16VarP(&serveFTPOpts.port, "port", "p", 2121, "Listen TCP port")
}
