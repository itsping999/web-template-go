package server

import (
	"github.com/wyuhsin/web-template-go/internal/conf"

	"github.com/google/wire"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(
	NewGRPCServer,
	NewHTTPServer,
	NewWebsocketServer,
	NewMQTTServer,
	NewRabbitMQServer,
	NewTCPServer,
	NewUDPServer,

	wire.FieldsOf(new(*conf.Server), "Http"),
	wire.FieldsOf(new(*conf.Server), "Grpc"),
)
