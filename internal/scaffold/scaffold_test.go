package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeProtoPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "plain api relative", input: "demo/v1/demo.proto", want: "demo/v1/demo.proto"},
		{name: "strip api prefix", input: "api/demo/v1/demo.proto", want: "demo/v1/demo.proto"},
		{name: "strip dot api prefix", input: "./api/demo/v1/demo.proto", want: "demo/v1/demo.proto"},
		{name: "reject empty", input: " ", wantErr: true},
		{name: "reject traversal", input: "../demo/v1/demo.proto", wantErr: true},
		{name: "reject absolute", input: "/tmp/demo.proto", wantErr: true},
		{name: "reject non proto", input: "demo/v1/demo.txt", wantErr: true},
		{name: "reject shallow proto", input: "demo.proto", wantErr: true},
		{name: "reject empty proto name", input: "demo/v1/.proto", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeProtoPath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeProtoPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGoPackageForProto(t *testing.T) {
	got, err := GoPackageForProto("github.com/example/app", "demo/v1/demo.proto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "github.com/example/app/api/demo/v1;v1"
	if got != want {
		t.Fatalf("GoPackageForProto() = %q, want %q", got, want)
	}
}

func TestReplaceGoPackage(t *testing.T) {
	input := []byte(`syntax = "proto3";

package demo.v1;

option go_package = "github.com/example/app/demo/v1;v1";
`)
	got, err := ReplaceGoPackage(input, "github.com/example/app/api/demo/v1;v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(got), `option go_package = "github.com/example/app/api/demo/v1;v1";`) {
		t.Fatalf("go_package was not replaced:\n%s", string(got))
	}
}

// setupTestRoot creates a temp directory with all files needed for scaffold testing.
func setupTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/example/app\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"internal/service", "internal/biz", "internal/data", "internal/server"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Provider set stubs
	if err := os.WriteFile(filepath.Join(root, "internal/service/service.go"), []byte(`package service

import "github.com/google/wire"

var ProviderSet = wire.NewSet()
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal/biz/biz.go"), []byte(`package biz

import "github.com/google/wire"

var ProviderSet = wire.NewSet()
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal/data/data.go"), []byte(`package data

import "github.com/google/wire"

var ProviderSet = wire.NewSet()
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Server stubs
	if err := os.WriteFile(filepath.Join(root, "internal/server/grpc.go"), []byte(`package server

import (
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

func NewGRPCServer() *grpc.Server {
	srv := grpc.NewServer()
	return srv
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal/server/http.go"), []byte(`package server

import (
	"github.com/go-kratos/kratos/v2/transport/http"
)

func NewHTTPServer() *http.Server {
	srv := http.NewServer()
	return srv
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Config stub
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs/config.yaml"), []byte(`server:
  http:
    enabled: true

data:
  remote_grpc:
    greeter:
      enabled: false
`), 0o644); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestRunProtoScaffold(t *testing.T) {
	root := setupTestRoot(t)

	runner := &fakeRunner{t: t, root: root}
	result, err := RunProto(context.Background(), ProtoOptions{
		RootDir:          root,
		Proto:            "demo/v1/demo.proto",
		ServiceTargetDir: "internal/service",
		Runner:           runner,
	})
	if err != nil {
		t.Fatalf("RunProto() error = %v", err)
	}

	wantCommands := []string{
		"api|kratos proto add demo/v1/demo.proto",
		".|kratos proto server api/demo/v1/demo.proto --target-dir=internal/service",
		".|make api",
		".|gofmt -w internal/service/demo.go",
	}
	if strings.Join(runner.commands, "\n") != strings.Join(wantCommands, "\n") {
		t.Fatalf("commands = %#v, want %#v", runner.commands, wantCommands)
	}
	if result.ProtoPath != filepath.Join(root, "api", "demo", "v1", "demo.proto") {
		t.Fatalf("unexpected proto path: %s", result.ProtoPath)
	}
	if result.ServicePath != filepath.Join(root, "internal", "service", "demo.go") {
		t.Fatalf("unexpected service path: %s", result.ServicePath)
	}
	if result.BizPath != filepath.Join(root, "internal", "biz", "demo.go") {
		t.Fatalf("unexpected biz path: %s", result.BizPath)
	}
	if result.DataPath != filepath.Join(root, "internal", "data", "demo.go") {
		t.Fatalf("unexpected data path: %s", result.DataPath)
	}

	// Verify biz stub content
	bizContent, err := os.ReadFile(result.BizPath)
	if err != nil {
		t.Fatal(err)
	}
	bizStr := string(bizContent)
	if !strings.Contains(bizStr, "type DemoRepo interface") {
		t.Fatalf("biz stub missing DemoRepo interface:\n%s", bizStr)
	}
	if !strings.Contains(bizStr, "type DemoUsecase struct") {
		t.Fatalf("biz stub missing DemoUsecase struct:\n%s", bizStr)
	}
	if !strings.Contains(bizStr, "ErrDemoNotFound") {
		t.Fatalf("biz stub missing ErrDemoNotFound:\n%s", bizStr)
	}
	if !strings.Contains(bizStr, "type Demo struct") {
		t.Fatalf("biz stub missing Demo domain model:\n%s", bizStr)
	}

	// Verify data stub content
	dataContent, err := os.ReadFile(result.DataPath)
	if err != nil {
		t.Fatal(err)
	}
	dataStr := string(dataContent)
	if !strings.Contains(dataStr, "type demoRepo struct") {
		t.Fatalf("data stub missing demoRepo struct:\n%s", dataStr)
	}
	if !strings.Contains(dataStr, "biz.DemoRepo") {
		t.Fatalf("data stub missing biz.DemoRepo reference:\n%s", dataStr)
	}
	if strings.Contains(dataStr, `"fmt"`) {
		t.Fatalf("non-CRUD data stub should not import fmt:\n%s", dataStr)
	}

	// Verify provider sets were auto-updated
	svcContent, err := os.ReadFile(filepath.Join(root, "internal/service/service.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(svcContent), "NewDemoService") {
		t.Fatalf("service provider set not updated:\n%s", string(svcContent))
	}
	bizProvContent, err := os.ReadFile(filepath.Join(root, "internal/biz/biz.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bizProvContent), "NewDemoUsecase") {
		t.Fatalf("biz provider set not updated:\n%s", string(bizProvContent))
	}
	dataProvContent, err := os.ReadFile(filepath.Join(root, "internal/data/data.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dataProvContent), "NewDemoRepo") {
		t.Fatalf("data provider set not updated:\n%s", string(dataProvContent))
	}

	// Verify auto-registration in server files
	grpcContent, err := os.ReadFile(filepath.Join(root, "internal/server/grpc.go"))
	if err != nil {
		t.Fatal(err)
	}
	grpcStr := string(grpcContent)
	if !strings.Contains(grpcStr, "RegisterDemoServer(srv, demo)") {
		t.Fatalf("gRPC server not auto-registered:\n%s", grpcStr)
	}
	if !strings.Contains(grpcStr, `v1 "github.com/example/app/api/demo/v1"`) {
		t.Fatalf("gRPC server missing import:\n%s", grpcStr)
	}

	httpContent, err := os.ReadFile(filepath.Join(root, "internal/server/http.go"))
	if err != nil {
		t.Fatal(err)
	}
	httpStr := string(httpContent)
	if !strings.Contains(httpStr, "RegisterDemoHTTPServer(srv, demo)") {
		t.Fatalf("HTTP server not auto-registered:\n%s", httpStr)
	}

	// Verify config entry was injected
	configContent, err := os.ReadFile(filepath.Join(root, "configs/config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	configStr := string(configContent)
	if !strings.Contains(configStr, "demo:") {
		t.Fatalf("config entry not injected:\n%s", configStr)
	}
	if !strings.Contains(configStr, "target: \"demo:9000\"") {
		t.Fatalf("config target not injected:\n%s", configStr)
	}

	wantNextSteps := []string{
		"Implement DemoRepo interface methods in internal/data/demo.go.",
		"Implement DemoUsecase methods in internal/biz/demo.go.",
		"Implement service methods in internal/service/demo.go.",
		"Run make generate, then make verify.",
	}
	if strings.Join(result.NextSteps, "\n") != strings.Join(wantNextSteps, "\n") {
		t.Fatalf("next steps = %#v, want %#v", result.NextSteps, wantNextSteps)
	}

	content, err := os.ReadFile(result.ProtoPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `option go_package = "github.com/example/app/api/demo/v1;v1";`) {
		t.Fatalf("go_package not fixed:\n%s", string(content))
	}
}

func TestRunProtoCRUD(t *testing.T) {
	root := setupTestRoot(t)

	runner := &fakeRunner{t: t, root: root}
	result, err := RunProto(context.Background(), ProtoOptions{
		RootDir:          root,
		Proto:            "orders/v1/orders.proto",
		ServiceTargetDir: "internal/service",
		CRUD:             true,
		Runner:           runner,
	})
	if err != nil {
		t.Fatalf("RunProto() error = %v", err)
	}

	// Verify CRUD proto content
	protoContent, err := os.ReadFile(result.ProtoPath)
	if err != nil {
		t.Fatal(err)
	}
	protoStr := string(protoContent)
	for _, want := range []string{
		"rpc CreateOrder",
		"rpc GetOrder",
		"rpc ListOrders",
		"rpc UpdateOrder",
		"rpc DeleteOrder",
		"message Order {",
		"google.protobuf.Timestamp created_at",
	} {
		if !strings.Contains(protoStr, want) {
			t.Fatalf("CRUD proto missing %q:\n%s", want, protoStr)
		}
	}

	// Verify biz stub has CRUD methods
	bizContent, err := os.ReadFile(result.BizPath)
	if err != nil {
		t.Fatal(err)
	}
	bizStr := string(bizContent)
	for _, want := range []string{
		"Create(context.Context, *Order)",
		"Get(context.Context, string)",
		"List(context.Context, int32, int32)",
		"Update(context.Context, *Order)",
		"Delete(context.Context, string)",
		"func (uc *OrderUsecase) CreateOrder",
	} {
		if !strings.Contains(bizStr, want) {
			t.Fatalf("CRUD biz stub missing %q:\n%s", want, bizStr)
		}
	}

	// Verify data stub has CRUD method stubs
	dataContent, err := os.ReadFile(result.DataPath)
	if err != nil {
		t.Fatal(err)
	}
	dataStr := string(dataContent)
	for _, want := range []string{
		"func (r *orderRepo) Create(",
		"func (r *orderRepo) Get(",
		"func (r *orderRepo) List(",
		"func (r *orderRepo) Update(",
		"func (r *orderRepo) Delete(",
		`fmt.Errorf("not implemented")`,
		`"fmt"`,
	} {
		if !strings.Contains(dataStr, want) {
			t.Fatalf("CRUD data stub missing %q:\n%s", want, dataStr)
		}
	}
}

func TestRunProtoSkipAutoRegister(t *testing.T) {
	root := setupTestRoot(t)

	runner := &fakeRunner{t: t, root: root}
	result, err := RunProto(context.Background(), ProtoOptions{
		RootDir:          root,
		Proto:            "demo/v1/demo.proto",
		ServiceTargetDir: "internal/service",
		SkipAutoRegister: true,
		Runner:           runner,
	})
	if err != nil {
		t.Fatalf("RunProto() error = %v", err)
	}

	// Server files should NOT be modified
	grpcContent, err := os.ReadFile(filepath.Join(root, "internal/server/grpc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(grpcContent), "RegisterDemoServer") {
		t.Fatalf("gRPC server should not be auto-registered with SkipAutoRegister")
	}

	// Next steps should include manual registration
	if len(result.NextSteps) == 0 {
		t.Fatalf("expected next steps")
	}
	if !strings.Contains(result.NextSteps[0], "Register") {
		t.Fatalf("first next step should mention registration, got: %s", result.NextSteps[0])
	}
}

func TestRunProtoIdempotent(t *testing.T) {
	root := setupTestRoot(t)
	runner := &fakeRunner{t: t, root: root}

	_, err := RunProto(context.Background(), ProtoOptions{
		RootDir:          root,
		Proto:            "demo/v1/demo.proto",
		ServiceTargetDir: "internal/service",
		Runner:           runner,
	})
	if err != nil {
		t.Fatalf("first RunProto() error = %v", err)
	}

	// Second run should fail because proto already exists
	_, err = RunProto(context.Background(), ProtoOptions{
		RootDir:          root,
		Proto:            "demo/v1/demo.proto",
		ServiceTargetDir: "internal/service",
		Runner:           runner,
	})
	if err == nil {
		t.Fatalf("expected error on duplicate proto")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected 'already exists' error, got: %v", err)
	}
}

func TestRegisterServiceInServer(t *testing.T) {
	t.Run("grpc registration", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "grpc.go")
		original := `package server

import (
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

func NewGRPCServer() *grpc.Server {
	srv := grpc.NewServer()
	return srv
}
`
		if err := os.WriteFile(filePath, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := registerServiceInServer(filePath, "Order", "github.com/example/app/api/order/v1", "v1", true); err != nil {
			t.Fatal(err)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatal(err)
		}
		str := string(content)
		if !strings.Contains(str, `v1 "github.com/example/app/api/order/v1"`) {
			t.Fatalf("missing import:\n%s", str)
		}
		if !strings.Contains(str, "v1.RegisterOrderServer(srv, order)") {
			t.Fatalf("missing registration:\n%s", str)
		}
	})

	t.Run("http registration", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "http.go")
		original := `package server

import (
	"github.com/go-kratos/kratos/v2/transport/http"
)

func NewHTTPServer() *http.Server {
	srv := http.NewServer()
	return srv
}
`
		if err := os.WriteFile(filePath, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := registerServiceInServer(filePath, "Order", "github.com/example/app/api/order/v1", "v1", false); err != nil {
			t.Fatal(err)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatal(err)
		}
		str := string(content)
		if !strings.Contains(str, "v1.RegisterOrderHTTPServer(srv, order)") {
			t.Fatalf("missing HTTP registration:\n%s", str)
		}
	})

	t.Run("idempotent", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "grpc.go")
		original := `package server

import (
	v1 "github.com/example/app/api/order/v1"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

func NewGRPCServer() *grpc.Server {
	srv := grpc.NewServer()
	v1.RegisterOrderServer(srv, order)
	return srv
}
`
		if err := os.WriteFile(filePath, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := registerServiceInServer(filePath, "Order", "github.com/example/app/api/order/v1", "v1", true); err != nil {
			t.Fatal(err)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatal(err)
		}
		// Count occurrences of RegisterOrderServer - should be exactly 1
		count := strings.Count(string(content), "RegisterOrderServer")
		if count != 1 {
			t.Fatalf("expected 1 RegisterOrderServer, got %d:\n%s", count, string(content))
		}
	})
}

func TestInjectConfigEntry(t *testing.T) {
	t.Run("adds new entry", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.yaml")
		original := `data:
  remote_grpc:
    greeter:
      enabled: false
`
		if err := os.WriteFile(configPath, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := injectConfigEntry(configPath, "order/v1/order.proto"); err != nil {
			t.Fatal(err)
		}

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		str := string(content)
		if !strings.Contains(str, "order:") {
			t.Fatalf("missing order entry:\n%s", str)
		}
		if !strings.Contains(str, `target: "order:9000"`) {
			t.Fatalf("missing target:\n%s", str)
		}
		if !strings.Contains(str, "circuitbreaker_enabled: false") {
			t.Fatalf("missing circuitbreaker:\n%s", str)
		}
	})

	t.Run("idempotent", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.yaml")
		original := `data:
  remote_grpc:
    order:
      enabled: false
`
		if err := os.WriteFile(configPath, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := injectConfigEntry(configPath, "order/v1/order.proto"); err != nil {
			t.Fatal(err)
		}

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		count := strings.Count(string(content), "order:")
		if count != 1 {
			t.Fatalf("expected 1 'order:', got %d:\n%s", count, string(content))
		}
	})

	t.Run("no remote_grpc section", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.yaml")
		original := `server:
  http:
    enabled: true
`
		if err := os.WriteFile(configPath, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := injectConfigEntry(configPath, "order/v1/order.proto"); err != nil {
			t.Fatal(err)
		}

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		str := string(content)
		if !strings.Contains(str, "remote_grpc:") {
			t.Fatalf("missing remote_grpc section:\n%s", str)
		}
		if !strings.Contains(str, "order:") {
			t.Fatalf("missing order entry:\n%s", str)
		}
	})
}

func TestRewriteProtoAsCRUD(t *testing.T) {
	dir := t.TempDir()
	protoPath := filepath.Join(dir, "orders.proto")
	if err := os.WriteFile(protoPath, []byte("placeholder"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := rewriteProtoAsCRUD(protoPath, "orders/v1/orders.proto"); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(protoPath)
	if err != nil {
		t.Fatal(err)
	}
	str := string(content)
	for _, want := range []string{
		"package orders.v1;",
		"service Orders {",
		"rpc CreateOrder(CreateOrderRequest) returns (Order);",
		"rpc GetOrder(GetOrderRequest) returns (Order);",
		"rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse);",
		"rpc UpdateOrder(UpdateOrderRequest) returns (Order);",
		"rpc DeleteOrder(DeleteOrderRequest) returns (google.protobuf.Empty);",
		"message Order {",
		"string id = 1;",
		"google.protobuf.Timestamp created_at = 20;",
		"message ListOrdersResponse {",
		"repeated Order items = 1;",
	} {
		if !strings.Contains(str, want) {
			t.Fatalf("CRUD proto missing %q:\n%s", want, str)
		}
	}
}

func TestSingularize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"orders", "order"},
		{"users", "user"},
		{"buses", "bus"},
		{"categories", "category"},
		{"demos", "demo"},
		{"greeter", "greeter"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := singularize(tt.input)
			if got != tt.want {
				t.Fatalf("singularize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLowerCamel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"order", "order"},
		{"greeter", "greeter"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := lowerCamel(tt.input)
			if got != tt.want {
				t.Fatalf("lowerCamel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

type fakeRunner struct {
	t        *testing.T
	root     string
	commands []string
}

func (r *fakeRunner) Run(_ context.Context, dir, name string, args ...string) error {
	relDir, err := filepath.Rel(r.root, dir)
	if err != nil {
		r.t.Fatal(err)
	}
	relDir = filepath.ToSlash(relDir)
	r.commands = append(r.commands, relDir+"|"+name+" "+strings.Join(args, " "))

	if name == "kratos" && strings.Join(args, " ") == "proto add demo/v1/demo.proto" {
		protoPath := filepath.Join(dir, "demo", "v1", "demo.proto")
		if err := os.MkdirAll(filepath.Dir(protoPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(protoPath, []byte(`syntax = "proto3";

package demo.v1;

option go_package = "github.com/example/app/demo/v1;v1";
`), 0o644)
	}
	if name == "kratos" && strings.Join(args, " ") == "proto server api/demo/v1/demo.proto --target-dir=internal/service" {
		servicePath := filepath.Join(r.root, "internal", "service", "demo.go")
		return os.WriteFile(servicePath, []byte("package service\n"), 0o644)
	}
	// Handle orders proto
	if name == "kratos" && strings.Join(args, " ") == "proto add orders/v1/orders.proto" {
		protoPath := filepath.Join(dir, "orders", "v1", "orders.proto")
		if err := os.MkdirAll(filepath.Dir(protoPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(protoPath, []byte(`syntax = "proto3";

package orders.v1;

option go_package = "github.com/example/app/orders/v1;v1";
`), 0o644)
	}
	if name == "kratos" && strings.Join(args, " ") == "proto server api/orders/v1/orders.proto --target-dir=internal/service" {
		servicePath := filepath.Join(r.root, "internal", "service", "orders.go")
		return os.WriteFile(servicePath, []byte("package service\n"), 0o644)
	}
	return nil
}
