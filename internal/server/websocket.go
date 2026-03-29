package server

import (
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/transport/websocket"

	"github.com/wyuhsin/web-template-go/internal/service"
)

type WebSocketConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Addr    string `json:"addr" yaml:"addr"`
	Path    string `json:"path" yaml:"path"`
}

// NewWebsocketServer create a websocket server.
func NewWebsocketServer(
	rt *WebSocketConfig,
	_ log.Logger,
	svc *service.GreeterService,
) *websocket.Server {
	if rt == nil || !rt.Enabled {
		return nil
	}
	addr := strings.TrimSpace(rt.Addr)
	path := strings.TrimSpace(rt.Path)
	if addr == "" || path == "" {
		return nil
	}
	srv := websocket.NewServer(
		websocket.WithAddress(addr),
		websocket.WithPath(path),
		websocket.WithConnectHandle(svc.OnWebsocketConnect),
		websocket.WithCodec("json"),
	)

	svc.SetWebsocketServer(srv)

	srv.RegisterMessageHandler(1,
		func(sessionId websocket.SessionID, payload websocket.MessagePayload) error {
			return svc.OnChatMessage(sessionId, payload)
		},
		func() websocket.Any { return struct{}{} },
	)

	return srv
}
