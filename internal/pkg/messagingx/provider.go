package messagingx

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewMQTTPublisher,
	NewRabbitMQPublisher,
)
