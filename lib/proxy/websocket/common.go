package websocket

import (
	"github.com/rumpelsepp/gcat/lib/proxy"
)

var helpArgs = []proxy.ProxyHelpArg{
	{
		Name:        "Host",
		Type:        "string",
		Explanation: "target ip address",
	},
	{
		Name:        "Port",
		Type:        "int",
		Explanation: "target port",
	},
}
