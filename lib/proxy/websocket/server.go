package websocket

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/jba/muxpatterns" // will be included in the stdlib
	"github.com/rumpelsepp/gcat/lib/helper"
	"github.com/rumpelsepp/gcat/lib/proxy"
	"nhooyr.io/websocket"
)

type wsConnWrapper struct {
	net.Conn
	isClosed bool
	doneCh   chan bool
	context  context.Context
	cancel   context.CancelCauseFunc
}

func (w *wsConnWrapper) Close() error {
	if w.isClosed {
		return nil
	}

	err := w.Conn.Close()

	defer w.cancel(err)

	if err != nil {
		w.isClosed = true
		return nil
	}
	return err
}

type listener struct {
	newConnCh   chan *wsConnWrapper
	errorCh     chan error
	httpServer  *http.Server
	isListening bool
	context     context.Context
}

func (ln *listener) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	wsConn, err := websocket.Accept(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx, cancel := context.WithCancelCause(r.Context())

	var (
		conn        = websocket.NetConn(ln.context, wsConn, websocket.MessageBinary)
		wrappedConn = &wsConnWrapper{
			Conn:    conn,
			doneCh:  make(chan bool),
			context: ctx,
			cancel:  cancel,
		}
	)

	ln.newConnCh <- wrappedConn

	select {
	case <-wrappedConn.context.Done():
		if err := context.Cause(wrappedConn.context); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (ln *listener) Listen(desc *proxy.ProxyDescription) error {
	handler := muxpatterns.NewServeMux()
	handler.HandleFunc(fmt.Sprintf("GET %s", desc.GetStringOption("Path")), ln.handleWebsocket)

	server, err := helper.NewHTTPServer(handler, desc.TargetHost(), "", nil)
	if err != nil {
		return err
	}

	ln.httpServer = server

	newConnCh := make(chan *wsConnWrapper)
	ln.newConnCh = newConnCh

	go func() {
		if err := server.ListenAndServe(); err != nil {
			ln.errorCh <- err
		}
	}()

	ln.isListening = true

	return nil
}

func (ln *listener) Accept() (net.Conn, error) {
	select {
	case conn := <-ln.newConnCh:
		return conn, nil
	case err := <-ln.errorCh:
		return nil, err
	}
}

func (ln *listener) IsListening() bool {
	return ln.isListening
}

func (ln *listener) Close() error {
	return ln.httpServer.Shutdown(ln.context)
}

func init() {
	ln := &listener{
		newConnCh:   make(chan *wsConnWrapper),
		errorCh:     make(chan error),
		isListening: false,
		context:     context.Background(),
	}

	proxy.Registry.Add(proxy.ProxyDescription{
		Scheme:           "ws-listen",
		Description:      "serve websocket",
		Listener:         ln,
		SupportsMultiple: true,
		Examples: []string{
			"$ gcat proxy ws-listen://localhost:1234/ws -",
		},
		StringOptions: options,
	})
}
