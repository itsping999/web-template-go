package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/wyuhsin/web-template-go/internal/biz"
	"github.com/wyuhsin/web-template-go/internal/pkg/messagingx"
)

const (
	greeterCreatedExchange   = "server.greeter.exchange"
	greeterCreatedRoutingKey = "server.greeter.routingkey"
)

type rabbitMQGreeterPublisher struct {
	client *messagingx.RabbitMQPublisher
	log    *log.Helper
}

type noopGreeterPublisher struct{}

type GreeterCreatedEvent struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

func (noopGreeterPublisher) PublishGreeterCreated(context.Context, *biz.Greeter) error {
	return nil
}

func NewGreeterEventPublisher(client *messagingx.RabbitMQPublisher, logger log.Logger) (biz.GreeterEventPublisher, func(), error) {
	helper := log.NewHelper(log.With(logger, "module", "data/rabbitmq-publisher"))
	if client == nil {
		helper.Info("rabbitmq publisher disabled, skip initialization")
		return noopGreeterPublisher{}, func() {}, nil
	}
	return &rabbitMQGreeterPublisher{
		client: client,
		log:    helper,
	}, func() {}, nil
}

func (p *rabbitMQGreeterPublisher) PublishGreeterCreated(ctx context.Context, g *biz.Greeter) error {
	if g == nil {
		return nil
	}
	evt := GreeterCreatedEvent{
		Type:    "greeter.created",
		Payload: g.Hello,
	}
	if err := p.client.Publish(ctx, greeterCreatedExchange, greeterCreatedRoutingKey, evt); err != nil {
		return err
	}
	p.log.WithContext(ctx).Infof("published rabbitmq event type=%s payload=%s", evt.Type, evt.Payload)
	return nil
}
