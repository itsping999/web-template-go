package tests

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/wyuhsin/web-template-go/internal/biz"
)

type mockGreeterRepo struct {
	saveFn func(context.Context, *biz.Greeter) (*biz.Greeter, error)
}

func (m *mockGreeterRepo) Save(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
	return m.saveFn(ctx, g)
}

func (m *mockGreeterRepo) Update(context.Context, *biz.Greeter) (*biz.Greeter, error) {
	return nil, nil
}

func (m *mockGreeterRepo) FindByID(context.Context, int64) (*biz.Greeter, error) {
	return nil, nil
}

func (m *mockGreeterRepo) ListByHello(context.Context, string) ([]*biz.Greeter, error) {
	return nil, nil
}

func (m *mockGreeterRepo) ListAll(context.Context) ([]*biz.Greeter, error) {
	return nil, nil
}

type mockPublisher struct {
	called bool
	err    error
}

func (m *mockPublisher) PublishGreeterCreated(_ context.Context, _ *biz.Greeter) error {
	m.called = true
	return m.err
}

func TestCreateGreeterPublishesEvent(t *testing.T) {
	repo := &mockGreeterRepo{
		saveFn: func(_ context.Context, g *biz.Greeter) (*biz.Greeter, error) {
			return g, nil
		},
	}
	pub := &mockPublisher{}
	uc := biz.NewGreeterUsecase(repo, pub, log.NewStdLogger(io.Discard))

	out, err := uc.CreateGreeter(context.Background(), &biz.Greeter{Hello: "alice"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if out == nil || out.Hello != "alice" {
		t.Fatalf("unexpected saved greeter: %+v", out)
	}
	if !pub.called {
		t.Fatalf("expected publisher to be called")
	}
}

func TestCreateGreeterPublishFailureDoesNotBreakMainFlow(t *testing.T) {
	repo := &mockGreeterRepo{
		saveFn: func(_ context.Context, g *biz.Greeter) (*biz.Greeter, error) {
			return g, nil
		},
	}
	pub := &mockPublisher{err: errors.New("publish failed")}
	uc := biz.NewGreeterUsecase(repo, pub, log.NewStdLogger(io.Discard))

	out, err := uc.CreateGreeter(context.Background(), &biz.Greeter{Hello: "bob"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if out == nil || out.Hello != "bob" {
		t.Fatalf("unexpected saved greeter: %+v", out)
	}
	if !pub.called {
		t.Fatalf("expected publisher to be called")
	}
}
