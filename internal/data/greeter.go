package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/wyuhsin/web-template-go/api/helloworld/v1"
	"github.com/wyuhsin/web-template-go/internal/biz"
)

type greeterRepo struct {
	grpcClient v1.GreeterClient
	log        *log.Helper
}

// NewGreeterRepo .
func NewGreeterRepo(
	greeterClient v1.GreeterClient,
	logger log.Logger,
) biz.GreeterRepo {
	return &greeterRepo{
		grpcClient: greeterClient,
		log:        log.NewHelper(log.With(logger, "module", "data/greeter")),
	}
}

func (r *greeterRepo) Save(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
	_ = ctx
	return g, nil
}
