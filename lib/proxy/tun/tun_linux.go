package tun

import (
	"fmt"
	"os"
	"os/user"
	"strconv"

	"github.com/vishvananda/netlink"
)

type NativeTUN struct {
	*os.File
	netlink.Link
}

func CreateTun(name string) (*NativeTUN, error) {
	la := netlink.NewLinkAttrs()
	la.Name = name

	u, err := user.Current()
	if err != nil {
		return nil, err
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return nil, err
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return nil, err
	}

	link := &netlink.Tuntap{
		LinkAttrs:  la,
		Mode:       netlink.TUNTAP_MODE_TUN,
		Flags:      netlink.TUNTAP_DEFAULTS | netlink.TUNTAP_NO_PI,
		NonPersist: true,
		Queues:     1,
		Owner:      uint32(uid),
		Group:      uint32(gid),
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
	return &NativeTUN{Link: iface, File: link.Fds[0]}, nil
}

func (tun *NativeTUN) Close() error {
	if err := tun.File.Close(); err != nil {
		return err
	}
	if err := netlink.LinkSetDown(tun.Link); err != nil {
		return err
	}
	return netlink.LinkDel(tun.Link)
}

func (tun *NativeTUN) Index() int {
	return tun.Link.Attrs().Index
}

func (tun *NativeTUN) SetMTU(mtu int) error {
	return netlink.LinkSetMTU(tun.Link, mtu)
}

func (tun *NativeTUN) MTU() int {
	return tun.Link.Attrs().MTU
}

func (tun *NativeTUN) SetUP() error {
	return netlink.LinkSetUp(tun.Link)
}

func (tun *NativeTUN) AddAddressCIDR(cidrAddr string) error {
	addr, err := netlink.ParseAddr(cidrAddr)
	if err != nil {
		return err
	}

	if err := netlink.AddrAdd(tun.Link, addr); err != nil {
		return err
	}
	return nil
}
