package gcat

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"codeberg.org/rumpelsepp/gcat/lib/helper"
	"nhooyr.io/websocket"
)

// TODO: Add a rawURL query param, in order to support exotic urls as well.
type ProxyWS struct {
	Address   string
	KeepAlive time.Duration
	Path      string
	ReqHeader http.Header
	Scheme    string
}

func (p *ProxyWS) Dial() (io.ReadWriteCloser, error) {
	var (
		target  = fmt.Sprintf("%s://%s/%s", p.Scheme, p.Address, p.Path)
		ctx     = context.Background()
		options = websocket.DialOptions{
			HTTPHeader: p.ReqHeader,
		}
	)
	wsConn, _, err := websocket.Dial(ctx, target, &options)
	if err != nil {
		return nil, err
	}
	conn := websocket.NetConn(ctx, wsConn, websocket.MessageBinary)
	return conn, nil
}

type wsConnWrapper struct {
	net.Conn
	doneCh chan bool
}

func (w *wsConnWrapper) Close() error {
	w.doneCh <- true
	return w.Conn.Close()
}

type ProxyWSListener struct {
	Address    string
	Path       string
	newConnCh  chan<- *wsConnWrapper
	httpServer *http.Server
}

func (p *ProxyWSListener) handleWebsocket(w http.ResponseWriter, r *http.Request) {
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
	p.newConnCh <- wrappedConn
	<-wrappedConn.doneCh
}

func (p *ProxyWSListener) Listen() (net.Listener, error) {
	handler := http.NewServeMux()
	handler.HandleFunc(p.Path, p.handleWebsocket)

	server, err := helper.NewHTTPServer(handler, p.Address, "", nil)
	if err != nil {
		return nil, err
	}

	p.httpServer = server

	newConnCh := make(chan *wsConnWrapper)
	p.newConnCh = newConnCh

	listener := wsListenerWrapper{
		server: server,
		connCh: newConnCh,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			// TODO: Make this better. :)
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	return &listener, nil
}

type wsListenerWrapper struct {
	server *http.Server
	connCh <-chan *wsConnWrapper
}

func (w *wsListenerWrapper) Accept() (net.Conn, error) {
	newConn := <-w.connCh
	return newConn, nil
}

func (w *wsListenerWrapper) Close() error {
	return w.server.Shutdown(context.Background())
}

func (w *wsListenerWrapper) Addr() net.Addr {
	return nil
}
