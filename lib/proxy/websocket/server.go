package websocket

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/rumpelsepp/gcat/lib/helper"
	"github.com/rumpelsepp/gcat/lib/proxy"
	"nhooyr.io/websocket"
)

type wsConnWrapper struct {
	net.Conn
	isClosed bool
	doneCh   chan bool
}

func (w *wsConnWrapper) Close() error {
	if w.isClosed {
		return nil
	}
	w.doneCh <- true
	err := w.Conn.Close()
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
}

func (ln *listener) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	wsConn, err := websocket.Accept(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	conn := websocket.NetConn(context.Background(), wsConn, websocket.MessageBinary)
	wrappedConn := &wsConnWrapper{
		Conn:   conn,
		doneCh: make(chan bool),
	}
	ln.newConnCh <- wrappedConn
	<-wrappedConn.doneCh
}

func (ln *listener) Listen(prox *proxy.Proxy) error {
	handler := http.NewServeMux()
	handler.HandleFunc(prox.GetStringOption("Path"), ln.handleWebsocket)

	server, err := helper.NewHTTPServer(handler, net.JoinHostPort(prox.GetStringOption("Hostname"), prox.GetStringOption("Port")), "", nil)
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
	return ln.httpServer.Shutdown(context.Background())
}

func init() {
	l := &listener{
		newConnCh:   make(chan *wsConnWrapper),
		errorCh:     make(chan error),
		isListening: false,
	}

	proxy.Registry.Add(proxy.Proxy{
		Scheme:      "ws-listen",
		Description: "serve websocket",
		Listener:    l,
		Examples: []string{
			"$ gcat proxy ws-listen://localhost:1234/ws -",
		},
		StringOptions: options,
	})
}
