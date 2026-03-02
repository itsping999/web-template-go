package server

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/tx7do/kratos-transport/transport/mqtt"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/wyuhsin/web-template-go/internal/service"
)

type MQTTConfig struct {
	Addr string `json:"addr" yaml:"addr"`
}

// NewMQTTServer create a mqtt server.
func NewMQTTServer(
	c *Config,
	_ *logrus.Entry,
	svc *service.GreeterService,
) *mqtt.Server {
	ctx := context.Background()

	srv := mqtt.NewServer(
		mqtt.WithAddress([]string{c.MQTT.Addr}),
		mqtt.WithCodec("json"),
	)

	_ = srv.RegisterSubscriber(ctx,
		"/hfp/v2/journey/ongoing/vp/bus/#",
		func(ctx context.Context, evt broker.Event) error {
			switch t := evt.Message().Body.(type) {
			case any:
				return nil
			default:
				return fmt.Errorf("unsupported type: %T", t)
			}
		},
		func() broker.Any { return struct{}{} },
	)

	return srv
}
