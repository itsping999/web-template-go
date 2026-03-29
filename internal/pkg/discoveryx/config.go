package discoveryx

import "time"

type Config struct {
	Kubernetes KubernetesConfig `json:"kubernetes" yaml:"kubernetes"`
}

type KubernetesConfig struct {
	Enabled            bool          `json:"enabled" yaml:"enabled"`
	Mode               string        `json:"mode" yaml:"mode"`
	APIServer          string        `json:"api_server" yaml:"api_server"`
	Kubeconfig         string        `json:"kubeconfig" yaml:"kubeconfig"`
	Namespace          string        `json:"namespace" yaml:"namespace"`
	Token              string        `json:"token" yaml:"token"`
	Timeout            time.Duration `json:"timeout" yaml:"timeout"`
	InsecureSkipVerify bool          `json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
}
