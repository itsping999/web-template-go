package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/sirupsen/logrus"
	v1 "github.com/wyuhsin/web-template-go/api/helloworld/v1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var (
	// ErrUserNotFound is user not found.
	ErrUserNotFound = errors.NotFound(v1.ErrorReason_USER_NOT_FOUND.String(), "user not found")
)

// Greeter is a Greeter model.
type Greeter struct {
	Hello string
}

// GreeterRepo is a Greater repo.
type GreeterRepo interface {
	Save(context.Context, *Greeter) (*Greeter, error)
	Update(context.Context, *Greeter) (*Greeter, error)
	FindByID(context.Context, int64) (*Greeter, error)
	ListByHello(context.Context, string) ([]*Greeter, error)
	ListAll(context.Context) ([]*Greeter, error)
}

// GreeterUsecase is a Greeter usecase.
type GreeterUsecase struct {
	repo GreeterRepo
	log  *logrus.Entry
}

// NewGreeterUsecase new a Greeter usecase.
func NewGreeterUsecase(repo GreeterRepo, logger *logrus.Entry) *GreeterUsecase {
	return &GreeterUsecase{repo: repo, log: logger.WithField("module", "biz/greeter")}
}

// CreateGreeter creates a Greeter, and returns the new Greeter.
func (uc *GreeterUsecase) CreateGreeter(
	ctx context.Context,
	g *Greeter,
) (*Greeter, error) {
	ctx, span := otel.Tracer("biz.greeter").Start(ctx, "GreeterUsecase.CreateGreeter")
	span.SetAttributes(attribute.String("greeter.hello", g.Hello))
	defer span.End()

	uc.log.WithContext(ctx).Infof("CreateGreeter: %v", g.Hello)
	return uc.repo.Save(ctx, g)
}
