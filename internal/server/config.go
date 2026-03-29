package server

type Config struct {
	HTTP       HTTPConfig       `json:"http" yaml:"http"`
	GRPC       GRPCConfig       `json:"grpc" yaml:"grpc"`
	Middleware MiddlewareConfig `json:"middleware" yaml:"middleware"`
	WebSocket  WebSocketConfig  `json:"websocket" yaml:"websocket"`
	MQTT       MQTTConfig       `json:"mqtt" yaml:"mqtt"`
	RabbitMQ   RabbitMQConfig   `json:"rabbitmq" yaml:"rabbitmq"`
}

type MiddlewareConfig struct {
	RateLimitEnabled bool `json:"ratelimit_enabled" yaml:"ratelimit_enabled"`
}
