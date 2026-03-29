//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/wyuhsin/web-template-go/internal/biz"
	"github.com/wyuhsin/web-template-go/internal/data"
	provider "github.com/wyuhsin/web-template-go/internal/pkg"
	"github.com/wyuhsin/web-template-go/internal/pkg/tracingx"
	"github.com/wyuhsin/web-template-go/internal/server"
	"github.com/wyuhsin/web-template-go/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*server.Config, *data.Config, *tracingx.Config, log.Logger) (*kratos.App, func(), error) {
	panic(
		wire.Build(
			wire.FieldsOf(new(*server.Config), "HTTP", "GRPC", "Middleware", "WebSocket", "MQTT", "RabbitMQ"),
			wire.FieldsOf(new(*data.Config), "Postgres", "Redis", "MongoDB", "Discovery", "RemoteGRPC"),
			server.ProviderSet,
			provider.ProviderSet,
			data.ProviderSet,
			biz.ProviderSet,
			service.ProviderSet,
			newApp,
		),
	)
}
