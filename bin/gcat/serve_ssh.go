package main

import (
	"io"
	"log"
	"os/exec"

	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
	"github.com/spf13/cobra"
)

type serveSSHCommand struct {
	opts    *runtimeOptions
	root    string
	address string
	user    string
	passwd  string
	shell   string
}

func sftpHandler(s ssh.Session) {
	server, err := sftp.NewServer(s)
	if err != nil {
		log.Printf("Sftp server init error: %s\n", err)
		return
	}

	log.Printf("New sftp connection from %s", s.RemoteAddr().String())
	if err := server.Serve(); err == io.EOF {
		server.Close()
		log.Println("Sftp connection closed by client")
	} else if err != nil {
		log.Println("Sftp server exited with error:", err)
	}
}

func makeSSHSessionHandler(shell string) ssh.Handler {
	return func(s ssh.Session) {
		log.Printf("New login from %s@%s", s.User(), s.RemoteAddr().String())
		_, _, isPty := s.Pty()

		switch {
		case isPty:
			log.Println("PTY requested")

			createPty(s, shell)

		case len(s.Command()) > 0:
			log.Printf("No PTY requested, executing command: '%s'", s.RawCommand())

			cmd := exec.Command(s.Command()[0], s.Command()[1:]...)
			// We use StdinPipe to avoid blocking on missing input
			if stdIn, err := cmd.StdinPipe(); err != nil {
				log.Println("Could not initialize StdInPipe", err)
				s.Exit(1)
				return
			} else {
				go func() {
					if _, err := io.Copy(stdIn, s); err != nil {
						log.Printf("Error while copying input from %s to stdIn: %s", s.RemoteAddr().String(), err)
					}
					if err := stdIn.Close(); err != nil {
						log.Println("Error while closing stdInPipe:", err)
					}
				}()
			}
			cmd.Stdout = s
			cmd.Stderr = s

			done := make(chan error, 1)
			go func() { done <- cmd.Run() }()

			select {
			case err := <-done:
				if err != nil {
					log.Println("Command execution failed:", err)
					io.WriteString(s, "Command execution failed: "+err.Error())
				} else {
					log.Println("Command execution successful")
				}
				s.Exit(cmd.ProcessState.ExitCode())

			case <-s.Context().Done():
				log.Println("Session closed by remote, killing dangling process")
				if cmd.Process != nil && cmd.ProcessState == nil {
					if err := cmd.Process.Kill(); err != nil {
						log.Println("Failed to kill process:", err)
					}
				}
			}

		default:
			log.Println("No PTY requested, no command supplied")

			select {
			case <-s.Context().Done():
				log.Println("Session closed")
			}
		}
	}
}

func (c *serveSSHCommand) run(cmd *cobra.Command, args []string) error {
	var (
		forwardHandler = &ssh.ForwardedTCPHandler{}
		server         = ssh.Server{
			Handler: makeSSHSessionHandler(c.shell),
			Addr:    c.address,
			PasswordHandler: ssh.PasswordHandler(func(ctx ssh.Context, pass string) bool {
				passed := pass == c.passwd
				if passed {
					log.Printf("Successful authentication with password from %s@%s", ctx.User(), ctx.RemoteAddr().String())
				} else {
					log.Printf("Invalid password from %s@%s", ctx.User(), ctx.RemoteAddr().String())
				}
				return passed
			}),
			LocalPortForwardingCallback: ssh.LocalPortForwardingCallback(func(ctx ssh.Context, dhost string, dport uint32) bool {
				log.Printf("Accepted forward to %s:%d", dhost, dport)
				return true
			}),
			ReversePortForwardingCallback: ssh.ReversePortForwardingCallback(func(ctx ssh.Context, host string, port uint32) bool {
				log.Printf("Attempt to bind at %s:%d granted", host, port)
				return true
			}),
			ChannelHandlers: map[string]ssh.ChannelHandler{
				"direct-tcpip": ssh.DirectTCPIPHandler,
				"session":      ssh.DefaultSessionHandler,
			},
			RequestHandlers: map[string]ssh.RequestHandler{
				"tcpip-forward":        forwardHandler.HandleSSHRequest,
				"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
			},
			SubsystemHandlers: map[string]ssh.SubsystemHandler{
				"sftp": sftpHandler,
			},
		}
	)

	if err := server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
