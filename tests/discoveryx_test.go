package tests

import (
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/wyuhsin/web-template-go/internal/pkg/discoveryx"
)

func TestNewKubernetesRegistryDisabled(t *testing.T) {
	reg, cleanup, err := discoveryx.NewKubernetesRegistrar(discoveryx.Config{
		Kubernetes: discoveryx.KubernetesConfig{Enabled: false},
	}, log.NewStdLogger(io.Discard))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cleanup == nil {
		t.Fatalf("expected non-nil cleanup when disabled")
	}
	if reg != nil {
		t.Fatalf("expected nil registrar when disabled")
	}
	cleanup()
}

func TestNewKubernetesRegistryInvalidMode(t *testing.T) {
	_, _, err := discoveryx.NewKubernetesRegistrar(discoveryx.Config{
		Kubernetes: discoveryx.KubernetesConfig{
			Enabled: true,
			Mode:    "invalid",
		},
	}, log.NewStdLogger(io.Discard))
	if err == nil {
		t.Fatalf("expected mode validation error")
	}
}

func TestNewKubernetesRegistryMissingKubeconfig(t *testing.T) {
	_, _, err := discoveryx.NewKubernetesRegistrar(discoveryx.Config{
		Kubernetes: discoveryx.KubernetesConfig{
			Enabled: true,
			Mode:    "kubeconfig",
		},
	}, log.NewStdLogger(io.Discard))
	if err == nil {
		t.Fatalf("expected kubeconfig validation error")
	}
}
