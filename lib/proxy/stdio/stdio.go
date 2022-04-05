package stdio

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"codeberg.org/rumpelsepp/gcat/lib/proxy"
	"golang.org/x/sys/unix"
)

type stdioWrapper struct {
	proxy.BaseConn

	stdin  *os.File
	stdout *os.File
	closed bool
}

func newStdioWrapper() *stdioWrapper {
	if err := unix.SetNonblock(unix.Stdin, true); err != nil {
		panic(err)
	}
	if err := unix.SetNonblock(unix.Stdout, true); err != nil {
		panic(err)
	}
	return &stdioWrapper{
		stdin:  os.NewFile(uintptr(unix.Stdin), "/dev/stdin"),
		stdout: os.NewFile(uintptr(unix.Stdout), "/dev/stdout"),
		closed: false,
	}
}

func (w *stdioWrapper) Read(p []byte) (int, error) {
	for {
		if w.closed {
			return 0, io.EOF
		}
		// In order to be able to properly cancel helpers.BidirectCopy(),
		// an artificial ReadDeadline is used.
		if err := w.stdin.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			return 0, err
		}
		n, err := w.stdin.Read(p)

		// The artificial poll timeout triggered.
		if errors.Is(err, os.ErrDeadlineExceeded) {
			continue
		}

		return n, err
	}
}

func (w *stdioWrapper) Write(p []byte) (int, error) {
	if w.closed {
		return 0, io.ErrClosedPipe
	}
	return w.stdout.Write(p)
}

func (w *stdioWrapper) Reopen() error {
	if !w.closed {
		return fmt.Errorf("stdio still open")
	}
	w.closed = false
	return nil
}

func (w *stdioWrapper) Close() error {
	if w.closed {
		return fmt.Errorf("stdio already closed")
	}
	w.closed = true
	return nil
}

func Create(addr *proxy.ProxyAddr) (*proxy.Proxy, error) {
	return proxy.CreateProxyFromConn(newStdioWrapper()), nil
}

func init() {
	proxy.Registry.Add(proxy.ProxyEntryPoint{
		Scheme:    "stdio",
		Create:    Create,
		ShortHelp: "just use stdio; shortcut is `-`",
		Help: `No arguments.

Example:

  $ gcat proxy tcp-listen://localhost:1234 stdio:

Short form:

  $ gcat proxy tcp-listen://localhost:1234 -`,
	})
}
