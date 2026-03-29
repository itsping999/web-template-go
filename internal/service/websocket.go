package service

import (
	"net/url"

	"github.com/tx7do/kratos-transport/transport/websocket"
)

func (s *GreeterService) SetWebsocketServer(ws *websocket.Server) {
	s.ws = ws
}

func (s *GreeterService) OnWebsocketConnect(
	sessionId websocket.SessionID,
	queries url.Values,
	register bool,
) {
	if register {
		s.log.Infof("%s connected\n", sessionId)
	} else {
		s.log.Infof("%s disconnect\n", sessionId)
	}
}

func (s *GreeterService) OnChatMessage(_ websocket.SessionID, msg any) error {
	if s.ws == nil {
		return nil
	}
	s.ws.Broadcast(websocket.MessageType(1), msg)
	return nil
}
