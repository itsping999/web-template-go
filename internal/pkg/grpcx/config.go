package grpcx

import "time"

type ServiceConfig struct {
	Enabled               bool          `json:"enabled" yaml:"enabled"`
	Target                string        `json:"target" yaml:"target"`
	Timeout               time.Duration `json:"timeout" yaml:"timeout"`
	CircuitBreakerEnabled bool          `json:"circuitbreaker_enabled" yaml:"circuitbreaker_enabled"`
}

type Config struct {
	Greeter ServiceConfig `json:"greeter" yaml:"greeter"`
}
