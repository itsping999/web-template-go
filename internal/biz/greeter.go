package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	ReasonUserNotFound = "USER_NOT_FOUND"

	// ErrUserNotFound is user not found.
	ErrUserNotFound = errors.NotFound(ReasonUserNotFound, "user not found")
)

// Greeter is a Greeter model.
type Greeter struct {
	Hello string
}

// GreeterRepo is a Greater repo.
type GreeterRepo interface {
	Save(context.Context, *Greeter) (*Greeter, error)
}

type GreeterEventPublisher interface {
	PublishGreeterCreated(context.Context, *Greeter) error
}

// GreeterUsecase is a Greeter usecase.
type GreeterUsecase struct {
	repo      GreeterRepo
	publisher GreeterEventPublisher
	log       *log.Helper
}

// NewGreeterUsecase new a Greeter usecase.
func NewGreeterUsecase(
	repo GreeterRepo,
	publisher GreeterEventPublisher,
	logger log.Logger,
) *GreeterUsecase {
	return &GreeterUsecase{
		repo:      repo,
		publisher: publisher,
		log:       log.NewHelper(log.With(logger, "module", "biz/greeter")),
	}
}

// CreateGreeter creates a Greeter, and returns the new Greeter.
func (uc *GreeterUsecase) CreateGreeter(
	ctx context.Context,
	g *Greeter,
) (*Greeter, error) {
	uc.log.WithContext(ctx).Infof("CreateGreeter: %v", g.Hello)
	saved, err := uc.repo.Save(ctx, g)
	if err != nil {
		return nil, err
	}
	if uc.publisher != nil {
		if err = uc.publisher.PublishGreeterCreated(ctx, saved); err != nil {
			uc.log.WithContext(ctx).Warnf("publish greeter created event failed: %v", err)
		}
	}
	return saved, nil
}
