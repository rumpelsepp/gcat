package doh

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"os"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/miekg/dns"
)

const mime = "application/dns-message"

type DoHServer struct {
	mutex    sync.Mutex
	curIndex int

	TLSConfig   *tls.Config
	TLSKeyFile  string
	TLSCertFile string
	Upstreams   []netip.AddrPort
	RequestLog  string
	Path        string
	Listen      string
}

func (s *DoHServer) nextIndex() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if len(s.Upstreams) > 1 {
		s.curIndex = (s.curIndex + 1) % len(s.Upstreams)
	}
	return s.curIndex
}

func (s *DoHServer) proxyDNSRequest(question *dns.Msg) (*dns.Msg, error) {
	resp, err := dns.Exchange(question, s.Upstreams[s.nextIndex()].String())
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *DoHServer) finishRequest(resp *dns.Msg, w http.ResponseWriter, r *http.Request) {
	buf, err := resp.Pack()
	if err != nil {
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

func (s *DoHServer) getRequest(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	question.Id = dns.Id()
	resp, err := s.proxyDNSRequest(&question)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	s.finishRequest(resp, w, r)
}

func (s *DoHServer) postRequest(w http.ResponseWriter, r *http.Request) {
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
	resp, err := s.proxyDNSRequest(&question)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	s.finishRequest(resp, w, r)
}

func (s *DoHServer) Run() error {
	r := mux.NewRouter()
	r.HandleFunc(s.Path, s.getRequest).Methods(http.MethodGet).Headers("Content-Type", mime)
	r.HandleFunc(s.Path, s.postRequest).Methods(http.MethodPost).Headers("Content-Type", mime)

	var h http.Handler = r
	if log := s.RequestLog; log != "" {
		if log == "-" {
			h = handlers.LoggingHandler(os.Stderr, r)
		} else {
			f, err := os.Create(s.RequestLog)
			if err != nil {
				return err
			}
			h = handlers.LoggingHandler(f, r)
			defer f.Close()
		}
	}

	httpServer := &http.Server{
		TLSConfig:    s.TLSConfig,
		Addr:         s.Listen,
		Handler:      h,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if s.TLSCertFile != "" && s.TLSKeyFile != "" {
		return httpServer.ListenAndServeTLS(s.TLSCertFile, s.TLSKeyFile)
	}
	return httpServer.ListenAndServe()
}
