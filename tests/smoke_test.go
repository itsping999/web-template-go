package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/wyuhsin/web-template-go/internal/pkg/dbx"
)

func TestSmokeExternalDependencies(t *testing.T) {
	if os.Getenv("RUN_SMOKE") != "1" {
		t.Skip("skip smoke test, set RUN_SMOKE=1 to enable")
	}

	logger := log.NewStdLogger(os.Stdout)

	t.Run("postgres", func(t *testing.T) {
		cfg := dbx.PostgresConfig{
			Enabled:     true,
			Source:      "postgres://postgres:postgres@codex-postgres:5432/test?sslmode=disable",
			PingTimeout: 5 * time.Second,
		}
		_, cleanup, err := dbx.NewPostgresDB(cfg, logger)
		if cleanup != nil {
			defer cleanup()
		}
		if err != nil {
			t.Fatalf("postgres smoke failed: %v", err)
		}
	})

	t.Run("redis", func(t *testing.T) {
		cfg := dbx.RedisConfig{
			Enabled:     true,
			Addr:        "codex-redis:6379",
			Network:     "tcp",
			DialTimeout: 3 * time.Second,
		}
		c, cleanup, err := dbx.NewRedisClient(cfg, logger)
		if cleanup != nil {
			defer cleanup()
		}
		if err != nil {
			t.Fatalf("redis smoke failed: %v", err)
		}
		if err := c.Set(context.Background(), "smoke:key", "ok", 5*time.Second).Err(); err != nil {
			t.Fatalf("redis set smoke failed: %v", err)
		}
	})

	t.Run("mongodb", func(t *testing.T) {
		cfg := dbx.MongoConfig{
			Enabled:        true,
			URI:            "mongodb://codex-mongo:27017",
			ConnectTimeout: 5 * time.Second,
			PingTimeout:    5 * time.Second,
		}
		_, cleanup, err := dbx.NewMongoClient(cfg, logger)
		if cleanup != nil {
			defer cleanup()
		}
		if err != nil {
			t.Fatalf("mongodb smoke failed: %v", err)
		}
	})
}
