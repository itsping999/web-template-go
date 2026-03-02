package server

import (
	"time"

	"github.com/wyuhsin/web-template-go/api/helloworld/v1"
	"github.com/wyuhsin/web-template-go/internal/pkg/tracingx"
	"github.com/wyuhsin/web-template-go/internal/service"

	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/sirupsen/logrus"
)

type HTTPConfig struct {
	Network string        `json:"network" yaml:"network"`
	Addr    string        `json:"addr" yaml:"addr"`
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
}

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	c *HTTPConfig,
	tracingCfg *tracingx.Config,
	logger *logrus.Entry,
	greeter *service.GreeterService,
	system *service.SystemService,
) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(commonMiddlewares(logger, tracingCfg)...),
	}
	if c.Network != "" {
		opts = append(opts, http.Network(c.Network))
	}
	if c.Addr != "" {
		opts = append(opts, http.Address(c.Addr))
	}
	if c.Timeout > 0 {
		opts = append(opts, http.Timeout(c.Timeout))
	}
	srv := http.NewServer(opts...)
	v1.RegisterGreeterHTTPServer(srv, greeter)
	r := srv.Route("/")
	r.GET("/healthz", func(ctx http.Context) error {
		return ctx.Result(200, system.Liveness())
	})
	r.GET("/readyz", func(ctx http.Context) error {
		return ctx.Result(200, system.Readiness())
	})
	r.GET("/meta", func(ctx http.Context) error {
		return ctx.Result(200, system.Metadata())
	})

	return srv
}
