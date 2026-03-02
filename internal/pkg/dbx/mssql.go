package dbx

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	gormSQLServer "gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type MSSQLConfig struct {
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

type MSSQLDB struct{ Conn *gorm.DB }

func NewMSSQLDB(cfg MSSQLConfig, logger *logrus.Entry) (MSSQLDB, func(), error) {
	if !cfg.Enabled {
		return MSSQLDB{}, nil, errors.New("mssql is not enabled")
	}
	source := strings.TrimSpace(cfg.Source)
	if source == "" {
		return MSSQLDB{}, nil, errors.New("mssql.source is empty")
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

	gdb, err := gorm.Open(gormSQLServer.Open(source), gormCfg)
	if err != nil {
		return MSSQLDB{}, nil, err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return MSSQLDB{}, nil, err
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
		return MSSQLDB{}, nil, err
	}

	if logger == nil {
		logger = logrus.NewEntry(logrus.New())
	}
	helper := logger.WithField("module", "dbx/mssql")
	helper.Info("mssql connected")
	cleanup := func() {
		if err := sqlDB.Close(); err != nil {
			helper.Errorf("close mssql error: %v", err)
		}
	}
	return MSSQLDB{Conn: gdb}, cleanup, nil
}
