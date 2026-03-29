package tests

import (
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/wyuhsin/web-template-go/internal/server"
)

func TestNewMQTTServerDisabled(t *testing.T) {
	logger := log.NewStdLogger(io.Discard)
	srv := server.NewMQTTServer(&server.MQTTConfig{Enabled: false}, logger)
	if srv != nil {
		t.Fatalf("expected nil mqtt server when disabled")
	}
}

func TestNewRabbitMQServerDisabled(t *testing.T) {
	logger := log.NewStdLogger(io.Discard)
	srv := server.NewRabbitMQServer(&server.RabbitMQConfig{Enabled: false}, logger, nil)
	if srv != nil {
		t.Fatalf("expected nil rabbitmq server when disabled")
	}
}

func TestNewWebsocketServerDisabled(t *testing.T) {
	srv := server.NewWebsocketServer(&server.WebSocketConfig{Enabled: false}, nil, nil)
	if srv != nil {
		t.Fatalf("expected nil websocket server when disabled")
	}
}
