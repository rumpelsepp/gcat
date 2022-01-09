package exec

import (
	"io"
	"os"
	"os/exec"
)

type CMDWrapper struct {
	Command *exec.Cmd
	stdout  io.ReadCloser
	stdin   io.WriteCloser
}

func (w *CMDWrapper) Write(p []byte) (int, error) {
	return w.stdin.Write(p)
}

func (w *CMDWrapper) Read(p []byte) (int, error) {
	return w.stdout.Read(p)
}

func (w *CMDWrapper) Close() error {
	// TODO: maybe use context here.
	if w.Command.Process != nil {
		if err := w.Command.Process.Kill(); err != nil {
			return err
		}
	}

	// The exit code is != 0 when we kill it.
	w.Command.Wait()
	return nil
}

type ProxyExec struct {
	Command *exec.Cmd
}

func (p *ProxyExec) Dial() (io.ReadWriteCloser, error) {
	p.Command.Stderr = os.Stderr
	stdout, err := p.Command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stdin, err := p.Command.StdinPipe()
	if err != nil {
		return nil, err
	}
	if err := p.Command.Start(); err != nil {
		return nil, err
	}
	return &CMDWrapper{
		Command: p.Command,
		stdout: stdout,
		stdin:  stdin,
	}, nil
}
