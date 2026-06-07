package scaffold

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const defaultServiceTargetDir = "internal/service"

var goPackageRE = regexp.MustCompile(`(?m)^option go_package = "([^"]+)";`)

type Runner interface {
	Run(ctx context.Context, dir, name string, args ...string) error
}

type ExecRunner struct {
	Stdout *os.File
	Stderr *os.File
}

func (r ExecRunner) Run(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	return cmd.Run()
}

type ProtoOptions struct {
	RootDir          string
	Proto            string
	ServiceTargetDir string
	Runner           Runner
}

type ProtoResult struct {
	ProtoPath   string
	ServicePath string
}

func RunProto(ctx context.Context, opts ProtoOptions) (*ProtoResult, error) {
	root, err := cleanRoot(opts.RootDir)
	if err != nil {
		return nil, err
	}
	protoRel, err := NormalizeProtoPath(opts.Proto)
	if err != nil {
		return nil, err
	}
	targetDir, err := cleanRelativeDir(opts.ServiceTargetDir, defaultServiceTargetDir)
	if err != nil {
		return nil, err
	}
	runner := opts.Runner
	if runner == nil {
		runner = ExecRunner{Stdout: os.Stdout, Stderr: os.Stderr}
	}

	apiDir := filepath.Join(root, "api")
	if err := ensureDir(apiDir); err != nil {
		return nil, err
	}
	if err := ensureDir(filepath.Join(root, targetDir)); err != nil {
		return nil, err
	}

	protoPath := filepath.Join(apiDir, filepath.FromSlash(protoRel))
	if _, err := os.Stat(protoPath); err == nil {
		return nil, fmt.Errorf("%s already exists", filepath.Join("api", protoRel))
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if err := runner.Run(ctx, apiDir, "kratos", "proto", "add", protoRel); err != nil {
		return nil, fmt.Errorf("kratos proto add: %w", err)
	}
	modulePath, err := modulePath(root)
	if err != nil {
		return nil, err
	}
	goPackage, err := GoPackageForProto(modulePath, protoRel)
	if err != nil {
		return nil, err
	}
	if err := rewriteGoPackage(protoPath, goPackage); err != nil {
		return nil, err
	}

	apiProtoPath := filepath.ToSlash(filepath.Join("api", protoRel))
	if err := runner.Run(ctx, root, "kratos", "proto", "server", apiProtoPath, "--target-dir="+targetDir); err != nil {
		return nil, fmt.Errorf("kratos proto server: %w", err)
	}
	if err := runner.Run(ctx, root, "make", "api"); err != nil {
		return nil, fmt.Errorf("make api: %w", err)
	}

	servicePath := filepath.Join(root, targetDir, serviceFilename(protoRel))
	if _, err := os.Stat(servicePath); err == nil {
		if err := runner.Run(ctx, root, "gofmt", "-w", filepath.ToSlash(filepath.Join(targetDir, filepath.Base(servicePath)))); err != nil {
			return nil, fmt.Errorf("gofmt generated service: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return &ProtoResult{
		ProtoPath:   protoPath,
		ServicePath: servicePath,
	}, nil
}

func NormalizeProtoPath(path string) (string, error) {
	v := strings.TrimSpace(path)
	if v == "" {
		return "", errors.New("proto path is required")
	}
	v = filepath.ToSlash(filepath.Clean(v))
	if filepath.IsAbs(v) {
		return "", errors.New("proto path must be relative")
	}
	v = strings.TrimPrefix(v, "./")
	v = strings.TrimPrefix(v, "api/")
	if strings.HasPrefix(v, "../") || v == ".." || strings.Contains(v, "/../") {
		return "", errors.New("proto path must stay under api")
	}
	if filepath.Base(v) == "." || filepath.Ext(v) != ".proto" {
		return "", errors.New("proto path must end with .proto")
	}
	if len(strings.Split(v, "/")) < 3 {
		return "", errors.New("proto path should include package and version, for example demo/v1/demo.proto")
	}
	return v, nil
}

func GoPackageForProto(modulePath, protoRel string) (string, error) {
	modulePath = strings.TrimSpace(modulePath)
	if modulePath == "" {
		return "", errors.New("module path is empty")
	}
	dir := filepath.ToSlash(filepath.Dir(protoRel))
	if dir == "." || dir == "" {
		return "", errors.New("proto path has no directory")
	}
	pkgName := filepath.Base(dir)
	return fmt.Sprintf("%s/api/%s;%s", modulePath, dir, pkgName), nil
}

func ReplaceGoPackage(content []byte, goPackage string) ([]byte, error) {
	if !goPackageRE.Match(content) {
		return nil, errors.New("proto file has no go_package option")
	}
	return goPackageRE.ReplaceAll(content, []byte(`option go_package = "`+goPackage+`";`)), nil
}

func cleanRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	return filepath.Abs(root)
}

func cleanRelativeDir(input, fallback string) (string, error) {
	v := strings.TrimSpace(input)
	if v == "" {
		v = fallback
	}
	v = filepath.ToSlash(filepath.Clean(v))
	if filepath.IsAbs(v) || strings.HasPrefix(v, "../") || v == ".." || strings.Contains(v, "/../") {
		return "", errors.New("target directory must be relative")
	}
	return v, nil
}

func ensureDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func modulePath(root string) (string, error) {
	file, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("go.mod has no module line")
}

func rewriteGoPackage(path, goPackage string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	next, err := ReplaceGoPackage(content, goPackage)
	if err != nil {
		return err
	}
	if bytes.Equal(content, next) {
		return nil
	}
	return os.WriteFile(path, next, 0o644)
}

func serviceFilename(protoRel string) string {
	base := strings.TrimSuffix(filepath.Base(protoRel), filepath.Ext(protoRel))
	return base + ".go"
}
