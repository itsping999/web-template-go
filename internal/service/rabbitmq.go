package service

import (
	"context"

	"github.com/tx7do/kratos-transport/broker"
)

type RabbitMessage struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

func (s *GreeterService) OnRabbitMQMessage(
	_ context.Context,
	topic string,
	headers broker.Headers,
	msg *RabbitMessage,
) error {
	if msg == nil {
		s.log.Warnf("rabbitmq message is nil, topic=%s", topic)
		return nil
	}
	s.log.Infof("rabbitmq recv topic=%s headers=%v type=%s payload=%s", topic, headers, msg.Type, msg.Payload)
	return nil
}
