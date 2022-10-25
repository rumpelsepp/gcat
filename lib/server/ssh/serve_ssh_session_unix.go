// reverseSSH - a lightweight ssh server with a reverse connection feature
// Copyright (C) 2021  Ferdinor <ferdinor@mailbox.org>

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package gssh

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
)

func (srv *SSHServer) createPty(s ssh.Session, shell string) {
	var (
		ptyReq, winCh, _ = s.Pty()
		ctx, cancel      = context.WithCancel(context.Background())
		cmd              = exec.CommandContext(ctx, shell)
	)
	defer cancel()

	cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
	if currentUser, err := user.Current(); err == nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", currentUser.HomeDir))
	}
	f, err := pty.Start(cmd)
	if err != nil {
		srv.logger.Error("Could not start shell", err)
		os.Exit(1)
	}
	go func() {
		for win := range winCh {
			winSize := &pty.Winsize{Rows: uint16(win.Height), Cols: uint16(win.Width)}
			pty.Setsize(f, winSize)
		}
	}()

	go func() {
		io.Copy(f, s)
		s.Close()
	}()
	go func() {
		io.Copy(s, f)
		s.Close()
	}()

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			srv.logger.Error("Session ended with error", err)
			s.Exit(255)
			return
		}
		srv.logger.Info("Session ended normally")
		s.Exit(cmd.ProcessState.ExitCode())
		return

	case <-s.Context().Done():
		srv.logger.Info(fmt.Sprintf("Session terminated: %s", s.Context().Err()))
		return
	}
}
