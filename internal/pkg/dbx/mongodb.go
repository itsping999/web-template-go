package dbx

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoConfig struct {
	Enabled                bool
	URI                    string
	ConnectTimeout         time.Duration
	PingTimeout            time.Duration
	MinPoolSize            int32
	MaxPoolSize            int32
	ServerSelectionTimeout time.Duration
	SocketTimeout          time.Duration
	Username               string
	Password               string
	AuthSource             string
}

func NewMongoClient(cfg MongoConfig, logger *logrus.Entry) (*mongo.Client, func(), error) {
	if !cfg.Enabled {
		return nil, func() {}, nil
	}
	if strings.TrimSpace(cfg.URI) == "" {
		return nil, nil, errors.New("mongodb.uri is empty")
	}
	if logger == nil {
		logger = logrus.NewEntry(logrus.New())
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

	helper := logger.WithField("module", "dbx/mongodb")
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
