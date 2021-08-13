package gcat

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type wrapper struct {
	conn *websocket.Conn
}

func (w *wrapper) Write(p []byte) (int, error) {
	wr, err := w.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(wr, bytes.NewReader(p))
	if err != nil {
		return 0, err
	}
	if err := wr.Close(); err != nil {
		return 0, err
	}
	return int(n), nil
}

func (w *wrapper) Read(p []byte) (int, error) {
	msgType, r, err := w.conn.NextReader()
	if err != nil {
		return 0, err
	}
	if msgType != websocket.BinaryMessage {
		return 0, fmt.Errorf("unexpected message type")
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return 0, err
	}
	return copy(p, buf.Bytes()), nil
}

type WSTransport struct {
	Conn             *websocket.Conn
	wrap             *wrapper
	bufReader        *bufio.Reader
	mutex            sync.Mutex
	keepAliveRunning bool
}

func NewWSTransport(conn *websocket.Conn) *WSTransport {
	conn.SetPingHandler(nil)
	conn.SetPongHandler(nil)
	wrap := &wrapper{conn}
	return &WSTransport{
		Conn:      conn,
		wrap:      wrap,
		bufReader: bufio.NewReader(wrap),
	}
}

func (t *WSTransport) SetKeepAlive(timeOut time.Duration) error {
	t.mutex.Lock()
	if t.keepAliveRunning {
		t.mutex.Unlock()
		return fmt.Errorf("keep alive is already running")
	}
	t.mutex.Unlock()
	go func() {
		for {
			t.mutex.Lock()
			if !t.keepAliveRunning {
				t.mutex.Unlock()
				return
			}
			t.mutex.Unlock()
			d := time.Now().Add(timeOut)
			if err := t.Conn.WriteControl(websocket.PingMessage, nil, d); err != nil {
				return
			}
			time.Sleep(timeOut)
		}
	}()
	return nil
}

func (t *WSTransport) Read(p []byte) (int, error) {
	return t.bufReader.Read(p)
}

func (t *WSTransport) Write(p []byte) (int, error) {
	return t.wrap.Write(p)
}

func (t *WSTransport) Close() error {
	return t.Conn.Close()
}

func (t *WSTransport) LocalAddr() net.Addr {
	return t.Conn.LocalAddr()
}

func (t *WSTransport) SetDeadline(ti time.Time) error {
	return nil
}

func (t *WSTransport) SetReadDeadline(ti time.Time) error {
	return t.Conn.SetReadDeadline(ti)
}

func (t *WSTransport) SetWriteDeadline(ti time.Time) error {
	return t.Conn.SetWriteDeadline(ti)
}

func (t *WSTransport) RemoteAddr() net.Addr {
	return t.Conn.RemoteAddr()
}

// TODO: Add a rawURL query param, in order to support exotic urls as well.
type ProxyWS struct {
	Address   string
	KeepAlive time.Duration
	Path      string
	ReqHeader http.Header
	Scheme    string
}

func (p *ProxyWS) Dial() (io.ReadWriteCloser, error) {
	d := websocket.DefaultDialer
	conn, _, err := d.Dial(fmt.Sprintf("%s://%s/%s", p.Scheme, p.Address, p.Path), p.ReqHeader)
	if err != nil {
		return nil, err
	}
	tr := NewWSTransport(conn)
	if p.KeepAlive > 0 {
		tr.SetKeepAlive(p.KeepAlive)
	}
	return tr, nil
}

type ProxyWSListener struct {
	Address string
}

func (p *ProxyWSListener) handleWSUpgrade(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()
}

func (p *ProxyWSListener) foo() {
	handler := http.NewServeMux()
	handler.HandleFunc("/", p.handleWSUpgrade)
	srv := &http.Server{
		Addr:         p.Address,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      handler,
	}
	if err := srv.ListenAndServe(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
