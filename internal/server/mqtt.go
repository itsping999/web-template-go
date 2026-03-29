package server

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/transport/mqtt"

	"github.com/tx7do/kratos-transport/broker"
)

type MQTTConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Addr    string `json:"addr" yaml:"addr"`
	Topic   string `json:"topic" yaml:"topic"`
}

// NewMQTTServer create a mqtt server.
func NewMQTTServer(
	c *MQTTConfig,
	logger log.Logger,
) *mqtt.Server {
	helper := log.NewHelper(log.With(logger, "module", "server/mqtt"))
	if c == nil || !c.Enabled {
		helper.Info("mqtt server disabled, skip initialization")
		return nil
	}
	addr := strings.TrimSpace(c.Addr)
	topic := strings.TrimSpace(c.Topic)
	if addr == "" || topic == "" {
		helper.Error("mqtt server config invalid: addr/topic must be set when enabled")
		return nil
	}
	ctx := context.Background()

	srv := mqtt.NewServer(
		mqtt.WithAddress([]string{addr}),
		mqtt.WithCodec("json"),
	)

	_ = srv.RegisterSubscriber(ctx,
		topic,
		func(context.Context, broker.Event) error {
			return nil
		},
		func() broker.Any { return struct{}{} },
	)

	return srv
}
