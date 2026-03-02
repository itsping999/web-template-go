package server

import (
	"context"
	"net"
	"time"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/sirupsen/logrus"
)

var (
	_ transport.Server = (*TCPServer)(nil)
)

type TCPConfig struct {
	Addr    string        `json:"addr" yaml:"addr"`
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
}

type TCPServer struct {
	lis net.Listener
	log *logrus.Entry
	c   *TCPConfig
}

func NewTCPServer(c *TCPConfig, logger *logrus.Entry) *TCPServer {
	srv := &TCPServer{
		log: logger.WithField("module", "server/tcp"),
		c:   c,
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
