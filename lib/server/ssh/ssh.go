package gssh

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/Fraunhofer-AISEC/penlogger"
	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
)

type SSHServer struct {
	logger         *penlogger.Logger
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
		logger: penlogger.NewLogger("ssh", os.Stderr),
	}
}

func (srv *SSHServer) sftpHandler(s ssh.Session) {
	server, err := sftp.NewServer(s)
	if err != nil {
		srv.logger.LogErrorf("SFTP server init error: %s\n", err)
		return
	}

	srv.logger.LogDebugf("New SFTP connection from %s", s.RemoteAddr().String())
	if err := server.Serve(); err == io.EOF {
		server.Close()
		srv.logger.LogDebug("SFTP connection closed by client")
	} else if err != nil {
		srv.logger.LogErrorf("SFTP server exited with error: %s", err)
	}
}

func (srv *SSHServer) makeSSHSessionHandler(shell string) ssh.Handler {
	return func(s ssh.Session) {
		srv.logger.LogInfof("New login from %s@%s", s.User(), s.RemoteAddr().String())
		_, _, isPty := s.Pty()

		switch {
		case isPty:
			srv.logger.LogDebug("PTY requested")
			// TODO: better function sig, error handling.
			srv.createPty(s, shell)

		case len(s.Command()) > 0:
			srv.logger.LogInfof("No PTY requested, executing command: '%s'", s.RawCommand())

			cmd := exec.CommandContext(s.Context(), s.Command()[0], s.Command()[1:]...)

			if stdin, err := cmd.StdinPipe(); err != nil {
				srv.logger.LogError("Could not initialize StdinPipe", err)
				s.Exit(1)
				return
			} else {
				go func() {
					if _, err := io.Copy(stdin, s); err != nil {
						srv.logger.LogErrorf("Error while copying input from %s to stdin: %s", s.RemoteAddr().String(), err)
					}
					s.Close()
				}()
			}

			cmd.Stdout = s
			cmd.Stderr = s

			logError := func(f string, v ...any) {
				srv.logger.LogErrorf(f, v...)
				fmt.Fprintf(s, f, v...)
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
				srv.logger.LogInfo("Command execution successful")
				s.Exit(cmd.ProcessState.ExitCode())
				return

			case <-s.Context().Done():
				srv.logger.LogInfof("Session terminated: %s", s.Context().Err())
				return
			}

		default:
			srv.logger.LogError("No PTY requested, no command supplied")
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
					srv.logger.LogInfof("Successful authentication with password from %s@%s", ctx.User(), ctx.RemoteAddr().String())
					return true
				}
				srv.logger.LogWarningf("Invalid password from %s@%s", ctx.User(), ctx.RemoteAddr().String())
				return false
			},
			LocalPortForwardingCallback: func(ctx ssh.Context, dhost string, dport uint32) bool {
				srv.logger.LogInfof("Accepted forward to %s:%d", dhost, dport)
				return true
			},
			ReversePortForwardingCallback: func(ctx ssh.Context, host string, port uint32) bool {
				srv.logger.LogInfof("Attempt to bind at %s:%d granted", host, port)
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
				srv.logger.LogWarning("Encountered error while parsing public key:", err)
				continue
			}
			keys = append(keys, key)
		}

		server.PublicKeyHandler = func(ctx ssh.Context, key ssh.PublicKey) bool {
			for _, authKey := range keys {
				if bytes.Equal(key.Marshal(), authKey.Marshal()) {
					srv.logger.LogInfof("Successful authentication with ssh key from %s@%s", ctx.User(), ctx.RemoteAddr().String())
					return true
				}
			}
			srv.logger.LogNoticef("Invalid ssh key from %s@%s", ctx.User(), ctx.RemoteAddr().String())
			return false
		}
	}

	if err := server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
