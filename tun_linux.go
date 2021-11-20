package gcat

import (
	"fmt"
	"os"

	"github.com/vishvananda/netlink"
)

type NativeTun struct {
	*os.File
	netlink.Link
}

func CreateTun(name string) (TunDevice, error) {
	la := netlink.NewLinkAttrs()
	la.Name = name

	link := &netlink.Tuntap{
		LinkAttrs:  la,
		Mode:       netlink.TUNTAP_MODE_TUN,
		Flags:      netlink.TUNTAP_DEFAULTS | netlink.TUNTAP_NO_PI,
		NonPersist: true,
		Queues:     1,
	}

	if err := netlink.LinkAdd(link); err != nil {
		return nil, err
	}

	iface, err := netlink.LinkByName(link.LinkAttrs.Name)
	if err != nil {
		return nil, err
	}
	if len(link.Fds) != 1 {
		return nil, fmt.Errorf("BUG: got too much tuntap fds")
	}
	return &NativeTun{Link: iface, File: link.Fds[0]}, nil
}

func (tun *NativeTun) Close() error {
	if err := netlink.LinkSetDown(tun.Link); err != nil {
		return err
	}
	return netlink.LinkDel(tun.Link)
}

func (tun *NativeTun) Index() int {
	return tun.Link.Attrs().Index
}

func (tun *NativeTun) SetMTU(mtu int) error {
	return netlink.LinkSetMTU(tun.Link, mtu)
}

func (tun *NativeTun) MTU() int {
	return tun.Link.Attrs().MTU
}

func (tun *NativeTun) SetUP() error {
	return netlink.LinkSetUp(tun.Link)
}

func (tun *NativeTun) AddAddressCIDR(cidrAddr string) error {
	addr, err := netlink.ParseAddr(cidrAddr)
	if err != nil {
		return err
	}

	if err := netlink.AddrAdd(tun.Link, addr); err != nil {
		return err
	}

	return nil
}
