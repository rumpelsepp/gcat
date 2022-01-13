package main

import (
	"crypto/tls"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"codeberg.org/rumpelsepp/gcat/lib/helper"
	"codeberg.org/rumpelsepp/gcat/lib/server/doh"

	"github.com/spf13/cobra"
)

func parseUpstreams(s string) ([]netip.AddrPort, error) {
	// TODO: An URL available dialer must be there first. So for now strip url.
	var (
		upstreamURLs = strings.Split(s, "|")
		out          []netip.AddrPort
	)
	for _, upstreamURL := range upstreamURLs {
		u, err := url.Parse(upstreamURL)
		if err != nil {
			return nil, err
		}
		addr, err := netip.ParseAddrPort(u.Host)
		if err != nil {
			return nil, err
		}
		out = append(out, addr)
	}
	return out, nil
}

type serveDOHCommand struct {
	state        *runtimeState
	upstream    string
	requestLog  string
	path        string
	listen      string
	tlsCertFile string
	tlsKeyFile  string
	randomTLS   bool
}

func (c *serveDOHCommand) run(cmd *cobra.Command, args []string) error {
	upstreams, err := parseUpstreams(c.upstream)
	if err != nil {
		return err
	}

	if c.randomTLS {
		keyfile, certfile, err := helper.GenKeypairFS()
		if err != nil {
			return err
		}

		c.tlsCertFile = certfile
		c.tlsKeyFile = keyfile

		defer func() {
			os.RemoveAll(filepath.Dir(certfile))
		}()
	}

	server := doh.DoHServer{
		Upstreams:   upstreams,
		Listen:      c.listen,
		Path:        c.path,
		RequestLog:  c.requestLog,
		TLSCertFile: c.tlsCertFile,
		TLSKeyFile:  c.tlsKeyFile,
		TLSConfig:   &tls.Config{},
	}
	return server.Run()
}
