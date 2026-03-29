package dbx

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoConfig struct {
	Enabled                bool          `json:"enabled" yaml:"enabled"`
	URI                    string        `json:"uri" yaml:"uri"`
	ConnectTimeout         time.Duration `json:"connect_timeout" yaml:"connect_timeout"`
	PingTimeout            time.Duration `json:"ping_timeout" yaml:"ping_timeout"`
	MinPoolSize            int32         `json:"min_pool_size" yaml:"min_pool_size"`
	MaxPoolSize            int32         `json:"max_pool_size" yaml:"max_pool_size"`
	ServerSelectionTimeout time.Duration `json:"server_selection_timeout" yaml:"server_selection_timeout"`
	SocketTimeout          time.Duration `json:"socket_timeout" yaml:"socket_timeout"`
	Username               string        `json:"username" yaml:"username"`
	Password               string        `json:"password" yaml:"password"`
	AuthSource             string        `json:"auth_source" yaml:"auth_source"`
}

func NewMongoClient(cfg MongoConfig, logger log.Logger) (*mongo.Client, func(), error) {
	if !cfg.Enabled {
		return nil, func() {}, nil
	}
	if strings.TrimSpace(cfg.URI) == "" {
		return nil, nil, errors.New("mongodb.uri is empty")
	}
	if logger == nil {
		logger = log.NewStdLogger(os.Stdout)
	}

	connectTimeout := 5 * time.Second
	if cfg.ConnectTimeout > 0 {
		connectTimeout = cfg.ConnectTimeout
	}
	pingTimeout := 3 * time.Second
	if cfg.PingTimeout > 0 {
		pingTimeout = cfg.PingTimeout
	}

	clientOpts := options.Client().ApplyURI(cfg.URI)
	if cfg.Username != "" || cfg.Password != "" {
		clientOpts.SetAuth(options.Credential{
			Username:   cfg.Username,
			Password:   cfg.Password,
			AuthSource: cfg.AuthSource,
		})
	}
	if cfg.MinPoolSize > 0 {
		clientOpts.SetMinPoolSize(uint64(cfg.MinPoolSize))
	}
	if cfg.MaxPoolSize > 0 {
		clientOpts.SetMaxPoolSize(uint64(cfg.MaxPoolSize))
	}
	if cfg.ServerSelectionTimeout > 0 {
		clientOpts.SetServerSelectionTimeout(cfg.ServerSelectionTimeout)
	}
	if cfg.SocketTimeout > 0 {
		clientOpts.SetSocketTimeout(cfg.SocketTimeout)
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, nil, err
	}

	pingCtx, pingCancel := context.WithTimeout(context.Background(), pingTimeout)
	defer pingCancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		return nil, nil, err
	}

	helper := log.NewHelper(log.With(logger, "module", "dbx/mongodb"))
	helper.Info("mongodb connected")
	cleanup := func() {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer disconnectCancel()
		if err := client.Disconnect(disconnectCtx); err != nil {
			helper.Errorf("close mongodb error: %v", err)
		}
	}
	return client, cleanup, nil
}
