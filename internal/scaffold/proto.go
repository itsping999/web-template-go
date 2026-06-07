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

const (
	defaultServiceTargetDir = "internal/service"
	defaultBizTargetDir     = "internal/biz"
	defaultDataTargetDir    = "internal/data"
)

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
	BizTargetDir     string
	DataTargetDir    string
	Runner           Runner
}

type ProtoResult struct {
	ProtoPath   string
	ServicePath string
	BizPath     string
	DataPath    string
	NextSteps   []string
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

	bizDir, err := cleanRelativeDir(opts.BizTargetDir, defaultBizTargetDir)
	if err != nil {
		return nil, err
	}
	dataDir, err := cleanRelativeDir(opts.DataTargetDir, defaultDataTargetDir)
	if err != nil {
		return nil, err
	}

	bizPath, err := writeBizStub(root, protoRel, bizDir, modulePath)
	if err != nil {
		return nil, fmt.Errorf("generate biz stub: %w", err)
	}

	dataPath, err := writeDataStub(root, protoRel, dataDir, modulePath)
	if err != nil {
		return nil, fmt.Errorf("generate data stub: %w", err)
	}

	if err := updateProviderSets(root, protoRel, targetDir, bizDir, dataDir); err != nil {
		return nil, fmt.Errorf("update provider sets: %w", err)
	}

	return &ProtoResult{
		ProtoPath:   protoPath,
		ServicePath: servicePath,
		BizPath:     bizPath,
		DataPath:    dataPath,
		NextSteps:   protoNextSteps(protoRel, targetDir, bizDir, dataDir),
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
	if strings.TrimSuffix(filepath.Base(v), filepath.Ext(v)) == "" {
		return "", errors.New("proto path must include a proto file name")
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

// updateProviderSet adds a constructor name to a wire provider set file.
// It finds the wire.NewSet( call and appends the constructor before the closing paren.
// Handles both single-line (wire.NewSet(A, B)) and multi-line formats.
func updateProviderSet(path, constructorName string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	str := string(content)
	if strings.Contains(str, constructorName) {
		return nil // already present
	}
	idx := strings.LastIndex(str, ")")
	if idx < 0 {
		return fmt.Errorf("cannot find closing paren in %s", path)
	}
	prefix := str[:idx]
	if strings.HasSuffix(strings.TrimRight(prefix, " \t"), "\n") {
		// Multi-line: insert with tab indentation before the closing paren
		insert := fmt.Sprintf("\t%s,\n", constructorName)
		next := prefix + insert + str[idx:]
		return os.WriteFile(path, []byte(next), 0o644)
	}
	// Single-line: insert before closing paren with comma separator
	insert := fmt.Sprintf(", %s", constructorName)
	next := prefix + insert + str[idx:]
	return os.WriteFile(path, []byte(next), 0o644)
}

// updateProviderSets adds New<Service>Service, New<Singular>Usecase, and
// New<Singular>Repo to their respective provider set files.
func updateProviderSets(root, protoRel, serviceDir, bizDir, dataDir string) error {
	baseName := strings.TrimSuffix(filepath.Base(protoRel), filepath.Ext(protoRel))
	singular := singularize(baseName)

	serviceConstructor := "New" + upperCamel(baseName) + "Service"
	bizConstructor := "New" + upperCamel(singular) + "Usecase"
	dataConstructor := "New" + upperCamel(singular) + "Repo"

	if err := updateProviderSet(filepath.Join(root, serviceDir, "service.go"), serviceConstructor); err != nil {
		return fmt.Errorf("update service provider set: %w", err)
	}
	if err := updateProviderSet(filepath.Join(root, bizDir, "biz.go"), bizConstructor); err != nil {
		return fmt.Errorf("update biz provider set: %w", err)
	}
	if err := updateProviderSet(filepath.Join(root, dataDir, "data.go"), dataConstructor); err != nil {
		return fmt.Errorf("update data provider set: %w", err)
	}
	return nil
}

func protoNextSteps(protoRel, serviceTargetDir, bizDir, dataDir string) []string {
	baseName := strings.TrimSuffix(filepath.Base(protoRel), filepath.Ext(protoRel))
	serviceFile := filepath.ToSlash(filepath.Join(serviceTargetDir, serviceFilename(protoRel)))
	serviceName := upperCamel(baseName)
	singular := singularize(baseName)
	bizFile := filepath.ToSlash(filepath.Join(bizDir, baseName+".go"))
	dataFile := filepath.ToSlash(filepath.Join(dataDir, baseName+".go"))
	return []string{
		fmt.Sprintf("Implement %sRepo interface methods in %s.", upperCamel(singular), dataFile),
		fmt.Sprintf("Implement %sUsecase methods in %s.", upperCamel(singular), bizFile),
		fmt.Sprintf("Implement service methods in %s.", serviceFile),
		fmt.Sprintf("Register the generated %s service in internal/server/grpc.go and internal/server/http.go.", serviceName),
		"Run go generate ./... && go mod tidy, then make verify.",
	}
}

func writeBizStub(root, protoRel, bizDir, modulePath string) (string, error) {
	baseName := strings.TrimSuffix(filepath.Base(protoRel), filepath.Ext(protoRel))
	singular := singularize(baseName)
	interfaceName := upperCamel(singular) + "Repo"
	usecaseName := upperCamel(singular) + "Usecase"
	constructorName := "New" + usecaseName

	tpl := `package biz

import (
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	Err` + upperCamel(singular) + `NotFound = errors.NotFound("` + strings.ToUpper(baseName) + `_NOT_FOUND", "` + singular + ` not found")
)

// ` + upperCamel(singular) + ` is the domain model for ` + singular + `.
type ` + upperCamel(singular) + ` struct {
	// TODO: add domain fields, for example: ID, Name, CreatedAt
}

// ` + interfaceName + ` defines the outbound port for ` + singular + ` persistence.
type ` + interfaceName + ` interface {
	// TODO: define domain methods, for example:
	// Save(context.Context, *` + upperCamel(singular) + `) (*` + upperCamel(singular) + `, error)
}

// ` + usecaseName + ` implements ` + singular + ` business rules.
type ` + usecaseName + ` struct {
	repo ` + interfaceName + `
	log  *log.Helper
}

// ` + constructorName + ` creates a new ` + usecaseName + `.
func ` + constructorName + `(repo ` + interfaceName + `, logger log.Logger) *` + usecaseName + ` {
	return &` + usecaseName + `{
		repo: repo,
		log:  log.NewHelper(log.With(logger, "module", "biz/` + baseName + `")),
	}
}
`

	if err := ensureOrCreateDir(filepath.Join(root, bizDir)); err != nil {
		return "", err
	}
	path := filepath.Join(root, bizDir, baseName+".go")
	if err := os.WriteFile(path, []byte(tpl), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func writeDataStub(root, protoRel, dataDir, modulePath string) (string, error) {
	baseName := strings.TrimSuffix(filepath.Base(protoRel), filepath.Ext(protoRel))
	singular := singularize(baseName)
	repoName := lowerCamel(singular) + "Repo"
	interfaceName := upperCamel(singular) + "Repo"
	constructorName := "New" + upperCamel(singular) + "Repo"

	tpl := `package data

import (
	"github.com/go-kratos/kratos/v2/log"
	"` + modulePath + `/internal/biz` + `"
)

type ` + repoName + ` struct {
	log *log.Helper
}

// ` + constructorName + ` implements biz.` + interfaceName + `.
func ` + constructorName + `(logger log.Logger) biz.` + interfaceName + ` {
	return &` + repoName + `{
		log: log.NewHelper(log.With(logger, "module", "data/` + baseName + `")),
	}
}

// TODO: implement biz.` + interfaceName + ` methods.
`

	if err := ensureOrCreateDir(filepath.Join(root, dataDir)); err != nil {
		return "", err
	}
	path := filepath.Join(root, dataDir, baseName+".go")
	if err := os.WriteFile(path, []byte(tpl), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// ensureOrCreateDir ensures a directory exists, creating it if necessary.
// Unlike ensureDir, this does not fail when the directory does not yet exist.
func ensureOrCreateDir(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", path)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.MkdirAll(path, 0o755)
}

// singularize returns a naive singular form of a plural English word.
// This is intentionally simple; override for irregular nouns.
func singularize(word string) string {
	if strings.HasSuffix(word, "ies") && len(word) > 3 {
		return word[:len(word)-3] + "y"
	}
	if strings.HasSuffix(word, "ses") || strings.HasSuffix(word, "xes") || strings.HasSuffix(word, "zes") {
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") {
		return word[:len(word)-1]
	}
	return word
}

func lowerCamel(input string) string {
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	})
	var out strings.Builder
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i == 0 {
			out.WriteString(strings.ToLower(part))
		} else {
			out.WriteString(strings.ToUpper(part[:1]))
			if len(part) > 1 {
				out.WriteString(part[1:])
			}
		}
	}
	return out.String()
}

func upperCamel(input string) string {
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	})
	var out strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		out.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			out.WriteString(part[1:])
		}
	}
	return out.String()
}
