package main

import (
	"crypto/tls"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rumpelsepp/gcat/lib/helper"
	"github.com/rumpelsepp/gcat/lib/server/doh"
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

type serveDOHOptions struct {
	upstream    string
	requestLog  string
	path        string
	listen      string
	tlsCertFile string
	tlsKeyFile  string
	randomTLS   bool
}

var (
	serveDOHOpts serveDOHOptions
	serveDOHCmd = &cobra.Command{
		Use:   "doh",
		Short: "Spawn a DOH server",
		RunE: func(cmd *cobra.Command, args []string) error {
			upstreams, err := parseUpstreams(serveDOHOpts.upstream)
			if err != nil {
				return err
			}

			if serveDOHOpts.randomTLS {
				keyfile, certfile, err := helper.GenKeypairFS()
				if err != nil {
					return err
				}

				serveDOHOpts.tlsCertFile = certfile
				serveDOHOpts.tlsKeyFile = keyfile

				defer func() {
					os.RemoveAll(filepath.Dir(certfile))
				}()
			}

			server := doh.DoHServer{
				Upstreams:   upstreams,
				Listen:      serveDOHOpts.listen,
				Path:        serveDOHOpts.path,
				RequestLog:  serveDOHOpts.requestLog,
				TLSCertFile: serveDOHOpts.tlsCertFile,
				TLSKeyFile:  serveDOHOpts.tlsKeyFile,
				TLSConfig:   &tls.Config{},
			}

			return server.Run()
		},
	}
)

func init() {
	serveCmd.AddCommand(serveDOHCmd)
	f := serveDOHCmd.Flags()
	f.StringVarP(&serveDOHOpts.listen, "listen", "l", "127.0.0.1:8053", "listen on this address:port")
	f.StringVarP(&serveDOHOpts.path, "path", "p", "/dns-query", "specify HTTP path")
	f.StringVarP(&serveDOHOpts.requestLog, "request-log", "r", "", "request logfile, `-` means stderr")
	f.StringVarP(&serveDOHOpts.upstream, "upstream", "u", "udp://127.0.0.1:53", "upstream DNS resolver, concatenate with `|`")
	f.BoolVarP(&serveDOHOpts.randomTLS, "random-keypair", "R", false, "autogenerate a TLS keypair")
	f.StringVarP(&serveDOHOpts.tlsKeyFile, "keyfile", "K", "", "path to TLS keyfile in PEM format")
	f.StringVarP(&serveDOHOpts.tlsCertFile, "certfile", "C", "", "path to TLS certfile in PEM format")
}
