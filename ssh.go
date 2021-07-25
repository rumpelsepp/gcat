package gcat

import (
	"io"
	"log"

	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
)

func SFTPHandler(s ssh.Session) {
	server, err := sftp.NewServer(s)
	if err != nil {
		log.Printf("Sftp server init error: %s\n", err)
		return
	}

	log.Printf("New sftp connection from %s", s.RemoteAddr().String())
	if err := server.Serve(); err == io.EOF {
		server.Close()
		log.Println("Sftp connection closed by client")
	} else if err != nil {
		log.Println("Sftp server exited with error:", err)
	}
}

// forwardHandler = &ssh.ForwardedTCPHandler{}
// 		server         = ssh.Server{
// 			Handler: makeSSHSessionHandler(shell),
// 			Addr:    bindAddr,
// 			PasswordHandler: ssh.PasswordHandler(func(ctx ssh.Context, pass string) bool {
// 				passed := pass == localPassword
// 				if passed {
// 					log.Printf("Successful authentication with password from %s@%s", ctx.User(), ctx.RemoteAddr().String())
// 				} else {
// 					log.Printf("Invalid password from %s@%s", ctx.User(), ctx.RemoteAddr().String())
// 				}
// 				return passed
// 			}),
// 			PublicKeyHandler: ssh.PublicKeyHandler(func(ctx ssh.Context, key ssh.PublicKey) bool {
// 				master, _, _, _, err := ssh.ParseAuthorizedKey([]byte(authorizedKey))
// 				if err != nil {
// 					log.Println("Encountered error while parsing public key:", err)
// 					return false
// 				}
// 				passed := bytes.Compare(key.Marshal(), master.Marshal()) == 0
// 				if passed {
// 					log.Printf("Successful authentication with ssh key from %s@%s", ctx.User(), ctx.RemoteAddr().String())
// 				} else {
// 					log.Printf("Invalid ssh key from %s@%s", ctx.User(), ctx.RemoteAddr().String())
// 				}
// 				return passed
// 			}),
// 			LocalPortForwardingCallback: ssh.LocalPortForwardingCallback(func(ctx ssh.Context, dhost string, dport uint32) bool {
// 				log.Printf("Accepted forward to %s:%d", dhost, dport)
// 				return true
// 			}),
// 			ReversePortForwardingCallback: ssh.ReversePortForwardingCallback(func(ctx ssh.Context, host string, port uint32) bool {
// 				log.Printf("Attempt to bind at %s:%d granted", host, port)
// 				return true
// 			}),
// 			ChannelHandlers: map[string]ssh.ChannelHandler{
// 				"direct-tcpip": ssh.DirectTCPIPHandler,
// 				"session":      ssh.DefaultSessionHandler,
// 			},
// 			RequestHandlers: map[string]ssh.RequestHandler{
// 				"tcpip-forward":        forwardHandler.HandleSSHRequest,
// 				"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
// 			},
// 			SubsystemHandlers: map[string]ssh.SubsystemHandler{
// 				"sftp": SFTPHandler,
// 			},
// 		}
