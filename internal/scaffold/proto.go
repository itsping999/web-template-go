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
	CRUD             bool
	SkipAutoRegister bool
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

	if opts.CRUD {
		if err := rewriteProtoAsCRUD(protoPath, protoRel); err != nil {
			return nil, fmt.Errorf("rewrite proto as CRUD: %w", err)
		}
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

	bizPath, err := writeBizStub(root, protoRel, bizDir, modulePath, opts.CRUD)
	if err != nil {
		return nil, fmt.Errorf("generate biz stub: %w", err)
	}

	dataPath, err := writeDataStub(root, protoRel, dataDir, modulePath, opts.CRUD)
	if err != nil {
		return nil, fmt.Errorf("generate data stub: %w", err)
	}

	if err := updateProviderSets(root, protoRel, targetDir, bizDir, dataDir); err != nil {
		return nil, fmt.Errorf("update provider sets: %w", err)
	}

	if !opts.SkipAutoRegister {
		baseName := strings.TrimSuffix(filepath.Base(protoRel), filepath.Ext(protoRel))
		svcName := upperCamel(baseName)
		protoPkg := protoPackage(protoRel)
		importPath := fmt.Sprintf("%s/api/%s", modulePath, filepath.ToSlash(filepath.Dir(protoRel)))

		if err := registerServiceInServer(filepath.Join(root, "internal", "server", "grpc.go"), svcName, importPath, protoPkg, true); err != nil {
			return nil, fmt.Errorf("register gRPC service: %w", err)
		}
		if err := registerServiceInServer(filepath.Join(root, "internal", "server", "http.go"), svcName, importPath, protoPkg, false); err != nil {
			return nil, fmt.Errorf("register HTTP service: %w", err)
		}
	}

	if err := injectConfigEntry(filepath.Join(root, "configs", "config.yaml"), protoRel); err != nil {
		return nil, fmt.Errorf("inject config entry: %w", err)
	}

	nextSteps := protoNextSteps(protoRel, targetDir, bizDir, dataDir)
	if opts.SkipAutoRegister {
		nextSteps = append([]string{
			fmt.Sprintf("Register the generated %s service in internal/server/grpc.go and internal/server/http.go.", upperCamel(strings.TrimSuffix(filepath.Base(protoRel), filepath.Ext(protoRel)))),
		}, nextSteps...)
	}

	return &ProtoResult{
		ProtoPath:   protoPath,
		ServicePath: servicePath,
		BizPath:     bizPath,
		DataPath:    dataPath,
		NextSteps:   nextSteps,
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

// protoNextSteps returns the remaining manual steps after auto-registration.
func protoNextSteps(protoRel, serviceTargetDir, bizDir, dataDir string) []string {
	baseName := strings.TrimSuffix(filepath.Base(protoRel), filepath.Ext(protoRel))
	serviceFile := filepath.ToSlash(filepath.Join(serviceTargetDir, serviceFilename(protoRel)))
	singular := singularize(baseName)
	bizFile := filepath.ToSlash(filepath.Join(bizDir, baseName+".go"))
	dataFile := filepath.ToSlash(filepath.Join(dataDir, baseName+".go"))
	return []string{
		fmt.Sprintf("Implement %sRepo interface methods in %s.", upperCamel(singular), dataFile),
		fmt.Sprintf("Implement %sUsecase methods in %s.", upperCamel(singular), bizFile),
		fmt.Sprintf("Implement service methods in %s.", serviceFile),
		"Run make generate, then make verify.",
	}
}

// protoPackage returns the protobuf package name from a proto relative path.
// For example "order/v1/order.proto" returns "order.v1".
func protoPackage(protoRel string) string {
	dir := filepath.ToSlash(filepath.Dir(protoRel))
	parts := strings.Split(dir, "/")
	return strings.Join(parts, ".")
}

// rewriteProtoAsCRUD replaces the default Kratos-generated proto content with
// a full CRUD service definition including Create, Get, List, Update, Delete RPCs.
func rewriteProtoAsCRUD(protoPath, protoRel string) error {
	baseName := strings.TrimSuffix(filepath.Base(protoRel), filepath.Ext(protoRel))
	pkg := protoPackage(protoRel)
	svcName := upperCamel(baseName)
	singular := upperCamel(singularize(baseName))
	plural := svcName

	lines := []string{
		`syntax = "proto3";`,
		``,
		`package ` + pkg + `;`,
		``,
		`option go_package = "placeholder";`,
		``,
		`import "google/protobuf/timestamp.proto";`,
		`import "google/protobuf/empty.proto";`,
		`import "validate/validate.proto";`,
		``,
		`service ` + svcName + ` {`,
		`  rpc Create` + singular + `(Create` + singular + `Request) returns (` + singular + `);`,
		`  rpc Get` + singular + `(Get` + singular + `Request) returns (` + singular + `);`,
		`  rpc List` + plural + `(List` + plural + `Request) returns (List` + plural + `Response);`,
		`  rpc Update` + singular + `(Update` + singular + `Request) returns (` + singular + `);`,
		`  rpc Delete` + singular + `(Delete` + singular + `Request) returns (google.protobuf.Empty);`,
		`}`,
		``,
		`message ` + singular + ` {`,
		`  string id = 1;`,
		`  // TODO: add domain fields`,
		`  google.protobuf.Timestamp created_at = 20;`,
		`  google.protobuf.Timestamp updated_at = 21;`,
		`}`,
		``,
		`message Create` + singular + `Request {`,
		`  // TODO: add creation fields`,
		`}`,
		``,
		`message Get` + singular + `Request {`,
		`  string id = 1 [(validate.rules).string.min_len = 1];`,
		`}`,
		``,
		`message List` + plural + `Request {`,
		`  int32 page = 1;`,
		`  int32 page_size = 2;`,
		`}`,
		``,
		`message List` + plural + `Response {`,
		`  repeated ` + singular + ` items = 1;`,
		`  int32 total = 2;`,
		`}`,
		``,
		`message Update` + singular + `Request {`,
		`  string id = 1 [(validate.rules).string.min_len = 1];`,
		`  // TODO: add updatable fields`,
		`}`,
		``,
		`message Delete` + singular + `Request {`,
		`  string id = 1 [(validate.rules).string.min_len = 1];`,
		`}`,
		``,
	}

	content := strings.Join(lines, "\n")
	return os.WriteFile(protoPath, []byte(content), 0o644)
}

// registerServiceInServer adds the service registration call and import to a
// server file (grpc.go or http.go). For gRPC it calls Register<Svc>Server,
// for HTTP it calls Register<Svc>HTTPServer.
func registerServiceInServer(filePath, svcName, importPath, protoPkg string, isGRPC bool) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	str := string(content)

	var registerCall string
	if isGRPC {
		registerCall = fmt.Sprintf("%s.Register%sServer(srv, %s)", protoPkg, svcName, lowerCamel(singularize(svcName)))
	} else {
		registerCall = fmt.Sprintf("%s.Register%sHTTPServer(srv, %s)", protoPkg, svcName, lowerCamel(singularize(svcName)))
	}

	// Already registered
	if strings.Contains(str, registerCall) {
		return nil
	}

	// Add import
	alias := protoPkg
	str = addImport(str, alias, importPath)

	// Add registration call after srv := ...NewServer(...)
	idx := strings.Index(str, "srv := ")
	if idx < 0 {
		// Try alternate pattern
		idx = strings.Index(str, "srv:=")
	}
	if idx < 0 {
		return fmt.Errorf("cannot find 'srv :=' in %s", filePath)
	}
	// Find end of line
	eol := strings.Index(str[idx:], "\n")
	if eol < 0 {
		eol = len(str) - idx
	}
	insertPos := idx + eol
	str = str[:insertPos] + "\n\t" + registerCall + str[insertPos:]

	return os.WriteFile(filePath, []byte(str), 0o644)
}

// addImport adds an import line to a Go file if not already present.
func addImport(content, alias, importPath string) string {
	if strings.Contains(content, `"`+importPath+`"`) {
		return content
	}

	// Find the import block
	lines := strings.Split(content, "\n")
	inImport := false
	parenDepth := 0
	insertIdx := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import") {
			if strings.Contains(trimmed, "(") {
				inImport = true
				parenDepth++
				continue
			}
			// single import line
			continue
		}
		if inImport {
			if strings.Contains(trimmed, "(") {
				parenDepth++
			}
			if strings.Contains(trimmed, ")") {
				parenDepth--
				if parenDepth == 0 {
					insertIdx = i
					break
				}
			}
		}
	}

	if insertIdx < 0 {
		return content
	}

	importLine := fmt.Sprintf("\t%s \"%s\"", alias, importPath)
	lines = append(lines[:insertIdx], append([]string{importLine}, lines[insertIdx:]...)...)
	return strings.Join(lines, "\n")
}

// injectConfigEntry adds a remote_grpc config entry for the new module
// in configs/config.yaml if not already present.
func injectConfigEntry(configPath, protoRel string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	str := string(content)

	baseName := strings.TrimSuffix(filepath.Base(protoRel), filepath.Ext(protoRel))
	entry := fmt.Sprintf("    %s:", baseName)

	if strings.Contains(str, entry) {
		return nil
	}

	block := fmt.Sprintf(`%s:
      enabled: false
      target: "%s:9000"
      timeout: 500ms
      circuitbreaker_enabled: false
`, entry, baseName)

	// Insert after "remote_grpc:" line
	idx := strings.Index(str, "remote_grpc:")
	if idx < 0 {
		// Append at end
		str = strings.TrimRight(str, "\n") + "\n\nremote_grpc:\n" + block
		return os.WriteFile(configPath, []byte(str), 0o644)
	}

	eol := strings.Index(str[idx:], "\n")
	if eol < 0 {
		str += "\n" + block
	} else {
		insertPos := idx + eol + 1
		str = str[:insertPos] + block + str[insertPos:]
	}

	return os.WriteFile(configPath, []byte(str), 0o644)
}

// writeBizStub generates a biz-layer stub file.
func writeBizStub(root, protoRel, bizDir, modulePath string, crud bool) (string, error) {
	baseName := strings.TrimSuffix(filepath.Base(protoRel), filepath.Ext(protoRel))
	singular := singularize(baseName)
	interfaceName := upperCamel(singular) + "Repo"
	usecaseName := upperCamel(singular) + "Usecase"
	constructorName := "New" + usecaseName
	modelName := upperCamel(singular)

	repoMethods := "\t// TODO: define domain methods, for example:\n\t// Save(context.Context, *" + modelName + ") (*" + modelName + ", error)"
	ucMethods := "\t// TODO: implement business methods"

	if crud {
		repoMethods = fmt.Sprintf(`	Create(context.Context, *%s) (*%s, error)
	Get(context.Context, string) (*%s, error)
	List(context.Context, int32, int32) ([]*%s, int32, error)
	Update(context.Context, *%s) (*%s, error)
	Delete(context.Context, string) error`, modelName, modelName, modelName, modelName, modelName, modelName)
		ucMethods = fmt.Sprintf(`// Create%s creates a new %s.
func (uc *%s) Create%s(ctx context.Context, %s *%s) (*%s, error) {
	// TODO: add validation and business rules
	return uc.repo.Create(ctx, %s)
}

// Get%s retrieves a %s by ID.
func (uc *%s) Get%s(ctx context.Context, id string) (*%s, error) {
	return uc.repo.Get(ctx, id)
}

// List%s returns a paginated list of %ss.
func (uc *%s) List%s(ctx context.Context, page, pageSize int32) ([]*%s, int32, error) {
	return uc.repo.List(ctx, page, pageSize)
}

// Update%s updates an existing %s.
func (uc *%s) Update%s(ctx context.Context, %s *%s) (*%s, error) {
	// TODO: add validation and business rules
	return uc.repo.Update(ctx, %s)
}

// Delete%s deletes a %s by ID.
func (uc *%s) Delete%s(ctx context.Context, id string) error {
	return uc.repo.Delete(ctx, id)
}`,
			modelName, singular,
			usecaseName, modelName, lowerCamel(singular), modelName, modelName, lowerCamel(singular),
			modelName, singular,
			usecaseName, modelName, modelName,
			modelName, singular,
			usecaseName, modelName, modelName,
			modelName, singular,
			usecaseName, modelName, lowerCamel(singular), modelName, modelName, lowerCamel(singular),
			modelName, singular,
			usecaseName, modelName,
		)
	}

	tpl := `package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	Err` + modelName + `NotFound = errors.NotFound("` + strings.ToUpper(baseName) + `_NOT_FOUND", "` + singular + ` not found")
)

// ` + modelName + ` is the domain model for ` + singular + `.
type ` + modelName + ` struct {
	ID string
	// TODO: add domain fields
}

// ` + interfaceName + ` defines the outbound port for ` + singular + ` persistence.
type ` + interfaceName + ` interface {
` + repoMethods + `
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

` + ucMethods + `
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

// writeDataStub generates a data-layer stub file.
func writeDataStub(root, protoRel, dataDir, modulePath string, crud bool) (string, error) {
	baseName := strings.TrimSuffix(filepath.Base(protoRel), filepath.Ext(protoRel))
	singular := singularize(baseName)
	repoName := lowerCamel(singular) + "Repo"
	interfaceName := upperCamel(singular) + "Repo"
	constructorName := "New" + upperCamel(singular) + "Repo"
	modelName := upperCamel(singular)

	methodImpls := "\t// TODO: implement biz." + interfaceName + " methods."

	if crud {
		methodImpls = fmt.Sprintf(`func (r *%s) Create(ctx context.Context, %s *biz.%s) (*biz.%s, error) {
	// TODO: implement with actual persistence (e.g. GORM, MongoDB)
	return nil, fmt.Errorf("not implemented")
}

func (r *%s) Get(ctx context.Context, id string) (*biz.%s, error) {
	// TODO: implement with actual persistence
	return nil, fmt.Errorf("not implemented")
}

func (r *%s) List(ctx context.Context, page, pageSize int32) ([]*biz.%s, int32, error) {
	// TODO: implement with actual persistence
	return nil, 0, fmt.Errorf("not implemented")
}

func (r *%s) Update(ctx context.Context, %s *biz.%s) (*biz.%s, error) {
	// TODO: implement with actual persistence
	return nil, fmt.Errorf("not implemented")
}

func (r *%s) Delete(ctx context.Context, id string) error {
	// TODO: implement with actual persistence
	return fmt.Errorf("not implemented")
}`,
			repoName, lowerCamel(singular), modelName, modelName,
			repoName, modelName,
			repoName, modelName,
			repoName, lowerCamel(singular), modelName, modelName,
			repoName,
		)
	}

	tpl := `package data

import (
	"context"

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

` + methodImpls + `
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
