package gexec

import (
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

func (d *execDialer) Dial(prox *proxy.Proxy) (net.Conn, error) {
	var (
		cmd      = prox.GetStringOption("cmd")
		cmdParts = strings.Split(cmd, " ")
		command  = exec.Command(cmdParts[0], cmdParts[1:]...)
	)

	return dialCommand(command, prox.Target())
}

type shellDialer struct{}

func (d *shellDialer) Dial(prox *proxy.Proxy) (net.Conn, error) {
	var (
		cmd   = prox.GetStringOption("cmd")
		shell = os.Getenv("SHELL")
	)

	if shell == "" {
		shell = "sh"
	}

	return dialCommand(exec.Command(shell, "-c", cmd), prox.Target())
}

func init() {
	proxy.Registry.Add(proxy.Proxy{
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

	proxy.Registry.Add(proxy.Proxy{
		Scheme:      "shell",
		Description: "spawn a shell and connect via stdio",
		Dialer:      &shellDialer{},
		Examples: []string{
			"$ gcat proxy 'shell:?cmd=cat -'",
			"$ gcat proxy 'shell:cat -'",
		},
		StringOptions: []proxy.ProxyOption[string]{
			{
				Name:        "cmd",
				Description: "shell script",
			},
		},
	})
}
