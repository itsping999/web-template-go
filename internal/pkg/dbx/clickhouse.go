package dbx

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	gormClickhouse "gorm.io/driver/clickhouse"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type ClickHouseConfig struct {
	Enabled                  bool
	Source                   string
	MaxOpenConns             int32
	MaxIdleConns             int32
	ConnMaxLifetime          time.Duration
	ConnMaxIdleTime          time.Duration
	PingTimeout              time.Duration
	PrepareStmt              bool
	SkipDefaultTransaction   bool
	DisableNestedTransaction bool
	TablePrefix              string
	SingularTable            bool
}

type ClickHouseDB struct{ Conn *gorm.DB }

func NewClickHouseDB(cfg ClickHouseConfig, logger *logrus.Entry) (ClickHouseDB, func(), error) {
	if !cfg.Enabled {
		return ClickHouseDB{}, nil, errors.New("clickhouse is not enabled")
	}
	source := strings.TrimSpace(cfg.Source)
	if source == "" {
		return ClickHouseDB{}, nil, errors.New("clickhouse.source is empty")
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

	gdb, err := gorm.Open(gormClickhouse.Open(source), gormCfg)
	if err != nil {
		return ClickHouseDB{}, nil, err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return ClickHouseDB{}, nil, err
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
		return ClickHouseDB{}, nil, err
	}

	if logger == nil {
		logger = logrus.NewEntry(logrus.New())
	}
	helper := logger.WithField("module", "dbx/clickhouse")
	helper.Info("clickhouse connected")
	cleanup := func() {
		if err := sqlDB.Close(); err != nil {
			helper.Errorf("close clickhouse error: %v", err)
		}
	}
	return ClickHouseDB{Conn: gdb}, cleanup, nil
}
