package tun

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"strconv"

	"github.com/rumpelsepp/gcat/lib/proxy"
	"github.com/vishvananda/netlink"
)

type nativeTUN struct {
	*os.File
	netlink.Link
	baseConn proxy.BaseConn
}

func createNativeTUN(dev string) (*nativeTUN, error) {
	la := netlink.NewLinkAttrs()
	la.Name = dev

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
	return &nativeTUN{
		Link: iface,
		File: link.Fds[0]}, nil
}

func (tun *nativeTUN) Close() error {
	if err := tun.File.Close(); err != nil {
		return err
	}
	if err := netlink.LinkSetDown(tun.Link); err != nil {
		return err
	}
	return netlink.LinkDel(tun.Link)
}

func (tun *nativeTUN) LocalAddr() net.Addr {
	return tun.baseConn.LocalAddr()
}

func (tun *nativeTUN) RemoteAddr() net.Addr {
	return tun.baseConn.RemoteAddr()
}

func (tun *nativeTUN) Index() int {
	return tun.Link.Attrs().Index
}

func (tun *nativeTUN) SetMTU(mtu int) error {
	return netlink.LinkSetMTU(tun.Link, mtu)
}

func (tun *nativeTUN) MTU() int {
	return tun.Link.Attrs().MTU
}

func (tun *nativeTUN) SetUP() error {
	return netlink.LinkSetUp(tun.Link)
}

func (tun *nativeTUN) AddAddressCIDR(cidrAddr string) error {
	addr, err := netlink.ParseAddr(cidrAddr)
	if err != nil {
		return err
	}

	if err := netlink.AddrAdd(tun.Link, addr); err != nil {
		return err
	}
	return nil
}
