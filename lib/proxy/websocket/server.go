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

type ProxyWSListener struct {
	addr        *proxy.ProxyAddr
	newConnCh   chan *wsConnWrapper
	errorCh     chan error
	httpServer  *http.Server
	isListening bool
}

func CreateWSListenerProxy(addr *proxy.ProxyAddr) (*proxy.Proxy, error) {
	p := &ProxyWSListener{
		addr:        addr,
		newConnCh:   make(chan *wsConnWrapper),
		errorCh:     make(chan error),
		isListening: false,
	}
	return proxy.CreateProxyFromListener(p), nil
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

func (p *ProxyWSListener) Listen() error {
	handler := http.NewServeMux()
	handler.HandleFunc(p.addr.Path, p.handleWebsocket)

	server, err := helper.NewHTTPServer(handler, p.addr.Host, "", nil)
	if err != nil {
		return err
	}

	p.httpServer = server

	newConnCh := make(chan *wsConnWrapper)
	p.newConnCh = newConnCh

	go func() {
		if err := server.ListenAndServe(); err != nil {
			p.errorCh <- err
		}
	}()

	p.isListening = true

	return nil
}

func (p *ProxyWSListener) Accept() (net.Conn, error) {
	select {
	case conn := <-p.newConnCh:
		return conn, nil
	case err := <-p.errorCh:
		return nil, err
	}
}

func (p *ProxyWSListener) IsListening() bool {
	return p.isListening
}

func init() {
	proxy.Registry.Add(proxy.ProxyEntryPoint{
		Scheme: "ws-listen",
		Create: CreateWSListenerProxy,
		Help: proxy.ProxyHelp{
			Description: "serve websocket",
			Examples: []string{
				"$ gcat proxy ws-listen://localhost:1234/ws -",
			},
			Args: helpArgs,
		},
	})
}
