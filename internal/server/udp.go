package server

import (
	"context"
	"net"
	"time"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/sirupsen/logrus"
)

type UDPConfig struct {
	Addr    string        `json:"addr" yaml:"addr"`
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
}

var (
	_ transport.Server = (*UDPServer)(nil)
)

type UDPServer struct {
	conn *net.UDPConn
	log  *logrus.Entry
	c    *UDPConfig
}

func NewUDPServer(c *UDPConfig, logger *logrus.Entry) *UDPServer {
	srv := &UDPServer{
		log: logger.WithField("module", "server/udp"),
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
