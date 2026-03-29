package server

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
	brokerRabbitMQ "github.com/tx7do/kratos-transport/broker/rabbitmq"
	transportRabbitMQ "github.com/tx7do/kratos-transport/transport/rabbitmq"
	"github.com/wyuhsin/web-template-go/internal/service"
)

type RabbitMQConfig struct {
	Enabled    bool   `json:"enabled" yaml:"enabled"`
	Addr       string `json:"addr" yaml:"addr"`
	Exchange   string `json:"exchange" yaml:"exchange"`
	RoutingKey string `json:"routing_key" yaml:"routing_key"`
	Queue      string `json:"queue" yaml:"queue"`
}

func NewRabbitMQServer(
	c *RabbitMQConfig,
	logger log.Logger,
	greeter *service.GreeterService,
) *transportRabbitMQ.Server {
	helper := log.NewHelper(log.With(logger, "module", "server/rabbitmq"))
	if c == nil || !c.Enabled {
		helper.Info("rabbitmq server disabled, skip initialization")
		return nil
	}
	addr := strings.TrimSpace(c.Addr)
	exchange := strings.TrimSpace(c.Exchange)
	routingKey := strings.TrimSpace(c.RoutingKey)
	queue := strings.TrimSpace(c.Queue)
	if addr == "" || exchange == "" || routingKey == "" || queue == "" {
		helper.Error("rabbitmq server config invalid: addr/exchange/routing_key/queue must be set when enabled")
		return nil
	}

	srv := transportRabbitMQ.NewServer(
		transportRabbitMQ.WithAddress([]string{addr}),
		transportRabbitMQ.WithExchange(exchange, true),
		transportRabbitMQ.WithCodec("json"),
	)

	err := transportRabbitMQ.RegisterSubscriber(srv, context.Background(),
		routingKey,
		func(ctx context.Context, topic string, headers broker.Headers, msg *service.RabbitMessage) error {
			if greeter == nil {
				return nil
			}
			return greeter.OnRabbitMQMessage(ctx, topic, headers, msg)
		},
		broker.WithQueueName(queue),
		brokerRabbitMQ.WithDurableQueue(),
	)
	if err != nil {
		helper.Errorf("register rabbitmq subscriber failed: %v", err)
		return nil
	}

	return srv
}
