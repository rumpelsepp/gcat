package gssh

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
	"golang.org/x/exp/slog"
)

type SSHServer struct {
	logger         slog.Logger
	HostKey        string
	AuthorizedKeys string
	Root           string
	Address        string
	User           string
	Passwd         string
	Shell          string
}

func NewSSHServer() *SSHServer {
	return &SSHServer{
		logger: slog.New(slog.NewTextHandler(os.Stderr)),
	}
}

func (srv *SSHServer) sftpHandler(s ssh.Session) {
	server, err := sftp.NewServer(s)
	if err != nil {
		srv.logger.Error("SFTP server init error: %s\n", err)
		return
	}

	srv.logger.Debug(fmt.Sprintf("New SFTP connection from %s", s.RemoteAddr().String()))
	if err := server.Serve(); err == io.EOF {
		server.Close()
		srv.logger.Debug("SFTP connection closed by client")
	} else if err != nil {
		srv.logger.Error("SFTP server exited with error: %s", err)
	}
}

func (srv *SSHServer) makeSSHSessionHandler(shell string) ssh.Handler {
	return func(s ssh.Session) {
		srv.logger.Info("New login from %s@%s", s.User(), s.RemoteAddr().String())
		_, _, isPty := s.Pty()

		switch {
		case isPty:
			if err := srv.createPty(s, shell); err != nil {
				srv.logger.Error("error serving pty", err)
			}
			return

		case len(s.Command()) > 0:
			cmd := exec.CommandContext(s.Context(), s.Command()[0], s.Command()[1:]...)

			stdin, err := cmd.StdinPipe()
			if err != nil {
				srv.logger.Error("Could not initialize StdinPipe", err)
				s.Exit(1)
				return
			}

			go func() {
				if _, err := io.Copy(stdin, s); err != nil {
					srv.logger.Error(fmt.Sprintf("copying input from %s to stdin", s.RemoteAddr().String()), err)
				}
				s.Close()
			}()

			cmd.Stdout = s
			cmd.Stderr = s

			logError := func(str string, err error) {
				srv.logger.Error(str, err)
				fmt.Fprintf(s, "%s: %s", str, err)
			}

			done := make(chan error, 1)
			go func() { done <- cmd.Run() }()

			select {
			case err := <-done:
				if err != nil {
					logError("Command execution failed: %s\n", err)
					s.Exit(255)
					return
				}
				srv.logger.Info("Command execution successful")
				s.Exit(cmd.ProcessState.ExitCode())
				return

			case <-s.Context().Done():
				srv.logger.Info("Session terminated: %s", s.Context().Err())
				return
			}

		default:
			srv.logger.Error("No PTY requested, no command supplied", nil)
		}
	}
}

func (srv *SSHServer) Run() error {
	var (
		forwardHandler = &ssh.ForwardedTCPHandler{}
		server         = ssh.Server{
			Handler: srv.makeSSHSessionHandler(srv.Shell),
			Addr:    srv.Address,
			PasswordHandler: func(ctx ssh.Context, pass string) bool {
				if pass == srv.Passwd {
					srv.logger.Info(fmt.Sprintf("Successful authentication with password from %s@%s", ctx.User(), ctx.RemoteAddr().String()))
					return true
				}
				srv.logger.Warn("Invalid password from %s@%s", ctx.User(), ctx.RemoteAddr().String())
				return false
			},
			LocalPortForwardingCallback: func(ctx ssh.Context, dhost string, dport uint32) bool {
				srv.logger.Info("Accepted forward to %s:%d", dhost, dport)
				return true
			},
			ReversePortForwardingCallback: func(ctx ssh.Context, host string, port uint32) bool {
				srv.logger.Info("Attempt to bind at %s:%d granted", host, port)
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
				"sftp": srv.sftpHandler,
			},
		}
	)

	hostKey := srv.HostKey
	if hostKey != "" {
		if err := ssh.HostKeyFile(hostKey)(&server); err != nil {
			return err
		}
	}

	authorizedKeys := srv.AuthorizedKeys
	if authorizedKeys != "" {
		var keys []ssh.PublicKey
		raw, err := os.ReadFile(authorizedKeys)
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(strings.NewReader(string(raw)))
		for scanner.Scan() {
			key, _, _, _, err := ssh.ParseAuthorizedKey(scanner.Bytes())
			if err != nil {
				srv.logger.Warn("Encountered error while parsing public key:", err)
				continue
			}
			keys = append(keys, key)
		}

		server.PublicKeyHandler = func(ctx ssh.Context, key ssh.PublicKey) bool {
			for _, authKey := range keys {
				if bytes.Equal(key.Marshal(), authKey.Marshal()) {
					srv.logger.Info("Successful authentication with ssh key from %s@%s", ctx.User(), ctx.RemoteAddr().String())
					return true
				}
			}
			srv.logger.Info("Invalid ssh key from %s@%s", ctx.User(), ctx.RemoteAddr().String())
			return false
		}
	}

	if err := server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
