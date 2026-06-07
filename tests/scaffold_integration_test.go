package tests

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wyuhsin/web-template-go/internal/scaffold"
)

// TestScaffoldIntegration validates the full scaffold workflow
// including auto-update of provider sets.
func TestScaffoldIntegration(t *testing.T) {
	root := t.TempDir()

	// Set up a minimal project structure
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/example/test\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"internal/service", "internal/biz", "internal/data"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Create provider set files
	providerFiles := map[string]string{
		"internal/service/service.go": `package service

import "github.com/google/wire"

var ProviderSet = wire.NewSet()
`,
		"internal/biz/biz.go": `package biz

import "github.com/google/wire"

var ProviderSet = wire.NewSet()
`,
		"internal/data/data.go": `package data

import "github.com/google/wire"

var ProviderSet = wire.NewSet()
`,
	}
	for path, content := range providerFiles {
		if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	runner := &integrationRunner{t: t, root: root}
	result, err := scaffold.RunProto(context.Background(), scaffold.ProtoOptions{
		RootDir:          root,
		Proto:            "order/v1/order.proto",
		ServiceTargetDir: "internal/service",
		Runner:           runner,
	})
	if err != nil {
		t.Fatalf("RunProto() error = %v", err)
	}

	// Verify proto was created
	if _, err := os.Stat(result.ProtoPath); err != nil {
		t.Fatalf("proto file not created: %v", err)
	}

	// Verify biz stub has domain model and error
	bizContent, err := os.ReadFile(result.BizPath)
	if err != nil {
		t.Fatal(err)
	}
	bizStr := string(bizContent)
	for _, want := range []string{"type Order struct", "ErrOrderNotFound", "type OrderRepo interface"} {
		if !strings.Contains(bizStr, want) {
			t.Errorf("biz stub missing %q:\n%s", want, bizStr)
		}
	}

	// Verify provider sets auto-updated
	svcContent, _ := os.ReadFile(filepath.Join(root, "internal/service/service.go"))
	if !strings.Contains(string(svcContent), "NewOrderService") {
		t.Errorf("service provider set not updated with NewOrderService")
	}
	bizProv, _ := os.ReadFile(filepath.Join(root, "internal/biz/biz.go"))
	if !strings.Contains(string(bizProv), "NewOrderUsecase") {
		t.Errorf("biz provider set not updated with NewOrderUsecase")
	}
	dataProv, _ := os.ReadFile(filepath.Join(root, "internal/data/data.go"))
	if !strings.Contains(string(dataProv), "NewOrderRepo") {
		t.Errorf("data provider set not updated with NewOrderRepo")
	}

	// Verify next steps are concise (no provider set manual steps)
	for _, step := range result.NextSteps {
		if strings.Contains(step, "ProviderSet") {
			t.Errorf("next steps should not mention ProviderSet (auto-updated): %s", step)
		}
	}
}

type integrationRunner struct {
	t    *testing.T
	root string
}

func (r *integrationRunner) Run(_ context.Context, dir, name string, args ...string) error {
	if name == "kratos" && strings.Join(args, " ") == "proto add order/v1/order.proto" {
		protoPath := filepath.Join(dir, "order", "v1", "order.proto")
		if err := os.MkdirAll(filepath.Dir(protoPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(protoPath, []byte(`syntax = "proto3";
package order.v1;
option go_package = "github.com/example/test/api/order/v1;v1";
`), 0o644)
	}
	if name == "kratos" && strings.Contains(strings.Join(args, " "), "proto server") {
		return os.WriteFile(filepath.Join(r.root, "internal/service/order.go"), []byte("package service\n"), 0o644)
	}
	return nil
}
