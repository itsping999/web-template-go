package dbx

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewMySQLDB,
	NewPostgresDB,
	NewMSSQLDB,
	NewClickHouseDB,
	NewSQLiteDB,
	NewGaussDB,
	NewMongoClient,
	NewRedisClient,
)
