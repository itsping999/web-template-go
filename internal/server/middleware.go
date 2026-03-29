package server

import (
	"github.com/wyuhsin/web-template-go/internal/pkg/metricsx"
	"github.com/wyuhsin/web-template-go/internal/pkg/tracingx"

	validateV2 "github.com/go-kratos/kratos/contrib/middleware/validate/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	kratosmetrics "github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"go.opentelemetry.io/otel"
)

func commonMiddlewares(logger log.Logger, tracingCfg *tracingx.Config, middlewareCfg *MiddlewareConfig) []middleware.Middleware {
	if _, err := metricsx.Init(logger); err != nil {
		log.NewHelper(log.With(logger, "module", "server/middleware")).Warnf("init metrics failed: %v", err)
	}

	helper := log.NewHelper(log.With(logger, "module", "server/middleware"))
	metricsMwOpts := make([]kratosmetrics.Option, 0, 2)
	meter := otel.Meter("server")
	requestsCounter, err := kratosmetrics.DefaultRequestsCounter(meter, kratosmetrics.DefaultServerRequestsCounterName)
	if err != nil {
		helper.Warnf("init server metrics request counter failed: %v", err)
	} else {
		metricsMwOpts = append(metricsMwOpts, kratosmetrics.WithRequests(requestsCounter))
	}
	secondsHistogram, err := kratosmetrics.DefaultSecondsHistogram(meter, kratosmetrics.DefaultServerSecondsHistogramName)
	if err != nil {
		helper.Warnf("init server metrics seconds histogram failed: %v", err)
	} else {
		metricsMwOpts = append(metricsMwOpts, kratosmetrics.WithSeconds(secondsHistogram))
	}

	mws := []middleware.Middleware{
		recovery.Recovery(),
		metadata.Server(),
		kratosmetrics.Server(metricsMwOpts...),
	}
	if middlewareCfg != nil && middlewareCfg.RateLimitEnabled {
		mws = append(mws, ratelimit.Server())
	}
	mws = append(mws,
		logging.Server(logger),
		validateV2.ProtoValidate(),
	)
	if tracingCfg != nil && tracingCfg.Enabled {
		mws = append([]middleware.Middleware{tracing.Server()}, mws...)
	}
	return mws
}
