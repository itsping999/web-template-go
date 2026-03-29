package tracingx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func Init(cfg *Config, appName, appVersion string, logger log.Logger) (func(), error) {
	if cfg == nil || !cfg.Enabled {
		return func() {}, nil
	}
	if logger == nil {
		logger = log.NewStdLogger(os.Stdout)
	}

	helper := log.NewHelper(log.With(logger, "module", "tracing"))
	serviceName := strings.TrimSpace(cfg.ServiceName)
	if serviceName == "" {
		serviceName = appName
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(appVersion),
		),
	)
	if err != nil {
		return nil, err
	}

	exporter, err := newExporter(cfg)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(newSampler(cfg)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	helper.Infof("tracing enabled: exporter=%s endpoint=%s", cfg.Exporter, cfg.Endpoint)
	cleanup := func() {
		timeout := durationOrDefault(cfg.Timeout, 5*time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			helper.Errorf("shutdown tracing provider error: %v", err)
		}
	}
	return cleanup, nil
}

func durationOrDefault(v time.Duration, fallback time.Duration) time.Duration {
	if v <= 0 {
		return fallback
	}
	return v
}

func newExporter(cfg *Config) (sdktrace.SpanExporter, error) {
	exporter := strings.ToLower(strings.TrimSpace(cfg.Exporter))
	switch exporter {
	case "", "otlp_grpc", "otlp-grpc":
		return newOTLPGRPCExporter(cfg)
	case "otlp_http", "otlp-http":
		return newOTLPHTTPExporter(cfg)
	case "zipkin":
		return newZipkinExporter(cfg)
	case "stdout":
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	default:
		return nil, fmt.Errorf("unsupported tracing exporter: %s", cfg.Exporter)
	}
}

func newOTLPGRPCExporter(cfg *Config) (sdktrace.SpanExporter, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return nil, errors.New("tracing.endpoint is required for otlp_grpc exporter")
	}
	options := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}
	if cfg.Insecure {
		options = append(options, otlptracegrpc.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		options = append(options, otlptracegrpc.WithHeaders(cfg.Headers))
	}
	if cfg.Timeout > 0 {
		options = append(options, otlptracegrpc.WithTimeout(cfg.Timeout))
	}
	return otlptracegrpc.New(context.Background(), options...)
}

func newOTLPHTTPExporter(cfg *Config) (sdktrace.SpanExporter, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return nil, errors.New("tracing.endpoint is required for otlp_http exporter")
	}
	options := []otlptracehttp.Option{otlptracehttp.WithEndpoint(endpoint)}
	if cfg.Insecure {
		options = append(options, otlptracehttp.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		options = append(options, otlptracehttp.WithHeaders(cfg.Headers))
	}
	if cfg.Timeout > 0 {
		options = append(options, otlptracehttp.WithTimeout(cfg.Timeout))
	}
	return otlptracehttp.New(context.Background(), options...)
}

func newZipkinExporter(cfg *Config) (sdktrace.SpanExporter, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return nil, errors.New("tracing.endpoint is required for zipkin exporter")
	}
	return zipkin.New(endpoint)
}

func newSampler(cfg *Config) sdktrace.Sampler {
	ratio := cfg.Ratio
	if ratio <= 0 {
		ratio = 1.0
	}

	sampler := strings.ToLower(strings.TrimSpace(cfg.Sampler))
	switch sampler {
	case "always_off", "off":
		return sdktrace.NeverSample()
	case "traceidratio", "ratio":
		return sdktrace.TraceIDRatioBased(ratio)
	case "always_on", "on", "":
		return sdktrace.AlwaysSample()
	case "parentbased_always_on":
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample())
	case "parentbased_traceidratio", "parent_ratio":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	default:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	}
}
