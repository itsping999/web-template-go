package dbx

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type RedisConfig struct {
	Enabled      bool
	Network      string
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Username     string
	Password     string
	DB           int32
	PoolSize     int32
	DialTimeout  time.Duration
	MinIdleConns int32
	MaxRetries   int32
	PoolTimeout  time.Duration
	IdleTimeout  time.Duration
	MaxConnAge   time.Duration
}

func NewRedisClient(cfg RedisConfig, logger *logrus.Entry) (*redis.Client, func(), error) {
	if !cfg.Enabled {
		return nil, nil, errors.New("redis is not enabled")
	}
	if strings.TrimSpace(cfg.Addr) == "" {
		return nil, nil, errors.New("redis.addr is empty")
	}
	if logger == nil {
		logger = logrus.NewEntry(logrus.New())
	}

	network := cfg.Network
	if network == "" {
		network = "tcp"
	}

	opts := &redis.Options{
		Network:  network,
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       int(cfg.DB),
	}
	if cfg.PoolSize > 0 {
		opts.PoolSize = int(cfg.PoolSize)
	}
	if cfg.MinIdleConns > 0 {
		opts.MinIdleConns = int(cfg.MinIdleConns)
	}
	if cfg.MaxRetries > 0 {
		opts.MaxRetries = int(cfg.MaxRetries)
	}
	if cfg.DialTimeout > 0 {
		opts.DialTimeout = cfg.DialTimeout
	}
	if cfg.PoolTimeout > 0 {
		opts.PoolTimeout = cfg.PoolTimeout
	}
	if cfg.IdleTimeout > 0 {
		opts.ConnMaxIdleTime = cfg.IdleTimeout
	}
	if cfg.MaxConnAge > 0 {
		opts.ConnMaxLifetime = cfg.MaxConnAge
	}
	if cfg.ReadTimeout > 0 {
		opts.ReadTimeout = cfg.ReadTimeout
	}
	if cfg.WriteTimeout > 0 {
		opts.WriteTimeout = cfg.WriteTimeout
	}

	client := redis.NewClient(opts)
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, nil, err
	}

	helper := logger.WithField("module", "dbx")
	helper.Infof("redis connected: addr=%s", cfg.Addr)
	cleanup := func() {
		if err := client.Close(); err != nil {
			helper.Errorf("close redis error: %v", err)
		}
	}
	return client, cleanup, nil
}
