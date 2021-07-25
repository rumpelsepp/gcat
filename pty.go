package gcat

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

type PTYWrapper struct {
	Command *exec.Cmd
	ptmx    *os.File
}

func (w *PTYWrapper) Write(p []byte) (int, error) {
	return w.ptmx.Write(p)
}

func (w *PTYWrapper) Read(p []byte) (int, error) {
	return w.ptmx.Read(p)
}

func (w *PTYWrapper) Close() error {
	// TODO: maybe use context here.
	if w.Command.Process != nil {
		if err := w.Command.Process.Kill(); err != nil {
			return err
		}
	}
	return w.Command.Wait()
}

type ProxyPTY struct {
	Command *exec.Cmd
}

func (p *ProxyPTY) Dial() (io.ReadWriteCloser, error) {
	// Start the command with a pty.
	ptmx, err := pty.Start(p.Command)
	if err != nil {
		return nil, err
	}
	// Make sure to close the pty at the end.
	defer func() { _ = ptmx.Close() }() // Best effort.

	// Handle pty size.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				fmt.Printf("error resizing pty: %s", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH                        // Initial resize.
	defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.

	// Set stdin in raw mode.
	oldState, err := term.MakeRaw(syscall.Stdin)
	if err != nil {
		panic(err)
	}
	defer func() { _ = term.Restore(syscall.Stdin, oldState) }() // Best effort.

	return &PTYWrapper{
		Command: p.Command,
		ptmx:    ptmx,
	}, nil
}
