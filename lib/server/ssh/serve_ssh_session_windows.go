package gssh

import (
	"github.com/gliderlabs/ssh"
)

func (c *serveSSHCommand) createPty(s ssh.Session, shell string) {
	panic("Windows support for SSH is not implemented")
}
