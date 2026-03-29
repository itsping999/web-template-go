package grpcx

import (
	"context"
	"errors"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/circuitbreaker"
	kratosmetrics "github.com/go-kratos/kratos/v2/middleware/metrics"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	v1 "github.com/wyuhsin/web-template-go/api/helloworld/v1"
	"go.opentelemetry.io/otel"
)

func NewGreeterClient(cfg Config, logger log.Logger) (v1.GreeterClient, func(), error) {
	helper := log.NewHelper(log.With(logger, "module", "grpcx/remote"))
	if !cfg.Greeter.Enabled {
		helper.Info("remote grpc client disabled, skip initialization")
		return nil, func() {}, nil
	}
	if cfg.Greeter.Target == "" {
		return nil, nil, errors.New("data.remote_grpc.greeter.target is required when enabled")
	}

	timeout := cfg.Greeter.Timeout
	if timeout <= 0 {
		return nil, nil, errors.New("data.remote_grpc.greeter.timeout must be > 0 when enabled")
	}

	metricsMwOpts := make([]kratosmetrics.Option, 0, 2)
	meter := otel.Meter("grpcx/remote")
	requestsCounter, err := kratosmetrics.DefaultRequestsCounter(meter, kratosmetrics.DefaultClientRequestsCounterName)
	if err != nil {
		helper.Warnf("init client metrics request counter failed: %v", err)
	} else {
		metricsMwOpts = append(metricsMwOpts, kratosmetrics.WithRequests(requestsCounter))
	}
	secondsHistogram, err := kratosmetrics.DefaultSecondsHistogram(meter, kratosmetrics.DefaultClientSecondsHistogramName)
	if err != nil {
		helper.Warnf("init client metrics seconds histogram failed: %v", err)
	} else {
		metricsMwOpts = append(metricsMwOpts, kratosmetrics.WithSeconds(secondsHistogram))
	}

	conn, err := kgrpc.DialInsecure(
		context.Background(),
		kgrpc.WithEndpoint(cfg.Greeter.Target),
		kgrpc.WithTimeout(timeout),
		kgrpc.WithMiddleware(buildClientMiddlewares(cfg, metricsMwOpts...)...),
	)
	if err != nil {
		return nil, nil, err
	}
	helper.Infow("msg", "remote grpc client connected", "service", "greeter", "target", cfg.Greeter.Target)

	cleanup := func() {
		if err := conn.Close(); err != nil {
			helper.Warnf("close remote grpc client failed: %v", err)
		}
	}
	return v1.NewGreeterClient(conn), cleanup, nil
}

func buildClientMiddlewares(cfg Config, metricsOpts ...kratosmetrics.Option) []middleware.Middleware {
	mws := []middleware.Middleware{
		kratosmetrics.Client(metricsOpts...),
	}
	if cfg.Greeter.CircuitBreakerEnabled {
		mws = append(mws, circuitbreaker.Client())
	}
	return mws
}
