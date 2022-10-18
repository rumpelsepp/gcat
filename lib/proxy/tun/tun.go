package tun

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/rumpelsepp/gcat/lib/proxy"
)

type TUNDevice interface {
	net.Conn
	MTU() int
	SetMTU(mtu int) error
	SetUP() error
	AddAddressCIDR(addrCIDR string) error
}

type ProxyTUN struct {
	TUNDevice
}

func CreateProxyTUN(addr *proxy.ProxyAddr) (*proxy.Proxy, error) {
	var (
		query = addr.Query()
		ip    = addr.Host
		mask  = strings.TrimPrefix(addr.Path, "/")
		mtu   = query.Get("mtu")
	)

	if ip == "" {
		return nil, fmt.Errorf("invalid ip address specified")
	}
	if mask == "" || strings.Contains(mask, "/") {
		return nil, fmt.Errorf("invalid subnet mask specified: %s", mask)
	}

	tun, err := CreateNativeTun(addr)
	if err != nil {
		return nil, err
	}

	if err := tun.AddAddressCIDR(fmt.Sprintf("%s/%s", ip, mask)); err != nil {
		return nil, err
	}

	if mtu != "" {
		mtuInt, err := strconv.Atoi(mtu)
		if err != nil {
			return nil, err
		}
		if err := tun.SetMTU(mtuInt); err != nil {
			return nil, err
		}
	}

	if err := tun.SetUP(); err != nil {
		return nil, err
	}
	return proxy.CreateProxyFromConn(&ProxyTUN{tun}), nil
}

func init() {
	proxy.Registry.Add(proxy.ProxyEntryPoint{
		Scheme: "tun",
		Create: CreateProxyTUN,
		Help: proxy.ProxyHelp{
			Description: "allocate a tun device and send/recv raw ip packets",
			Examples: []string{
				"# gcat proxy 'tun://10.0.0.1/24?dev=tun%d' -",
			},
			Args: []proxy.ProxyHelpArg{
				{
					Name:        "Host",
					Type:        "string",
					Explanation: "IP address to assign to the device",
				},
				{
					Name:        "Path",
					Type:        "int",
					Explanation: "subnet mask",
				},
				{
					Name:        "mtu",
					Type:        "int",
					Explanation: "the mtu of the allocated 'tun' device",
					Default:     "1500",
				},
				{
					Name:        "dev",
					Type:        "string",
					Explanation: "Device name; can include '%d' for letting the kernel chose an index.",
				},
			},
		},
	})
}
