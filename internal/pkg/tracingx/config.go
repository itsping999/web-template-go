package tracingx

import "time"

type Config struct {
	Enabled     bool              `json:"enabled" yaml:"enabled"`
	Exporter    string            `json:"exporter" yaml:"exporter"`
	Endpoint    string            `json:"endpoint" yaml:"endpoint"`
	Insecure    bool              `json:"insecure" yaml:"insecure"`
	Sampler     string            `json:"sampler" yaml:"sampler"`
	Ratio       float64           `json:"ratio" yaml:"ratio"`
	Headers     map[string]string `json:"headers" yaml:"headers"`
	Timeout     time.Duration     `json:"timeout" yaml:"timeout"`
	ServiceName string            `json:"service_name" yaml:"service_name"`
}
