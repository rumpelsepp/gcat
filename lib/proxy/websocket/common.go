package websocket

import (
	"github.com/rumpelsepp/gcat/lib/proxy"
)

var options = []proxy.ProxyOption[string]{
	{
		Name:        "Hostname",
		Description: "target ip address",
	},
	{
		Name:        "Port",
		Description: "target port",
	},
	{
		Name:        "Path",
		Description: "http path",
	},
}
