package main

// TODO

// import (
// 	"net/http"
// 	"time"
//
// 	"github.com/spf13/cobra"
// 	"golang.org/x/net/webdav"
// )
//
// type serveWebDAVCommand struct {
// 	// opts    *runtimeOptions
// 	path    string
// 	root    string
// 	address string
// }
//
// func (c *serveWebDAVCommand) run(cmd *cobra.Command, args []string) error {
// 	handler := &webdav.Handler{
// 		FileSystem: webdav.Dir(c.root),
// 		LockSystem: webdav.NewMemLS(),
// 	}
//
// 	handler := http.NewServeMux()
// 	handler.Handle(c.path, http.FileServer(http.Dir(c.root)))
// 	srv := &http.Server{
// 		Addr:         c.address,
// 		WriteTimeout: time.Second * 15,
// 		ReadTimeout:  time.Second * 15,
// 		IdleTimeout:  time.Second * 60,
// 		Handler:      handler,
// 	}
//
// 	if err := srv.ListenAndServe(); err != nil {
// 		return err
// 	}
// 	return nil
// }
