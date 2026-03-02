//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/wyuhsin/web-template-go/internal/biz"
	"github.com/wyuhsin/web-template-go/internal/data"
	"github.com/wyuhsin/web-template-go/internal/pkg/dbx"
	"github.com/wyuhsin/web-template-go/internal/pkg/tracingx"
	"github.com/wyuhsin/web-template-go/internal/server"
	"github.com/wyuhsin/web-template-go/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"
	"github.com/sirupsen/logrus"
)

// wireApp init kratos application.
func wireApp(*server.Config, *data.Config, *tracingx.Config, *logrus.Entry) (*kratos.App, func(), error) {
	panic(
		wire.Build(
			wire.FieldsOf(new(*server.Config), "TCP", "UDP", "HTTP", "GRPC"),
			wire.FieldsOf(new(*data.Config), "MySQL", "Redis"),
			server.ProviderSet,
			dbx.ProviderSet,
			data.ProviderSet,
			biz.ProviderSet,
			service.ProviderSet,
			newApp,
		),
	)
}
