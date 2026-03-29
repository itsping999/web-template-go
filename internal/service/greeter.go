package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/transport/websocket"
	v1 "github.com/wyuhsin/web-template-go/api/helloworld/v1"
	"github.com/wyuhsin/web-template-go/internal/biz"
)

// GreeterService is a greeter service.
type GreeterService struct {
	v1.UnimplementedGreeterServer

	log *log.Helper
	uc  *biz.GreeterUsecase
	ws  *websocket.Server
}

// NewGreeterService new a greeter service.
func NewGreeterService(logger log.Logger, uc *biz.GreeterUsecase) *GreeterService {
	l := log.NewHelper(log.With(logger, "module", "service/greeter"))
	return &GreeterService{
		log: l,
		uc:  uc,
	}
}

// SayHello implements helloworld.GreeterServer.
func (s *GreeterService) SayHello(ctx context.Context, in *v1.HelloRequest) (*v1.HelloReply, error) {
	g, err := s.uc.CreateGreeter(ctx, &biz.Greeter{Hello: in.Name})
	if err != nil {
		return nil, err
	}
	return &v1.HelloReply{Message: "Hello " + g.Hello}, nil
}
