package server

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tx7do/kratos-transport/transport/websocket"

	"github.com/wyuhsin/web-template-go/internal/service"
)

type WebsocketConfig struct {
	Addr    string        `json:"addr" yaml:"addr"`
	Path    string        `json:"path" yaml:"path"`
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
}

// NewWebsocketServer create a websocket server.
func NewWebsocketServer(c *Config, _ *logrus.Entry, svc *service.GreeterService) *websocket.Server {
	srv := websocket.NewServer(
		websocket.WithAddress(c.WS.Addr),
		websocket.WithPath(c.WS.Path),
		websocket.WithConnectHandle(svc.OnWebsocketConnect),
		websocket.WithCodec("json"),
	)

	svc.SetWebsocketServer(srv)

	srv.RegisterMessageHandler(1,
		func(sessionId websocket.SessionID, payload websocket.MessagePayload) error {
			switch t := payload.(type) {
			case any:
				return svc.OnChatMessage(sessionId, t)
			default:
				return fmt.Errorf("unsupported type: %T", t)
			}
		},
		func() websocket.Any { return struct{}{} },
	)

	return srv
}
