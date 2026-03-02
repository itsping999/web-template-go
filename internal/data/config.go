package data

import "github.com/wyuhsin/web-template-go/internal/pkg/dbx"

type Config struct {
	MySQL      dbx.MySQLConfig      `json:"mysql" yaml:"mysql"`
	Postgres   dbx.PostgresConfig   `json:"postgres" yaml:"postgres"`
	MSSQL      dbx.MSSQLConfig      `json:"mssql" yaml:"mssql"`
	ClickHouse dbx.ClickHouseConfig `json:"clickhouse" yaml:"clickhouse"`
	SQLite     dbx.SQLiteConfig     `json:"sqlite" yaml:"sqlite"`
	GaussDB    dbx.GaussDBConfig    `json:"gaussdb" yaml:"gaussdb"`
	Redis      dbx.RedisConfig      `json:"redis" yaml:"redis"`
	MongoDB    dbx.MongoConfig      `json:"mongodb" yaml:"mongodb"`
}
