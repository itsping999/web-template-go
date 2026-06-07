package tests

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/wyuhsin/web-template-go/internal/biz"
	"github.com/wyuhsin/web-template-go/internal/data"
	"github.com/wyuhsin/web-template-go/internal/pkg/messagingx"
	"github.com/wyuhsin/web-template-go/internal/server"
)

func TestNewGreeterEventPublisherDisabled(t *testing.T) {
	client, cleanupClient, err := messagingx.NewRabbitMQPublisher(
		server.RabbitMQConfig{Enabled: false},
		log.NewStdLogger(io.Discard),
	)
	if err != nil {
		t.Fatalf("expected nil error when disabled, got %v", err)
	}
	cleanupClient()

	publisher, cleanup, err := data.NewGreeterEventPublisher(client, log.NewStdLogger(io.Discard))
	if err != nil {
		t.Fatalf("expected nil error when disabled, got %v", err)
	}
	if cleanup == nil {
		t.Fatalf("expected non-nil cleanup")
	}
	cleanup()

	if err := publisher.PublishGreeterCreated(context.Background(), &biz.Greeter{Name: "demo", Message: "Hello demo", Source: "local"}); err != nil {
		t.Fatalf("expected noop publisher to return nil error, got %v", err)
	}
}
