package messagingx

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
	brokerRabbitMQ "github.com/tx7do/kratos-transport/broker/rabbitmq"
	"github.com/wyuhsin/web-template-go/internal/server"
)

type RabbitMQPublisher struct {
	mu      sync.RWMutex
	clients map[string]broker.Broker
	addr    string
	logger  *log.Helper
}

func NewRabbitMQPublisher(cfg server.RabbitMQConfig, logger log.Logger) (*RabbitMQPublisher, func(), error) {
	helper := log.NewHelper(log.With(logger, "module", "messagingx/rabbitmq"))
	if !cfg.Enabled {
		helper.Info("rabbitmq publisher disabled, skip initialization")
		return nil, func() {}, nil
	}
	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		return nil, nil, errors.New("rabbitmq.addr is empty")
	}
	helper.Infow("msg", "rabbitmq publisher ready", "addr", addr)
	clients := make(map[string]broker.Broker)
	cleanup := func() {
		for exchange, c := range clients {
			if err := c.Disconnect(); err != nil {
				helper.Warnf("disconnect rabbitmq publisher failed: exchange=%s err=%v", exchange, err)
			}
		}
	}
	return &RabbitMQPublisher{
		clients: clients,
		addr:    addr,
		logger:  helper,
	}, cleanup, nil
}

func (p *RabbitMQPublisher) Publish(ctx context.Context, exchange, routingKey string, msg any) error {
	if p == nil {
		return nil
	}
	exchange = strings.TrimSpace(exchange)
	if exchange == "" {
		return errors.New("exchange is empty")
	}
	routingKey = strings.TrimSpace(routingKey)
	if routingKey == "" {
		return errors.New("routingKey is empty")
	}
	c, err := p.getOrCreateClient(exchange)
	if err != nil {
		return err
	}
	if err := c.Publish(ctx, routingKey, msg); err != nil {
		return err
	}
	return nil
}

func (p *RabbitMQPublisher) getOrCreateClient(exchange string) (broker.Broker, error) {
	p.mu.RLock()
	c, ok := p.clients[exchange]
	p.mu.RUnlock()
	if ok {
		return c, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok = p.clients[exchange]; ok {
		return c, nil
	}
	var err error
	c, err = buildRabbitMQBroker(p.addr, exchange)
	if err != nil {
		return nil, err
	}
	p.clients[exchange] = c
	p.logger.Infow("msg", "rabbitmq publisher exchange connected", "addr", p.addr, "exchange", exchange)
	return c, nil
}

func buildRabbitMQBroker(addr, exchange string) (broker.Broker, error) {
	b := brokerRabbitMQ.NewBroker(
		broker.WithAddress(addr),
		broker.WithCodec("json"),
		brokerRabbitMQ.WithExchangeName(exchange),
		brokerRabbitMQ.WithDurableExchange(),
	)
	if err := b.Init(); err != nil {
		return nil, err
	}
	if err := b.Connect(); err != nil {
		return nil, err
	}
	return b, nil
}
