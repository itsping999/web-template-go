package server

import (
	"context"
	"net"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/wyuhsin/web-template-go/internal/conf"
)

var (
	_ transport.Server = (*TCPServer)(nil)
)

type TCPServer struct {
	lis    net.Listener
	log    *log.Helper
	c      *conf.Server_TCP
}

func NewTCPServer(c *conf.Server_TCP, logger log.Logger) *TCPServer {
	srv := &TCPServer{
		log:    log.NewHelper(logger),
		c:      c,
	}
	return srv
}

func (s *TCPServer) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.c.Addr)
	if err != nil {
		return err
	}
	s.lis = lis
	s.log.Infof("[TCP] server listening on: %s", s.lis.Addr().String())

	go func() {
		for {
			conn, err := s.lis.Accept()
			if err != nil {
				s.log.Error(err)
				return
			}
			go s.handleTCPConn(conn)
		}
	}()
	return nil
}

func (s *TCPServer) Stop(ctx context.Context) error {
	s.log.Info("[TCP] server stopping")
	return s.lis.Close()
}

func (s *TCPServer) handleTCPConn(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			s.log.Error(err)
			return
		}
		s.log.Infof("[TCP] received: %s", string(buf[:n]))
		conn.Write(buf[:n])
	}
}
