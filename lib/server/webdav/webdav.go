package webdav

import (
	"crypto/tls"
	"net/http"

	"codeberg.org/rumpelsepp/gcat/lib/helper"
	"github.com/Fraunhofer-AISEC/penlogger"
	"golang.org/x/net/webdav"
)

type WebDAVServer struct {
	Root   string
	Listen string
	Logger *penlogger.Logger
}

func (s *WebDAVServer) Run() error {
	srv := &webdav.Handler{
		FileSystem: webdav.Dir(s.Root),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				s.Logger.LogWarningf("[%s]: %s, %s", r.Method, r.URL, err)
			} else {
				s.Logger.LogInfof("[%s]: %s", r.Method, r.URL)
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
