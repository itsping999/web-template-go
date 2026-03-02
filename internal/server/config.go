package server

type Config struct {
	HTTP     HTTPConfig      `json:"http" yaml:"http"`
	GRPC     GRPCConfig      `json:"grpc" yaml:"grpc"`
	RabbitMQ RabbitMQConfig  `json:"rabbitmq" yaml:"rabbitmq"`
	WS       WebsocketConfig `json:"ws" yaml:"ws"`
	MQTT     MQTTConfig      `json:"mqtt" yaml:"mqtt"`
	TCP      TCPConfig       `json:"tcp" yaml:"tcp"`
	UDP      UDPConfig       `json:"udp" yaml:"udp"`
}
