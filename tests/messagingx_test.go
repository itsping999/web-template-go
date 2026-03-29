package tests

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/wyuhsin/web-template-go/internal/pkg/messagingx"
	"github.com/wyuhsin/web-template-go/internal/server"
)

func TestNewRabbitMQPublisherDisabled(t *testing.T) {
	client, cleanup, err := messagingx.NewRabbitMQPublisher(server.RabbitMQConfig{Enabled: false}, log.NewStdLogger(io.Discard))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cleanup == nil {
		t.Fatalf("expected non-nil cleanup when disabled")
	}
	cleanup()
	if client != nil {
		t.Fatalf("expected nil publisher when disabled")
	}
}

func TestNewRabbitMQPublisherMissingRequiredConfig(t *testing.T) {
	_, _, err := messagingx.NewRabbitMQPublisher(server.RabbitMQConfig{Enabled: true}, log.NewStdLogger(io.Discard))
	if err == nil {
		t.Fatalf("expected error when rabbitmq client config missing")
	}
}

func TestRabbitMQPublisherPublishToNilSafe(t *testing.T) {
	var publisher *messagingx.RabbitMQPublisher
	if err := publisher.Publish(context.Background(), "ex.any", "rk.any", map[string]any{"k": "v"}); err != nil {
		t.Fatalf("expected nil error for nil publisher, got %v", err)
	}
}

func TestRabbitMQPublisherPublishRequiresExplicitTarget(t *testing.T) {
	var publisher messagingx.RabbitMQPublisher
	if err := publisher.Publish(context.Background(), "", "rk", map[string]any{"k": "v"}); err == nil {
		t.Fatalf("expected error when using Publish without exchange/routingKey")
	}
}

func TestNewMQTTPublisherDisabled(t *testing.T) {
	client, cleanup, err := messagingx.NewMQTTPublisher(server.MQTTConfig{Enabled: false}, log.NewStdLogger(io.Discard))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cleanup == nil {
		t.Fatalf("expected non-nil cleanup when disabled")
	}
	cleanup()
	if client != nil {
		t.Fatalf("expected nil publisher when disabled")
	}
}

func TestNewMQTTPublisherMissingAddr(t *testing.T) {
	_, _, err := messagingx.NewMQTTPublisher(server.MQTTConfig{Enabled: true}, log.NewStdLogger(io.Discard))
	if err == nil {
		t.Fatalf("expected error when mqtt client addr missing")
	}
}
