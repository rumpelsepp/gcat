package socks5

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"
	"time"

	"github.com/rumpelsepp/gcat/lib/helper"
	"go.uber.org/zap"
)

const (
	AddrIPv4       = 0x01
	AddrDomainName = 0x03
	AddrIPv6       = 0x04
)

const (
	AuthNoAuthRequired      = 0x00
	AuthGSSAPI              = 0x01
	AuthUsernamePassword    = 0x02
	AuthNoAcceptableMethods = 0xff
)

const (
	CmdConnect      = 0x01
	CmdBind         = 0x02
	CmdUDPAssociate = 0x03
)

const (
	RepSucceeded                 = 0x00
	RepGeneralSOCKSServerFailure = 0x01
	RepConnectionNotAllowed      = 0x02
	RepNetworkUnreachable        = 0x03
	RepHostUnreachable           = 0x04
	RepConnectionRefused         = 0x05
	RepTTLExpired                = 0x06
	RepCommandNotSupported       = 0x07
	RepAddressTypeNotSupported   = 0x08
)

const SocksVersion = 0x05

var (
	ErrNoAcceptableMethods = errors.New("no acceptable athentication method")
	ErrInvalidAddrType     = errors.New("invalid address type")
	ErrUnsupportedVersion  = errors.New("unsupported SOCKS protocol version")
)

// RFC1928 types
type VersionIdentifier struct {
	VER      byte
	NMETHODS byte
	METHODS  []byte
}

type MethodSelection struct {
	VER    byte
	METHOD byte
}

type Request struct {
	VER     byte
	CMD     byte
	RSV     byte
	ATYP    byte
	DSTAddr []byte
	DSTPort uint16
}

type Reply struct {
	VER     byte
	REP     byte
	RSV     byte
	ATYP    byte
	BNDAddr []byte
	BNDPort uint16
}

// RFC1929 types
type UsernamePasswordRequest struct {
	VER    byte // This is always 1
	ULEN   byte
	UNAME  []byte
	PLEN   byte
	PASSWD []byte
}

type UsernamePasswordResponse struct {
	VER    byte // This is always 1
	STATUS byte
}

type Server struct {
	Listen   string
	Logger   *zap.SugaredLogger
	Auth     int
	Username string
	Password string
}

func (s *Server) readHandshake(conn io.ReadWriteCloser) (byte, error) {
	var (
		vIdentifier   VersionIdentifier
		versionHeader = make([]byte, 2)
	)
	if _, err := io.ReadFull(conn, versionHeader); err != nil {
		return 0, err
	}

	vIdentifier.VER = versionHeader[0]
	vIdentifier.NMETHODS = versionHeader[1]

	methods := make([]byte, vIdentifier.NMETHODS)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return 0, err
	}
	vIdentifier.METHODS = methods

	// Currently, only AuthNoAuthRequired and Username/Password is supported
	// Need to rework this somehow, but it works for now. :)
	for _, method := range vIdentifier.METHODS {
		switch method {
		case AuthNoAuthRequired:
			if s.Auth == AuthNoAuthRequired {
				return AuthNoAuthRequired, nil
			}
		case AuthUsernamePassword:
			if s.Auth == AuthUsernamePassword && (s.Username != "" && s.Password != "") {
				return AuthUsernamePassword, nil
			}
		}
	}
	return 0, ErrNoAcceptableMethods
}

func (s *Server) readRequest(conn io.ReadWriteCloser) (Request, error) {
	var (
		req       Request
		addr      []byte
		reqHeader = make([]byte, 4)
		port      = make([]byte, 2)
	)
	if _, err := io.ReadFull(conn, reqHeader); err != nil {
		return Request{}, err
	}

	req.VER = reqHeader[0]
	req.CMD = reqHeader[1]
	req.RSV = reqHeader[2]
	req.ATYP = reqHeader[3]

	if req.VER != SocksVersion {
		return Request{}, ErrUnsupportedVersion
	}

	switch req.ATYP {
	case AddrIPv4:
		addr = make([]byte, 4)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return Request{}, err
		}
		req.DSTAddr = addr
	case AddrIPv6:
		addr := make([]byte, 16)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return Request{}, err
		}
		req.DSTAddr = addr
	case AddrDomainName:
		length := make([]byte, 1)
		if _, err := io.ReadFull(conn, length); err != nil {
			return Request{}, err
		}
		addr = make([]byte, length[0])
		if _, err := io.ReadFull(conn, addr); err != nil {
			return Request{}, err
		}
		req.DSTAddr = addr
	default:
		return Request{}, ErrInvalidAddrType
	}
	if _, err := io.ReadFull(conn, port); err != nil {
		return Request{}, err
	}
	req.DSTPort = binary.BigEndian.Uint16(port[:2])
	return req, nil
}

func (s *Server) sendMethodSelection(conn io.ReadWriteCloser, method byte) error {
	msg := MethodSelection{
		VER:    SocksVersion,
		METHOD: method,
	}
	if err := binary.Write(conn, binary.BigEndian, &msg); err != nil {
		return err
	}
	return nil
}

func (s *Server) readUsernamePasswordReq(conn io.ReadWriteCloser) (UsernamePasswordRequest, error) {
	var (
		req  UsernamePasswordRequest
		hdr  = make([]byte, 2)
		plen = make([]byte, 1)
	)
	if _, err := io.ReadFull(conn, hdr); err != nil {
		return UsernamePasswordRequest{}, err
	}

	req.VER = hdr[0]
	req.ULEN = hdr[1]
	req.UNAME = make([]byte, req.ULEN)

	if _, err := io.ReadFull(conn, req.UNAME); err != nil {
		return UsernamePasswordRequest{}, err
	}
	if _, err := io.ReadFull(conn, plen); err != nil {
		return UsernamePasswordRequest{}, err
	}

	req.PLEN = plen[0]
	req.PASSWD = make([]byte, req.PLEN)

	if _, err := io.ReadFull(conn, req.PASSWD); err != nil {
		return UsernamePasswordRequest{}, err
	}
	return req, nil
}

func (s *Server) sendUsernamePasswordReply(conn io.ReadWriteCloser, code byte) error {
	msg := UsernamePasswordResponse{
		VER:    1,
		STATUS: code,
	}
	if err := binary.Write(conn, binary.BigEndian, msg); err != nil {
		return err
	}
	return nil
}

func (s *Server) sendReply(conn io.ReadWriteCloser, code byte, addrType byte, addr []byte, port uint16) error {
	var (
		msg = Reply{
			VER:     SocksVersion,
			REP:     code,
			RSV:     0,
			ATYP:    addrType,
			BNDAddr: addr,
			BNDPort: port,
		}
		writer = bufio.NewWriter(conn)
	)
	if msg.ATYP == AddrIPv4 {
		msg.BNDAddr = []byte(net.IP(addr).To4())
	}
	if err := writer.WriteByte(msg.VER); err != nil {
		return err
	}
	if err := writer.WriteByte(msg.REP); err != nil {
		return err
	}
	if err := writer.WriteByte(msg.RSV); err != nil {
		return err
	}
	if err := writer.WriteByte(msg.ATYP); err != nil {
		return err
	}
	if _, err := writer.Write(msg.BNDAddr); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.BigEndian, msg.BNDPort); err != nil {
		return err
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	return nil
}

func (s *Server) sendError(conn io.ReadWriteCloser, code byte) error {
	return s.sendReply(conn, code, AddrIPv4, []byte{0, 0, 0, 0}, 0)
}

func (s *Server) serveClient(conn io.ReadWriteCloser) error {
	auth, err := s.readHandshake(conn)
	if err != nil {
		if err == ErrNoAcceptableMethods {
			s.sendMethodSelection(conn, AuthNoAcceptableMethods)
		}
		s.Logger.Info(err)
		conn.Close()
		return err
	}
	if err := s.sendMethodSelection(conn, auth); err != nil {
		s.Logger.Info(err)
		conn.Close()
		return err
	}
	if auth != AuthNoAuthRequired {
		switch auth {
		case AuthUsernamePassword:
			req, err := s.readUsernamePasswordReq(conn)
			if err != nil {
				s.Logger.Info(err)
				conn.Close()
				return err
			}
			if string(req.UNAME) != s.Username || string(req.PASSWD) != s.Password {
				s.sendUsernamePasswordReply(conn, 1)
				conn.Close()
				return err
			}
			if err := s.sendUsernamePasswordReply(conn, 0); err != nil {
				s.Logger.Info(err)
				conn.Close()
				return err
			}
		default:
			// This is a bug. It fails earlier in readHandshake().
			panic("BUG: auth method")
		}
	}
	req, err := s.readRequest(conn)
	if err != nil {
		s.Logger.Info(err)
		conn.Close()
		return err
	}

	switch req.CMD {
	case CmdConnect:
		var (
			addr net.IP
			host string
			port = req.DSTPort
		)
		switch req.ATYP {
		case AddrIPv4:
			addr = net.IPv4(req.DSTAddr[0], req.DSTAddr[1], req.DSTAddr[2], req.DSTAddr[3])
			host = net.JoinHostPort(addr.String(), fmt.Sprintf("%d", port))
		case AddrIPv6:
			addr = req.DSTAddr
			host = net.JoinHostPort(addr.String(), fmt.Sprintf("%d", port))
		case AddrDomainName:
			host = string(req.DSTAddr)
		default:
			// This is a bug. It fails earlier in readRequest().
			panic("BUG: address type")
		}
		upstreamConn, err := net.DialTimeout("tcp", host, 2*time.Second)
		if err != nil {
			s.Logger.Info(err)
			if errors.Is(err, syscall.ECONNREFUSED) {
				if err := s.sendError(conn, RepConnectionRefused); err != nil {
					s.Logger.Info(err)
					return err
				}
			} else if errors.Is(err, syscall.EHOSTUNREACH) {
				if err := s.sendError(conn, RepHostUnreachable); err != nil {
					s.Logger.Info(err)
					return err
				}
			} else if errors.Is(err, syscall.ENETUNREACH) {
				if err := s.sendError(conn, RepNetworkUnreachable); err != nil {
					s.Logger.Info(err)
					return err
				}
			}
			return err
		}
		if err := s.sendReply(conn, RepSucceeded, req.ATYP, addr, req.DSTPort); err != nil {
			s.Logger.Info(err)
			upstreamConn.Close()
			return err
		}
		if _, _, err = helper.BidirectCopy(upstreamConn, conn); err != nil {
			s.Logger.Debug(err)
			return err
		}
	}
	return nil
}

func (s *Server) Serve() error {
	ln, err := net.Listen("tcp", s.Listen)
	if err != nil {
		return err
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			s.Logger.Info(err)
			continue
		}
		go s.serveClient(conn)
	}
}

func (s *Server) ServeFrom(conn io.ReadWriteCloser) error {
	return s.serveClient(conn)
}
