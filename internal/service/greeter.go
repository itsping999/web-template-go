package service

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/tx7do/kratos-transport/transport/rabbitmq"
	"github.com/tx7do/kratos-transport/transport/websocket"
	v1 "github.com/wyuhsin/web-template-go/api/helloworld/v1"
	"github.com/wyuhsin/web-template-go/internal/biz"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// GreeterService is a greeter service.
type GreeterService struct {
	v1.UnimplementedGreeterServer

	log *logrus.Entry
	uc  *biz.GreeterUsecase
	ws  *websocket.Server
	mq  *rabbitmq.Server
}

// NewGreeterService new a greeter service.
func NewGreeterService(logger *logrus.Entry, uc *biz.GreeterUsecase) *GreeterService {
	l := logger.WithField("module", "service/greeter")
	return &GreeterService{
		log: l,
		uc:  uc,
	}
}

// SayHello implements helloworld.GreeterServer.
func (s *GreeterService) SayHello(ctx context.Context, in *v1.HelloRequest) (*v1.HelloReply, error) {
	ctx, span := otel.Tracer("service.greeter").Start(ctx, "GreeterService.SayHello")
	span.SetAttributes(attribute.String("hello.name", in.GetName()))
	defer span.End()

	g, err := s.uc.CreateGreeter(ctx, &biz.Greeter{Hello: in.Name})
	if err != nil {
		return nil, err
	}
	return &v1.HelloReply{Message: "Hello " + g.Hello}, nil
}
