package server

import (
	"github.com/wyuhsin/web-template-go/internal/pkg/logx"
	"github.com/wyuhsin/web-template-go/internal/pkg/tracingx"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/sirupsen/logrus"
)

func commonMiddlewares(logger *logrus.Entry, tracingCfg *tracingx.Config) []middleware.Middleware {
	mws := []middleware.Middleware{
		recovery.Recovery(),
		metadata.Server(),
		logging.Server(logx.NewKratosLogger(logger.WithField("module", "server/middleware"))),
		validate.Validator(),
	}
	if tracingCfg != nil && tracingCfg.Enabled {
		mws = append([]middleware.Middleware{tracing.Server()}, mws...)
	}
	return mws
}
