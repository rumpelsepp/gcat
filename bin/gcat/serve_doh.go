package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/miekg/dns"
	"github.com/spf13/cobra"
)

var mime = "application/dns-message"

type dohServer struct {
	upstreams []string
	mutex     sync.Mutex
	curIndex  int
}

func (h *dohServer) nextIndex() int {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.curIndex = (h.curIndex + 1) % len(h.upstreams)
	return h.curIndex
}

func (h *dohServer) proxyDNSRequest(question *dns.Msg) (*dns.Msg, error) {
	resp, err := dns.Exchange(question, h.upstreams[h.nextIndex()])
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *dohServer) finishRequest(resp *dns.Msg, w http.ResponseWriter, r *http.Request) {
	buf, err := resp.Pack()
	if err != nil {
		fmt.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	// RFC8484 5.1
	cacheTTL := uint32(0)
	for _, m := range resp.Answer {
		ttl := m.Header().Ttl
		if cacheTTL == 0 {
			cacheTTL = ttl
		}
		if ttl < cacheTTL {
			cacheTTL = ttl
		}
	}
	w.Header().Set("Content-Type", mime)
	if cacheTTL > 0 {
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", cacheTTL))
	}
	if _, err := io.Copy(w, bytes.NewReader(buf)); err != nil {
		fmt.Println(err)
	}
}

func (h *dohServer) getRequest(w http.ResponseWriter, r *http.Request) {
	veryRawQuestion, ok := r.URL.Query()["dns"]
	if !ok {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	rawQuestion, err := base64.RawURLEncoding.DecodeString(veryRawQuestion[0])
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	var question dns.Msg
	if err := question.Unpack(rawQuestion); err != nil {
		fmt.Println(err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	question.Id = dns.Id()
	resp, err := h.proxyDNSRequest(&question)
	if err != nil {
		fmt.Println(err)
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	h.finishRequest(resp, w, r)
}

func (h *dohServer) postRequest(w http.ResponseWriter, r *http.Request) {
	rawQuestion, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	var question dns.Msg
	if err := question.Unpack(rawQuestion); err != nil {
		fmt.Println(err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	resp, err := h.proxyDNSRequest(&question)
	if err != nil {
		fmt.Println(err)
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	h.finishRequest(resp, w, r)
}

func parseUpstreams(s string) ([]string, error) {
	// TODO: An URL available dialer must be there first. So for now strip url.
	var (
		upstreamURLs = strings.Split(s, "|")
		out          []string
	)
	for _, upstreamURL := range upstreamURLs {
		u, err := url.Parse(upstreamURL)
		if err != nil {
			return nil, err
		}
		out = append(out, u.Host)
	}
	return out, nil
}

type serveDOHCommand struct {
	opts       *runtimeOptions
	upstream   string
	requestLog string
	path       string
	listen     string
}

func (c *serveDOHCommand) run(cmd *cobra.Command, args []string) error {
	upstreams, err := parseUpstreams(c.upstream)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	doh := dohServer{
		upstreams: upstreams,
	}
	r := mux.NewRouter()
	r.HandleFunc(c.path, doh.getRequest).Methods(http.MethodGet).Headers("Content-Type", mime)
	r.HandleFunc(c.path, doh.postRequest).Methods(http.MethodPost).Headers("Content-Type", mime)

	var h http.Handler = r
	if log := c.requestLog; log != "" {
		if log == "-" {
			h = handlers.LoggingHandler(os.Stderr, r)
		} else {
			f, err := os.Open(c.requestLog)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			h = handlers.LoggingHandler(f, r)
		}
	}

	httpServer := &http.Server{
		Addr:         c.listen,
		Handler:      h,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := httpServer.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
