package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"codeberg.org/rumpelsepp/penlogger"
	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
	"github.com/spf13/cobra"
)

type serveSSHCommand struct {
	logger  *penlogger.Logger
	opts    *runtimeOptions
	root    string
	address string
	user    string
	passwd  string
	shell   string
}

func newServerSSHCommand(state *runtimeOptions) *serveSSHCommand {
	return &serveSSHCommand{
		logger: penlogger.NewLogger("ssh", os.Stderr),
		opts:   state,
	}
}

func (c *serveSSHCommand) sftpHandler(s ssh.Session) {
	server, err := sftp.NewServer(s)
	if err != nil {
		c.logger.LogErrorf("SFTP server init error: %s\n", err)
		return
	}

	c.logger.LogDebugf("New SFTP connection from %s", s.RemoteAddr().String())
	if err := server.Serve(); err == io.EOF {
		server.Close()
		c.logger.LogDebug("SFTP connection closed by client")
	} else if err != nil {
		c.logger.LogErrorf("SFTP server exited with error: %s", err)
	}
}

func (c *serveSSHCommand) makeSSHSessionHandler(shell string) ssh.Handler {
	return func(s ssh.Session) {
		c.logger.LogInfof("New login from %s@%s", s.User(), s.RemoteAddr().String())
		_, _, isPty := s.Pty()

		switch {
		case isPty:
			c.logger.LogDebug("PTY requested")
			// TODO: better function sig, error handling.
			c.createPty(s, shell)

		case len(s.Command()) > 0:
			c.logger.LogInfo("No PTY requested, executing command: '%s'", s.RawCommand())

			var (
				ctx, cancel = context.WithCancel(context.Background())
				cmd         = exec.CommandContext(ctx, s.Command()[0], s.Command()[1:]...)
			)
			defer cancel()

			if stdin, err := cmd.StdinPipe(); err != nil {
				c.logger.LogError("Could not initialize StdinPipe", err)
				s.Exit(1)
				return
			} else {
				go func() {
					if _, err := io.Copy(stdin, s); err != nil {
						c.logger.LogErrorf("Error while copying input from %s to stdin: %s", s.RemoteAddr().String(), err)
					}
					// When the copy returns, kill the child process
					// by cancelling the context. Everything is cleaned
					// up automatically.
					cancel()
				}()
			}

			cmd.Stdout = s
			cmd.Stderr = s

			logError := func(f string, v ...interface{}) {
				c.logger.LogErrorf(f, v...)
				fmt.Fprintf(s, f, v...)
			}

			if err := cmd.Run(); err != nil {
				logError("Command execution failed: %s\n", err)
				s.Exit(255)
				return
			}
			if err := ctx.Err(); err != nil {
				logError("Command execution failed: %s\n", err)
				s.Exit(254)
				return
			}

			if cmd.ProcessState != nil {
				s.Exit(cmd.ProcessState.ExitCode())
				// When the process state is not populated something strange
				// happenend. I would consider this a bug that I overlooked.
			} else {
				logError("Unknown error happenend. Bug?\n")
				logError("You may report it: https://codeberg.org/rumpelsepp/gcat/issues\n")
			}

		default:
			c.logger.LogError("No PTY requested, no command supplied")
		}
	}
}

func (c *serveSSHCommand) run(cmd *cobra.Command, args []string) error {
	var (
		forwardHandler = &ssh.ForwardedTCPHandler{}
		server         = ssh.Server{
			Handler: c.makeSSHSessionHandler(c.shell),
			Addr:    c.address,
			PasswordHandler: func(ctx ssh.Context, pass string) bool {
				passed := pass == c.passwd
				if passed {
					c.logger.LogInfof("Successful authentication with password from %s@%s", ctx.User(), ctx.RemoteAddr().String())
				} else {
					c.logger.LogWarningf("Invalid password from %s@%s", ctx.User(), ctx.RemoteAddr().String())
				}
				return passed
			},
			LocalPortForwardingCallback: func(ctx ssh.Context, dhost string, dport uint32) bool {
				c.logger.LogInfof("Accepted forward to %s:%d", dhost, dport)
				return true
			},
			ReversePortForwardingCallback: func(ctx ssh.Context, host string, port uint32) bool {
				c.logger.LogInfof("Attempt to bind at %s:%d granted", host, port)
				return true
			},
			ChannelHandlers: map[string]ssh.ChannelHandler{
				"direct-tcpip": ssh.DirectTCPIPHandler,
				"session":      ssh.DefaultSessionHandler,
			},
			RequestHandlers: map[string]ssh.RequestHandler{
				"tcpip-forward":        forwardHandler.HandleSSHRequest,
				"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
			},
			SubsystemHandlers: map[string]ssh.SubsystemHandler{
				"sftp": c.sftpHandler,
			},
		}
	)

	if err := server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
