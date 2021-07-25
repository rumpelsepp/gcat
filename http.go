package gcat

import (
	"net/http"
	"time"
)

type ServerHTTPFile struct {
	Address string
	Root    string
	Path    string
}

func (s *ServerHTTPFile) Serve() error {
	path := s.Path
	if path == "" {
		path = "/"
	}

	handler := http.NewServeMux()
	handler.Handle(path, http.FileServer(http.Dir(s.Root)))
	srv := &http.Server{
		Addr:         s.Address,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      handler,
	}

	if err := srv.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
