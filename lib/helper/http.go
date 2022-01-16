package helper

import (
	"crypto/tls"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
)

func NewHTTPServer(handler http.Handler, listen, requestLog string, tlsConfig *tls.Config) (*http.Server, error) {
	var h http.Handler = handler
	if requestLog != "" {
		if requestLog == "-" {
			h = handlers.LoggingHandler(os.Stderr, handler)
		} else {
			f, err := os.Create(requestLog)
			if err != nil {
				return nil, err
			}
			h = handlers.LoggingHandler(f, handler)
			defer f.Close()
		}
	}

	return  &http.Server{
		TLSConfig:    tlsConfig,
		Addr:         listen,
		Handler:      h,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}, nil
}
