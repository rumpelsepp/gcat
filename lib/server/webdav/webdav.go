package webdav

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/rumpelsepp/gcat/lib/helper"
	"golang.org/x/net/webdav"
)

type WebDAVServer struct {
	Root   string
	Listen string
	Logger *slog.Logger
}

func (s *WebDAVServer) Run() error {
	srv := &webdav.Handler{
		FileSystem: webdav.Dir(s.Root),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				s.Logger.Warn(fmt.Sprintf("[%s]: %s, %s", r.Method, r.URL, err))
			} else {
				s.Logger.Warn(fmt.Sprintf("[%s]: %s", r.Method, r.URL))
			}
		},
	}

	httpServer, err := helper.NewHTTPServer(srv, s.Listen, "", &tls.Config{})
	if err != nil {
		return err
	}

	if err := httpServer.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
