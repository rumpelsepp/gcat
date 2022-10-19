package websocket

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/rumpelsepp/gcat/lib/proxy"
	"nhooyr.io/websocket"
)

type ProxyWS struct {
	KeepAlive time.Duration
	addr      *proxy.ProxyAddr
}

func CreateWSProxy(addr *proxy.ProxyAddr) (*proxy.Proxy, error) {
	return proxy.CreateProxyFromDialer(&ProxyWS{addr: addr}), nil
}

func (p *ProxyWS) Dial() (net.Conn, error) {
	var (
		target  = fmt.Sprintf("%s://%s%s", p.addr.Scheme, p.addr.Host, p.addr.Path)
		ctx     = context.Background()
		options = websocket.DialOptions{}
	)
	wsConn, _, err := websocket.Dial(ctx, target, &options)
	if err != nil {
		return nil, err
	}
	return websocket.NetConn(ctx, wsConn, websocket.MessageBinary), nil
}

func init() {
	proxy.Registry.Add(proxy.ProxyEntryPoint{
		Scheme: "ws",
		Create: CreateWSProxy,
		Help: proxy.ProxyHelp{
			Description: "connect to a quic host:port",
			Examples: []string{
				"$ gcat proxy ws://localhost:1234 -",
			},
			Args: helpArgs,
		},
	})
	proxy.Registry.Add(proxy.ProxyEntryPoint{
		Scheme: "wss",
		Create: CreateWSProxy,
		Help: proxy.ProxyHelp{
			Description: "connect to a quic host:port",
			Examples: []string{
				"$ gcat proxy wss://localhost:1234 -",
			},
			Args: helpArgs,
		},
	})
}
