package main

import (
	gssh "github.com/rumpelsepp/gcat/lib/server/ssh"
	"github.com/spf13/cobra"
)

var (
	sshServer = gssh.NewSSHServer()
	serveSSHCmd   = &cobra.Command{
		Use:   "ssh",
		Short: "spawn a SSH server with SFTP support",
		RunE: func (cmd *cobra.Command, args []string) error {
			return sshServer.Run()
		},
	}
)

func init() {
	serveCmd.AddCommand(serveSSHCmd)
	f := serveSSHCmd.Flags()
	f.StringVarP(&sshServer.Address, "listen", "l", ":2222", "SSH listen address")
	f.StringVarP(&sshServer.User, "user", "u", "gcat", "SSH user")
	f.StringVarP(&sshServer.Passwd, "passwd", "P", "gcat", "SSH password")
	f.StringVarP(&sshServer.Shell, "shell", "s", "/bin/bash", "shell to use")
	f.StringVarP(&sshServer.HostKey, "host-key", "K", "", "path to host key file, if empty a random key is generated")
	f.StringVarP(&sshServer.AuthorizedKeys, "authorized-keys", "a", "", "path to authorized_keys file")
}
