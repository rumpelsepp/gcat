package websocket

import (
	"context"
	"fmt"
	"net"

	"github.com/rumpelsepp/gcat/lib/proxy"
	"nhooyr.io/websocket"
)

type dialer struct{}

func (p *dialer) Dial(prox *proxy.Proxy) (net.Conn, error) {
	var (
		target  = fmt.Sprintf("%s://%s%s", prox.Scheme, net.JoinHostPort(prox.GetStringOption("Hostname"), prox.GetStringOption("Port")), prox.GetStringOption("Path"))
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
	proxy.Registry.Add(proxy.Proxy{
		Scheme:      "ws",
		Description: "connect websocket host over http",
		Dialer:      &dialer{},
		Examples: []string{
			"$ gcat proxy ws://localhost:1234 -",
		},
		StringOptions: options,
	})
	proxy.Registry.Add(proxy.Proxy{
		Scheme:      "wss",
		Description: "connect websocket host over https",
		Dialer:      &dialer{},
		Examples: []string{
			"$ gcat proxy wss://localhost:1234 -",
		},
		StringOptions: options,
	})
}
