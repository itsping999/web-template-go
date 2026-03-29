package messagingx

import (
	"context"
	"errors"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
	brokerMQTT "github.com/tx7do/kratos-transport/broker/mqtt"
	"github.com/wyuhsin/web-template-go/internal/server"
)

type MQTTPublisher struct {
	client broker.Broker
	logger *log.Helper
}

func NewMQTTPublisher(cfg server.MQTTConfig, logger log.Logger) (*MQTTPublisher, func(), error) {
	helper := log.NewHelper(log.With(logger, "module", "messagingx/mqtt"))
	if !cfg.Enabled {
		helper.Info("mqtt publisher disabled, skip initialization")
		return nil, func() {}, nil
	}
	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		return nil, nil, errors.New("mqtt.addr is empty")
	}
	b := brokerMQTT.NewBroker(
		broker.WithAddress(addr),
		broker.WithCodec("json"),
	)
	if err := b.Init(); err != nil {
		return nil, nil, err
	}
	if err := b.Connect(); err != nil {
		return nil, nil, err
	}
	helper.Infow("msg", "mqtt publisher connected", "addr", addr)
	cleanup := func() {
		if err := b.Disconnect(); err != nil {
			helper.Warnf("disconnect mqtt publisher failed: %v", err)
		}
	}
	return &MQTTPublisher{
		client: b,
		logger: helper,
	}, cleanup, nil
}

func (p *MQTTPublisher) Publish(ctx context.Context, topic string, msg any) error {
	if p == nil || p.client == nil {
		return nil
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return errors.New("topic is empty")
	}
	return p.client.Publish(ctx, topic, msg)
}
