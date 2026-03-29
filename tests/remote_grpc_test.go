package tests

import (
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/wyuhsin/web-template-go/internal/pkg/grpcx"
)

func TestNewGreeterClientDisabled(t *testing.T) {
	client, cleanup, err := grpcx.NewGreeterClient(grpcx.Config{
		Greeter: grpcx.ServiceConfig{Enabled: false},
	}, log.NewStdLogger(io.Discard))
	if err != nil {
		t.Fatalf("expected nil error when disabled, got %v", err)
	}
	if client != nil {
		t.Fatalf("expected nil client when disabled")
	}
	if cleanup == nil {
		t.Fatalf("expected non-nil cleanup")
	}
	cleanup()
}

func TestNewGreeterClientEnabledWithoutTarget(t *testing.T) {
	_, _, err := grpcx.NewGreeterClient(grpcx.Config{
		Greeter: grpcx.ServiceConfig{Enabled: true},
	}, log.NewStdLogger(io.Discard))
	if err == nil {
		t.Fatalf("expected error when target is empty")
	}
}

func TestNewGreeterClientEnabledWithoutTimeout(t *testing.T) {
	_, _, err := grpcx.NewGreeterClient(grpcx.Config{
		Greeter: grpcx.ServiceConfig{Enabled: true, Target: "127.0.0.1:9000"},
	}, log.NewStdLogger(io.Discard))
	if err == nil {
		t.Fatalf("expected error when timeout is not configured")
	}
}
