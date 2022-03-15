package tun

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
)

type TUNDevice interface {
	io.ReadWriteCloser
	MTU() int
	SetMTU(mtu int) error
	SetUP() error
	AddAddressCIDR(addrCIDR string) error
}

type ProxyTUN struct {
	TUNDevice
}

func CreateProxyTUN(u *url.URL) (*ProxyTUN, error) {
	var (
		query = u.Query()
		dev   = query.Get("dev")
		ip    = u.Host
		mask  = strings.TrimPrefix(u.Path, "/")
		mtu   = query.Get("mtu")
	)

	if ip == "" {
		return nil, fmt.Errorf("invalid ip address specified")
	}
	if mask == "" || strings.Contains(mask, "/") {
		return nil, fmt.Errorf("invalid subnet mask specified: %s", mask)
	}

	tun, err := CreateTun(dev)
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

	return &ProxyTUN{tun}, nil
}

func (p *ProxyTUN) Dial() error {
	return nil
}
