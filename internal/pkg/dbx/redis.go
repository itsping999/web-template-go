package dbx

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Enabled      bool          `json:"enabled" yaml:"enabled"`
	Network      string        `json:"network" yaml:"network"`
	Addr         string        `json:"addr" yaml:"addr"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`
	Username     string        `json:"username" yaml:"username"`
	Password     string        `json:"password" yaml:"password"`
	DB           int32         `json:"db" yaml:"db"`
	PoolSize     int32         `json:"pool_size" yaml:"pool_size"`
	DialTimeout  time.Duration `json:"dial_timeout" yaml:"dial_timeout"`
	MinIdleConns int32         `json:"min_idle_conns" yaml:"min_idle_conns"`
	MaxRetries   int32         `json:"max_retries" yaml:"max_retries"`
	PoolTimeout  time.Duration `json:"pool_timeout" yaml:"pool_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout" yaml:"idle_timeout"`
	MaxConnAge   time.Duration `json:"max_conn_age" yaml:"max_conn_age"`
}

func NewRedisClient(cfg RedisConfig, logger log.Logger) (*redis.Client, func(), error) {
	if logger == nil {
		logger = log.NewStdLogger(os.Stdout)
	}
	helper := log.NewHelper(log.With(logger, "module", "dbx/redis"))
	if !cfg.Enabled {
		helper.Info("redis disabled, skip initialization")
		return nil, func() {}, nil
	}
	if strings.TrimSpace(cfg.Addr) == "" {
		return nil, nil, errors.New("redis.addr is empty")
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

	helper.Infof("redis connected: addr=%s", cfg.Addr)
	cleanup := func() {
		if err := client.Close(); err != nil {
			helper.Errorf("close redis error: %v", err)
		}
	}
	return client, cleanup, nil
}
