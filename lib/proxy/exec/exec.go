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
	// TODO: maybe use context here.
	if w.command.Process != nil {
		if err := w.command.Process.Kill(); err != nil {
			return err
		}
	}

	// The exit code is != 0 when we kill it.
	w.command.Wait()
	return nil
}

type ExecDialer struct {
}

func (d *ExecDialer) Dial(prox *proxy.Proxy) (net.Conn, error) {
	var (
		cmd      = prox.GetStringOption("cmd")
		cmdParts = strings.Split(cmd, " ")
		command  = exec.Command(cmdParts[0], cmdParts[1:]...)
	)

	command.Stderr = os.Stderr
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
			RemoteAddress: prox.Target(),
		},
		command: command,
		stdout:  stdout,
		stdin:   stdin,
	}, nil
}

func init() {
	proxy.Registry.Add(proxy.Proxy{
		Scheme:      "exec",
		Description: "spawn a programm and connect via stdio",
		Dialer:      &ExecDialer{},
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
}
