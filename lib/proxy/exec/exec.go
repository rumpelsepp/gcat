package gexec

import (
	"io"
	"net"
	"os"
	"os/exec"
	"strings"

	"codeberg.org/rumpelsepp/gcat/lib/proxy"
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
	command    *exec.Cmd
	remoteAddr *proxy.ProxyAddr
}

func (d *ExecDialer) Dial() (net.Conn, error) {
	d.command.Stderr = os.Stderr
	stdout, err := d.command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stdin, err := d.command.StdinPipe()
	if err != nil {
		return nil, err
	}
	if err := d.command.Start(); err != nil {
		return nil, err
	}
	return &cmdConn{
		BaseConn: proxy.BaseConn{
			LocalAddress:  nil,
			RemoteAddress: d.remoteAddr,
		},
		command: d.command,
		stdout:  stdout,
		stdin:   stdin,
	}, nil
}

func CreateProxy(addr *proxy.ProxyAddr) (*proxy.Proxy, error) {
	var (
		query    = addr.Query()
		cmd      = query.Get("cmd")
		cmdParts = strings.Split(cmd, " ")
	)
	return &proxy.Proxy{
		Dialer: &ExecDialer{
			command:    exec.Command(cmdParts[0], cmdParts[1:]...),
			remoteAddr: addr,
		},
	}, nil
}

func init() {
	scheme := proxy.ProxyScheme("exec")

	proxy.ProxyRegistry[scheme] = proxy.ProxyEntryPoint{
		Scheme:    scheme,
		Create:    CreateProxy,
		ShortHelp: "spawn a programm and connect via stdio",
	}
}
