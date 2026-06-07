package tests

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/wyuhsin/web-template-go/api/helloworld/v1"
	"github.com/wyuhsin/web-template-go/internal/biz"
	"github.com/wyuhsin/web-template-go/internal/service"
)

func TestGreeterServiceSayHelloMapsTransportToUsecase(t *testing.T) {
	repo := &mockGreeterRepo{
		saveFn: func(_ context.Context, g *biz.Greeter) (*biz.Greeter, error) {
			return &biz.Greeter{
				Name:    g.Name,
				Message: "Hello from adapter",
				Source:  "test",
			}, nil
		},
	}
	uc := biz.NewGreeterUsecase(repo, nil, log.NewStdLogger(io.Discard))
	svc := service.NewGreeterService(log.NewStdLogger(io.Discard), uc)

	reply, err := svc.SayHello(context.Background(), &v1.HelloRequest{Name: " alice "})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if reply.GetMessage() != "Hello from adapter" {
		t.Fatalf("expected service to return usecase message, got %q", reply.GetMessage())
	}
}
