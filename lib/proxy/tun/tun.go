package tun

import (
	"fmt"
	"net"
	"strings"

	"github.com/rumpelsepp/gcat/lib/proxy"
)

type tunDevice interface {
	net.Conn
	MTU() int
	SetMTU(mtu int) error
	SetUP() error
	AddAddressCIDR(addrCIDR string) error
}

type dialer struct {
	tunDevice
}

func (d *dialer) Dial(prox *proxy.Proxy) (net.Conn, error) {
	var (
		ip   = prox.GetStringOption("Host")
		mask = strings.TrimPrefix(prox.GetStringOption("Path"), "/")
		mtu  = prox.GetIntOption("mtu", 10)
		dev  = prox.GetStringOption("dev")
	)

	if ip == "" {
		return nil, fmt.Errorf("invalid ip address specified")
	}
	if mask == "" || strings.Contains(mask, "/") {
		return nil, fmt.Errorf("invalid subnet mask specified: %s", mask)
	}

	tun, err := createNativeTUN(dev)
	if err != nil {
		return nil, err
	}

	if err := tun.AddAddressCIDR(fmt.Sprintf("%s/%s", ip, mask)); err != nil {
		return nil, err
	}

	if err := tun.SetMTU(mtu); err != nil {
		return nil, err
	}

	if err := tun.SetUP(); err != nil {
		return nil, err
	}
	return tun, nil
}

func init() {
	proxy.Registry.Add(proxy.Proxy{
		Scheme:      "tun",
		Description: "allocate a tun device and send/recv raw ip packets",
		Examples: []string{
			"# gcat proxy 'tun://10.0.0.1/24?dev=tun%d' -",
		},
		Dialer: &dialer{},
		StringOptions: []proxy.ProxyOption[string]{
			{
				Name:        "Host",
				Description: "IP address to assign to the device",
				Default:     "10.0.0.1",
			},
			{
				Name:        "Path",
				Description: "subnet mask",
				Default:     "24",
			},
			{
				Name:        "dev",
				Description: "Device name; can include '%d' for letting the kernel chose an index.",
				Default:     "gcat-tun%d",
			},
		},
		IntOptions: []proxy.ProxyOption[int]{
			{
				Name:        "mtu",
				Description: "mtu of the allocated 'tun' device",
				Default:     1500,
			},
		},
	})
}
