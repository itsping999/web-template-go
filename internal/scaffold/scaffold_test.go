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

func TestRunProtoScaffold(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/example/app\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal", "service"), 0o755); err != nil {
		t.Fatal(err)
	}

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

	content, err := os.ReadFile(result.ProtoPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `option go_package = "github.com/example/app/api/demo/v1;v1";`) {
		t.Fatalf("go_package not fixed:\n%s", string(content))
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
	return nil
}
