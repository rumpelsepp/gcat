package main

import (
	"log"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/net/webdav"
)

type serveWebDAVCommand struct {
	opts    *runtimeOptions
	root    string
	address string
}

func (c *serveWebDAVCommand) run(cmd *cobra.Command, args []string) error {
	srv := &webdav.Handler{
		FileSystem: webdav.Dir(c.root),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				log.Printf("WEBDAV [%s]: %s, ERROR: %s\n", r.Method, r.URL, err)
			} else {
				log.Printf("WEBDAV [%s]: %s \n", r.Method, r.URL)
			}
		},
	}

	httpServer := &http.Server{
		Addr:         c.address,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      srv,
	}

	if err := httpServer.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
