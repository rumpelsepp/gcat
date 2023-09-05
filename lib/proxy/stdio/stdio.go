package stdio

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"time"

	"github.com/rumpelsepp/gcat/lib/proxy"
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

func (w *stdioWrapper) Open() error {
	if w.closed {
		w.closed = false
	}
	return nil
}

func (w *stdioWrapper) Close() error {
	if !w.closed {
		w.closed = true
	}
	return nil
}

type stdioDialer struct {
	stdioWrapper
}

func (p *stdioDialer) Dial(ctx context.Context, desc *proxy.ProxyDescription) (net.Conn, error) {
	if p.stdioWrapper.closed {
		if err := p.stdioWrapper.Open(); err != nil {
			return nil, err
		}
	}
	return &p.stdioWrapper, nil
}

func init() {
	proxy.Registry.Add(proxy.ProxyDescription{
		Scheme:      "stdio",
		Description: "use stdio; shortcut is `-`",
		Dialer: &stdioDialer{
			stdioWrapper: *newStdioWrapper(),
		},
		Examples: []string{
			"$ gcat proxy tcp-listen://localhost:1234 stdio:",
			"$ gcat proxy tcp-listen://localhost:1234 -",
		},
	})
}
