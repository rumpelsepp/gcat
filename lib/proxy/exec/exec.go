package gexec

import (
	"context"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/rumpelsepp/gcat/lib/proxy"
)

type cmdConn struct {
	proxy.BaseConn

	command *exec.Cmd
	stdout  io.ReadCloser
	stdin   io.WriteCloser
}

func (w *cmdConn) Write(p []byte) (int, error) {
	return w.stdin.Write(p)
}

func (w *cmdConn) Read(p []byte) (int, error) {
	return w.stdout.Read(p)
}

func (w *cmdConn) Close() error {
	if w.command.Process != nil {
		if err := w.command.Process.Kill(); err != nil {
			return err
		}
	}

	// The exit code is != 0 when we kill it.
	w.command.Wait()
	return nil
}

func dialCommand(command *exec.Cmd, target *proxy.ProxyAddr) (*cmdConn, error) {
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, err
	}
	if err := command.Start(); err != nil {
		return nil, err
	}
	return &cmdConn{
		BaseConn: proxy.BaseConn{
			LocalAddress:  nil,
			RemoteAddress: target,
		},
		command: command,
		stdout:  stdout,
		stdin:   stdin,
	}, nil
}

type execDialer struct{}

func (d *execDialer) Dial(ctx context.Context, desc *proxy.ProxyDescription) (net.Conn, error) {
	var (
		cmd      = desc.GetStringOption("cmd")
		cmdParts = strings.Split(cmd, " ")
		command  = exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
	)

	return dialCommand(command, desc.Target())
}

type shellDialer struct{}

func (d *shellDialer) Dial(ctx context.Context, desc *proxy.ProxyDescription) (net.Conn, error) {
	var (
		cmd   = desc.GetStringOption("cmd")
		shell = os.Getenv("SHELL")
	)

	if shell == "" {
		shell = "sh"
	}

	return dialCommand(exec.Command(shell, "-c", cmd), desc.Target())
}

func init() {
	proxy.Registry.Add(proxy.ProxyDescription{
		Scheme:      "exec",
		Description: "spawn a programm and connect via stdio",
		Dialer:      &execDialer{},
		Examples: []string{
			"$ gcat proxy 'exec:?cmd=cat -'",
			"$ gcat proxy 'exec:cat -'",
		},
		StringOptions: []proxy.ProxyOption[string]{
			{
				Name:        "cmd",
				Description: "the relevant command",
			},
		},
	})

	proxy.Registry.Add(proxy.ProxyDescription{
		Scheme:      "system",
		Description: "execute `cmd` via a shell and connect via stdio",
		Dialer:      &shellDialer{},
		Examples: []string{
			"$ gcat proxy 'system:?cmd=cat -'",
			"$ gcat proxy 'system:cat -'",
		},
		StringOptions: []proxy.ProxyOption[string]{
			{
				Name:        "cmd",
				Description: "shell script",
			},
		},
	})
}
