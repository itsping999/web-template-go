package dbx

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type PostgresConfig struct {
	Enabled                  bool          `json:"enabled" yaml:"enabled"`
	Source                   string        `json:"source" yaml:"source"`
	MaxOpenConns             int32         `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns             int32         `json:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifetime          time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	ConnMaxIdleTime          time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time"`
	PingTimeout              time.Duration `json:"ping_timeout" yaml:"ping_timeout"`
	PrepareStmt              bool          `json:"prepare_stmt" yaml:"prepare_stmt"`
	SkipDefaultTransaction   bool          `json:"skip_default_transaction" yaml:"skip_default_transaction"`
	DisableNestedTransaction bool          `json:"disable_nested_transaction" yaml:"disable_nested_transaction"`
	TablePrefix              string        `json:"table_prefix" yaml:"table_prefix"`
	SingularTable            bool          `json:"singular_table" yaml:"singular_table"`
}

func NewPostgresDB(cfg PostgresConfig, logger log.Logger) (*gorm.DB, func(), error) {
	if logger == nil {
		logger = log.NewStdLogger(os.Stdout)
	}
	helper := log.NewHelper(log.With(logger, "module", "dbx/postgres"))
	if !cfg.Enabled {
		helper.Info("postgres disabled, skip initialization")
		return nil, func() {}, nil
	}
	source := strings.TrimSpace(cfg.Source)
	if source == "" {
		return nil, nil, errors.New("postgres.source is empty")
	}

	gormCfg := &gorm.Config{
		PrepareStmt:              cfg.PrepareStmt,
		SkipDefaultTransaction:   cfg.SkipDefaultTransaction,
		DisableNestedTransaction: cfg.DisableNestedTransaction,
	}
	if cfg.TablePrefix != "" || cfg.SingularTable {
		gormCfg.NamingStrategy = schema.NamingStrategy{
			TablePrefix:   cfg.TablePrefix,
			SingularTable: cfg.SingularTable,
		}
	}

	gdb, err := gorm.Open(gormPostgres.Open(source), gormCfg)
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, nil, err
	}

	maxOpenConns := 20
	if cfg.MaxOpenConns > 0 {
		maxOpenConns = int(cfg.MaxOpenConns)
	}
	maxIdleConns := 10
	if cfg.MaxIdleConns > 0 {
		maxIdleConns = int(cfg.MaxIdleConns)
	}
	connMaxLifetime := 30 * time.Minute
	if cfg.ConnMaxLifetime > 0 {
		connMaxLifetime = cfg.ConnMaxLifetime
	}
	connMaxIdleTime := 10 * time.Minute
	if cfg.ConnMaxIdleTime > 0 {
		connMaxIdleTime = cfg.ConnMaxIdleTime
	}
	pingTimeout := 3 * time.Second
	if cfg.PingTimeout > 0 {
		pingTimeout = cfg.PingTimeout
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	pingCtx, pingCancel := context.WithTimeout(context.Background(), pingTimeout)
	defer pingCancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, nil, err
	}

	helper.Info("postgres connected")
	cleanup := func() {
		if err := sqlDB.Close(); err != nil {
			helper.Errorf("close postgres error: %v", err)
		}
	}
	return gdb, cleanup, nil
}
