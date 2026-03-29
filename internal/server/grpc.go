package server

import (
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/wyuhsin/web-template-go/api/helloworld/v1"
	"github.com/wyuhsin/web-template-go/internal/pkg/tracingx"
	"github.com/wyuhsin/web-template-go/internal/service"

	"github.com/go-kratos/kratos/v2/transport/grpc"
)

type GRPCConfig struct {
	Enabled bool          `json:"enabled" yaml:"enabled"`
	Network string        `json:"network" yaml:"network"`
	Addr    string        `json:"addr" yaml:"addr"`
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
}

// NewGRPCServer new a gRPC server.
func NewGRPCServer(
	c *GRPCConfig,
	middlewareCfg *MiddlewareConfig,
	tracingCfg *tracingx.Config,
	logger log.Logger,
	greeter *service.GreeterService,
) *grpc.Server {
	helper := log.NewHelper(log.With(logger, "module", "server/grpc"))
	if c == nil || !c.Enabled {
		helper.Info("grpc server disabled, skip initialization")
		return nil
	}
	var opts = []grpc.ServerOption{
		grpc.Middleware(commonMiddlewares(logger, tracingCfg, middlewareCfg)...),
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
