package gcat

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"
)

type StdioWrapper struct {
	Stdin  *os.File
	Stdout *os.File
	closed bool
}

func NewStdioWrapper() *StdioWrapper {
	if err := syscall.SetNonblock(syscall.Stdin, true); err != nil {
		panic(err)
	}
	if err := syscall.SetNonblock(syscall.Stdout, true); err != nil {
		panic(err)
	}
	return &StdioWrapper{
		Stdin:  os.NewFile(uintptr(syscall.Stdin), "/dev/stdin"),
		Stdout: os.NewFile(uintptr(syscall.Stdout), "/dev/stdout"),
		closed: false,
	}
}

func (w *StdioWrapper) Read(p []byte) (int, error) {
	for {
		if w.closed {
			return 0, io.EOF
		}
		// In order to be able to properly cancel helpers.BidirectCopy(),
		// an artificial ReadDeadline is used.
		if err := w.Stdin.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			return 0, err
		}
		n, err := w.Stdin.Read(p)

		// The artificial poll timeout triggered.
		if errors.Is(err, os.ErrDeadlineExceeded) {
			continue
		}

		return n, err
	}
}

func (w *StdioWrapper) Write(p []byte) (int, error) {
	if w.closed {
		return 0, io.ErrClosedPipe
	}
	return w.Stdout.Write(p)
}

func (w *StdioWrapper) Reopen() error {
	if !w.closed {
		return fmt.Errorf("stdio still open")
	}
	w.closed = false
	return nil
}

func (w *StdioWrapper) Close() error {
	if w.closed {
		return fmt.Errorf("stdio already closed")
	}
	w.closed = true
	return nil
}
