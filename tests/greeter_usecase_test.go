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
	called bool
	saveFn func(context.Context, *biz.Greeter) (*biz.Greeter, error)
}

func (m *mockGreeterRepo) Save(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
	m.called = true
	return m.saveFn(ctx, g)
}

type mockPublisher struct {
	called  bool
	greeter *biz.Greeter
	err     error
}

func (m *mockPublisher) PublishGreeterCreated(_ context.Context, g *biz.Greeter) error {
	m.called = true
	m.greeter = g
	return m.err
}

func TestCreateGreeterNormalizesNameAndPublishesEvent(t *testing.T) {
	repo := &mockGreeterRepo{
		saveFn: func(_ context.Context, g *biz.Greeter) (*biz.Greeter, error) {
			if g.Name != "alice" {
				t.Fatalf("expected normalized name, got %q", g.Name)
			}
			if g.Message != "Hello alice" {
				t.Fatalf("expected default message, got %q", g.Message)
			}
			return g, nil
		},
	}
	pub := &mockPublisher{}
	uc := biz.NewGreeterUsecase(repo, pub, log.NewStdLogger(io.Discard))

	out, err := uc.CreateGreeter(context.Background(), " alice ")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if out == nil || out.Name != "alice" || out.Message != "Hello alice" {
		t.Fatalf("unexpected saved greeter: %+v", out)
	}
	if !pub.called {
		t.Fatalf("expected publisher to be called")
	}
	if pub.greeter == nil || pub.greeter.Name != "alice" {
		t.Fatalf("unexpected published greeter: %+v", pub.greeter)
	}
}

func TestCreateGreeterRejectsBlankName(t *testing.T) {
	repo := &mockGreeterRepo{
		saveFn: func(_ context.Context, g *biz.Greeter) (*biz.Greeter, error) {
			return g, nil
		},
	}
	uc := biz.NewGreeterUsecase(repo, nil, log.NewStdLogger(io.Discard))

	if _, err := uc.CreateGreeter(context.Background(), " "); err == nil {
		t.Fatalf("expected error for blank name")
	}
	if repo.called {
		t.Fatalf("repo should not be called for invalid input")
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

	out, err := uc.CreateGreeter(context.Background(), "bob")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if out == nil || out.Name != "bob" || out.Message != "Hello bob" {
		t.Fatalf("unexpected saved greeter: %+v", out)
	}
	if !pub.called {
		t.Fatalf("expected publisher to be called")
	}
}
