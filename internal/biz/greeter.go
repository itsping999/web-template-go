package biz

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	ReasonUserNotFound = "USER_NOT_FOUND"
	ReasonInvalidName  = "INVALID_NAME"

	// ErrUserNotFound is user not found.
	ErrUserNotFound = errors.NotFound(ReasonUserNotFound, "user not found")
	ErrInvalidName  = errors.BadRequest(ReasonInvalidName, "name is required")
)

// Greeter is a Greeter model.
type Greeter struct {
	Name    string
	Message string
	Source  string
}

// GreeterRepo stores or forwards Greeter domain data.
type GreeterRepo interface {
	Save(context.Context, *Greeter) (*Greeter, error)
}

// GreeterEventPublisher publishes optional Greeter domain events.
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

// CreateGreeter normalizes input, persists the greeting through the repo, and
// emits an optional event after the main flow succeeds.
func (uc *GreeterUsecase) CreateGreeter(ctx context.Context, name string) (*Greeter, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidName
	}

	g := &Greeter{
		Name:    name,
		Message: "Hello " + name,
	}
	uc.log.WithContext(ctx).Infof("CreateGreeter: %v", g.Name)
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
