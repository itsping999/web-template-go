package tests

import (
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/wyuhsin/web-template-go/internal/pkg/dbx"
)

func TestDisabledProvidersReturnNoErrorAndCleanup(t *testing.T) {
	logger := log.NewStdLogger(io.Discard)

	if _, cleanup, err := dbx.NewPostgresDB(dbx.PostgresConfig{Enabled: false}, logger); err != nil || cleanup == nil {
		t.Fatalf("postgres disabled should return nil error and non-nil cleanup")
	}
	if _, cleanup, err := dbx.NewRedisClient(dbx.RedisConfig{Enabled: false}, logger); err != nil || cleanup == nil {
		t.Fatalf("redis disabled should return nil error and non-nil cleanup")
	}
	if _, cleanup, err := dbx.NewMongoClient(dbx.MongoConfig{Enabled: false}, logger); err != nil || cleanup == nil {
		t.Fatalf("mongodb disabled should return nil error and non-nil cleanup")
	}
}

func TestNewMongoClientMissingRequiredConfig(t *testing.T) {
	_, _, err := dbx.NewMongoClient(dbx.MongoConfig{Enabled: true}, log.NewStdLogger(io.Discard))
	if err == nil {
		t.Fatalf("expected error when uri is missing")
	}
}
