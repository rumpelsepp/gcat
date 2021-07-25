package gcat

import (
	"fmt"
	"io"
	"log"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
)

func makeSSHSessionHandler(shell string) ssh.Handler {
	return func(s ssh.Session) {
		log.Printf("New login from %s@%s", s.User(), s.RemoteAddr().String())
		cmd := exec.Command(shell)
		ptyReq, winCh, isPty := s.Pty()
		if isPty {
			cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
			f, err := pty.Start(cmd)
			if err != nil {
				panic(err)
			}
			go func() {
				for win := range winCh {
					winSize := &pty.Winsize{Rows: uint16(win.Height), Cols: uint16(win.Width)}
					pty.Setsize(f, winSize)
				}
			}()

			go io.Copy(f, s)
			go io.Copy(s, f)

			if err := cmd.Wait(); err != nil {
				log.Println("Session ended with error:", err)
				s.Exit(1)
			}
			log.Println("Session ended normally")
			s.Exit(0)
		} else {
			io.WriteString(s, "Remote forwarding available...\n")
			select {}
		}
	}
}
