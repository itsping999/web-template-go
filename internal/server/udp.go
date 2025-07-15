package server

import (
	"context"
	"net"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/wyuhsin/web-template-go/internal/conf"
)

var (
	_ transport.Server = (*UDPServer)(nil)
)

type UDPServer struct {
	conn *net.UDPConn
	log  *log.Helper
	c    *conf.Server_UDP
}

func NewUDPServer(c *conf.Server_UDP, logger log.Logger) *UDPServer {
	srv := &UDPServer{
		log: log.NewHelper(logger),
		c:   c,
	}
	return srv
}

func (s *UDPServer) Start(ctx context.Context) error {
	addr, err := net.ResolveUDPAddr("udp", s.c.Addr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	s.conn = conn
	s.log.Infof("[UDP] server listening on: %s", s.conn.LocalAddr().String())

	go func() {
		buf := make([]byte, 1024)
		for {
			n, remoteAddr, err := s.conn.ReadFromUDP(buf)
			if err != nil {
				s.log.Error(err)
				return
			}
			s.log.Infof("[UDP] received: %s from %s", string(buf[:n]), remoteAddr.String())
			s.conn.WriteToUDP(buf[:n], remoteAddr)
		}
	}()
	return nil
}

func (s *UDPServer) Stop(ctx context.Context) error {
	s.log.Info("[UDP] server stopping")
	return s.conn.Close()
}
