package metricsx

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var (
	initOnce sync.Once
	initErr  error

	exporter *prometheus.Exporter
	provider *sdkmetric.MeterProvider
)

func Init(logger log.Logger) (func(), error) {
	initOnce.Do(func() {
		helper := log.NewHelper(log.With(logger, "module", "metrics"))

		var err error
		exporter, err = prometheus.New()
		if err != nil {
			initErr = err
			return
		}
		provider = sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
		otel.SetMeterProvider(provider)
		helper.Info("prometheus metrics enabled")
	})
	if initErr != nil {
		return nil, initErr
	}
	return func() {
		if provider == nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = provider.Shutdown(ctx)
	}, nil
}

func Handler() http.Handler {
	if exporter == nil {
		return nil
	}
	return promhttp.Handler()
}
