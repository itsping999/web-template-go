package server

import (
	"time"

	"github.com/wyuhsin/web-template-go/api/helloworld/v1"
	"github.com/wyuhsin/web-template-go/internal/pkg/tracingx"
	"github.com/wyuhsin/web-template-go/internal/service"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/sirupsen/logrus"
)

type GRPCConfig struct {
	Network string        `json:"network" yaml:"network"`
	Addr    string        `json:"addr" yaml:"addr"`
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
}

// NewGRPCServer new a gRPC server.
func NewGRPCServer(
	c *GRPCConfig,
	tracingCfg *tracingx.Config,
	logger *logrus.Entry,
	greeter *service.GreeterService,
) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(commonMiddlewares(logger, tracingCfg)...),
	}
	if c.Network != "" {
		opts = append(opts, grpc.Network(c.Network))
	}
	if c.Addr != "" {
		opts = append(opts, grpc.Address(c.Addr))
	}
	if c.Timeout > 0 {
		opts = append(opts, grpc.Timeout(c.Timeout))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterGreeterServer(srv, greeter)
	return srv
}
