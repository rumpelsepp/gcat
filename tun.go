package gcat

import (
	"io"
)

type TunDevice interface {
	io.ReadWriteCloser
	MTU() int
	SetMTU(mtu int) error
	SetUP() error
	AddAddressCIDR(addrCIDR string) error
}
