package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport"
	grpcTransport "github.com/go-kratos/kratos/v2/transport/grpc"
	httpTransport "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/kratos-transport/transport/mqtt"
	"github.com/tx7do/kratos-transport/transport/rabbitmq"
	"github.com/tx7do/kratos-transport/transport/websocket"
	"github.com/wyuhsin/web-template-go/internal/conf"
	"github.com/wyuhsin/web-template-go/internal/pkg/logx"
	"github.com/wyuhsin/web-template-go/internal/pkg/metricsx"
	"github.com/wyuhsin/web-template-go/internal/pkg/tracingx"
	"gopkg.in/yaml.v3"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string
	id, _    = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "./configs", "config path, eg: -conf config.yaml")
}

func newApp(
	logger log.Logger,
	registrar registry.Registrar,
	gs *grpcTransport.Server,
	hs *httpTransport.Server,
	ws *websocket.Server,
	ms *mqtt.Server,
	rs *rabbitmq.Server,
) *kratos.App {
	servers := make([]transport.Server, 0, 5)
	if gs != nil {
		servers = append(servers, gs)
	}
	if hs != nil {
		servers = append(servers, hs)
	}
	if ws != nil {
		servers = append(servers, ws)
	}
	if ms != nil {
		servers = append(servers, ms)
	}
	if rs != nil {
		servers = append(servers, rs)
	}

	opts := []kratos.Option{
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(servers...),
	}
	if registrar != nil {
		opts = append(opts, kratos.Registrar(registrar))
	}
	return kratos.New(opts...)
}

func main() {
	flag.Parse()

	var bc conf.Bootstrap
	if err := loadBootstrap(flagconf, &bc); err != nil {
		panic(err)
	}
	applyEnvOverrides(&bc)

	baseLogger := logx.NewWithOptions(bc.Logger)
	logger := log.With(
		baseLogger,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)

	tracingCleanup, err := tracingx.Init(&bc.Tracing, Name, Version, logger)
	if err != nil {
		panic(err)
	}
	defer tracingCleanup()
	metricsCleanup, err := metricsx.Init(logger)
	if err != nil {
		panic(err)
	}
	defer metricsCleanup()

	app, cleanup, err := wireApp(&bc.Server, &bc.Data, &bc.Tracing, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}

func loadBootstrap(confPath string, bc *conf.Bootstrap) error {
	target := confPath
	stat, err := os.Stat(target)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		target = filepath.Join(target, "config.yaml")
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(content, bc)
}
