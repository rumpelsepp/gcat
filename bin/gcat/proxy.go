package main

import (
	"fmt"
	"io"
	"sync"

	"codeberg.org/rumpelsepp/gcat/lib/proxy"
	_ "codeberg.org/rumpelsepp/gcat/lib/proxy/exec"
	_ "codeberg.org/rumpelsepp/gcat/lib/proxy/stdio"
	_ "codeberg.org/rumpelsepp/gcat/lib/proxy/tcp"
	_ "codeberg.org/rumpelsepp/gcat/lib/proxy/tun"
	"github.com/spf13/cobra"
)

func bidirectCopy(left io.ReadWriteCloser, right io.ReadWriteCloser) (int, int, error) {
	var (
		n1   = 0
		n2   = 0
		err  error
		err1 error
		err2 error
		wg   sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		if n, err := io.Copy(right, left); err != nil {
			err1 = err
		} else {
			n1 = int(n)
		}

		right.Close()
		wg.Done()
	}()

	go func() {
		if n, err := io.Copy(left, right); err != nil {
			err2 = err
		} else {
			n2 = int(n)
		}

		left.Close()
		wg.Done()
	}()

	wg.Wait()

	if err1 != nil && err2 != nil {
		err = fmt.Errorf("both copier failed; left: %s; right: %s", err1, err2)
	} else {
		if err1 != nil {
			err = err1
		} else if err2 != nil {
			err = err2
		}
	}

	return n1, n2, err
}

func mainLoop(left *proxy.Proxy, right *proxy.Proxy) error {
	connLeft, err := left.Connect()
	if err != nil {
		return err
	}

	connRight, err := right.Connect()
	if err != nil {
		return err
	}

	_, _, err = bidirectCopy(connLeft, connRight)
	if err != nil {
		return err
	}

	return nil
}

type proxyCommand struct {
	state *runtimeState
}

func (c *proxyCommand) run(cmd *cobra.Command, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("provide two urls")
	}

	var (
		addrLeftRaw  = args[0]
		addrRightRaw = args[1]
	)

	addrLeft, err := proxy.ParseAddr(addrLeftRaw)
	if err != nil {
		return err
	}

	addrRight, err := proxy.ParseAddr(addrRightRaw)
	if err != nil {
		return err
	}

	proxyLeft, err :=  proxy.Registry.Create(addrLeft)
	if err != nil {
		return err
	}

	proxyRight, err := proxy.Registry.Create(addrRight)
	if err != nil {
		return err
	}

	if c.state.loop {
		for {
			if err := mainLoop(proxyLeft, proxyRight); err != nil {
				return err
			}
		}
	}
	return mainLoop(proxyLeft, proxyRight)
}
