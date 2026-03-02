package conf

import (
	"github.com/wyuhsin/web-template-go/internal/data"
	"github.com/wyuhsin/web-template-go/internal/pkg/logx"
	"github.com/wyuhsin/web-template-go/internal/pkg/tracingx"
	"github.com/wyuhsin/web-template-go/internal/server"
)

type Bootstrap struct {
	Server  server.Config   `json:"server" yaml:"server"`
	Data    data.Config     `json:"data" yaml:"data"`
	Logger  logx.Options    `json:"logger" yaml:"logger"`
	Tracing tracingx.Config `json:"tracing" yaml:"tracing"`
}
