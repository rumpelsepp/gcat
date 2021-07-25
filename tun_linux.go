package gcat

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	cloneDevicePath = "/dev/net/tun"
	ifReqSize       = unix.IFNAMSIZ + 64
)

type NativeTun struct {
	*os.File
	nfd   int
	index int
	name  string
}

func netdevGetInt(name string, attribute int) (int, error) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return 0, err
	}

	var ifr [ifReqSize]byte
	copy(ifr[:], name)
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(attribute),
		uintptr(unsafe.Pointer(&ifr[0])),
	)
	if errno != 0 {
		return 0, fmt.Errorf("failed to get attribute %d: %s", attribute, errno.Error())
	}

	return int(*(*int32)(unsafe.Pointer(&ifr[unix.IFNAMSIZ]))), nil
}

func netdevSetInt(name string, attribute, value int) error {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	var ifr [ifReqSize]byte
	copy(ifr[:], name)
	*(*uint32)(unsafe.Pointer(&ifr[unix.IFNAMSIZ])) = uint32(value)
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(attribute),
		uintptr(unsafe.Pointer(&ifr[0])),
	)
	if errno != 0 {
		return fmt.Errorf("failed to set attribute %d: %s", attribute, errno.Error())
	}

	return nil
}

func netdevSetShort(name string, attribute, value int) error {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	var ifr [ifReqSize]byte
	copy(ifr[:], name)
	*(*uint16)(unsafe.Pointer(&ifr[unix.IFNAMSIZ])) = uint16(value)
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(attribute),
		uintptr(unsafe.Pointer(&ifr[0])),
	)
	if errno != 0 {
		return fmt.Errorf("failed to set attribute %d: %s", attribute, errno.Error())
	}

	return nil
}

func CreateTun(name string) (TunDevice, error) {
	nfd, err := unix.Open(cloneDevicePath, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	var (
		ifr       [ifReqSize]byte
		flags     uint16 = unix.IFF_TUN | unix.IFF_NO_PI
		nameBytes        = []byte(name)
	)

	if len(nameBytes) >= unix.IFNAMSIZ {
		return nil, errors.New("interface name too long")
	}

	copy(ifr[:], nameBytes)
	*(*uint16)(unsafe.Pointer(&ifr[unix.IFNAMSIZ])) = flags

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(nfd),
		uintptr(unix.TUNSETIFF),
		uintptr(unsafe.Pointer(&ifr[0])),
	)

	if errno != 0 {
		return nil, errno
	}

	b := bytes.Trim(ifr[:unix.IFNAMSIZ], "\x00")
	name = string(b)

	if err := unix.SetNonblock(nfd, true); err != nil {
		return nil, err
	}

	return &NativeTun{
		File: os.NewFile(uintptr(nfd), cloneDevicePath),
		nfd:  nfd,
		name: name,
	}, nil
}

func (tun *NativeTun) Index() (int, error) {
	return netdevGetInt(tun.name, unix.SIOCGIFINDEX)
}

func (tun *NativeTun) SetMTU(mtu int) error {
	return netdevSetInt(tun.name, unix.SIOCSIFMTU, mtu)
}

func (tun *NativeTun) MTU() (int, error) {
	return netdevGetInt(tun.name, unix.SIOCGIFMTU)
}

func (tun *NativeTun) SetUP() error {
	flags := unix.IFF_UP | unix.IFF_RUNNING
	return netdevSetShort(tun.name, unix.SIOCSIFFLAGS, flags)
}

// FIXME: convert to netlink or something more native
// https://stackoverflow.com/a/49334944
func (tun *NativeTun) AddAddressCIDR(cidrAddr string) error {
	ip, _, err := net.ParseCIDR(cidrAddr)
	if err != nil {
		return err
	}

	if ip.To4() != nil {
		out, err := exec.Command("ip", "address", "add", cidrAddr, "dev", tun.name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %s", err, string(out))
		}
	} else {
		out, err := exec.Command("ip", "-6", "address", "add", cidrAddr, "dev", tun.name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %s", err, string(out))
		}
	}

	return nil
}
