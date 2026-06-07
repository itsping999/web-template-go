package tests

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/wyuhsin/web-template-go/api/helloworld/v1"
	"github.com/wyuhsin/web-template-go/internal/biz"
	"github.com/wyuhsin/web-template-go/internal/data"
	"google.golang.org/grpc"
)

func TestGreeterRepoSaveUsesLocalAdapterWhenRemoteDisabled(t *testing.T) {
	repo := data.NewGreeterRepo(nil, log.NewStdLogger(io.Discard))

	out, err := repo.Save(context.Background(), &biz.Greeter{Name: "alice", Message: "Hello alice"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if out == nil || out.Name != "alice" || out.Message != "Hello alice" || out.Source != "local" {
		t.Fatalf("unexpected greeter: %+v", out)
	}
}

func TestGreeterRepoSaveUsesRemoteAdapterWhenConfigured(t *testing.T) {
	client := &fakeGreeterClient{
		reply: &v1.HelloReply{Message: "Hello from remote"},
	}
	repo := data.NewGreeterRepo(client, log.NewStdLogger(io.Discard))

	out, err := repo.Save(context.Background(), &biz.Greeter{Name: "alice", Message: "Hello alice"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if client.name != "alice" {
		t.Fatalf("expected remote request name alice, got %q", client.name)
	}
	if out == nil || out.Name != "alice" || out.Message != "Hello from remote" || out.Source != "remote" {
		t.Fatalf("unexpected greeter: %+v", out)
	}
}

func TestGreeterRepoSaveReturnsRemoteError(t *testing.T) {
	client := &fakeGreeterClient{err: errors.New("remote failed")}
	repo := data.NewGreeterRepo(client, log.NewStdLogger(io.Discard))

	_, err := repo.Save(context.Background(), &biz.Greeter{Name: "alice", Message: "Hello alice"})
	if err == nil {
		t.Fatalf("expected remote error")
	}
}

type fakeGreeterClient struct {
	name  string
	reply *v1.HelloReply
	err   error
}

func (f *fakeGreeterClient) SayHello(_ context.Context, in *v1.HelloRequest, _ ...grpc.CallOption) (*v1.HelloReply, error) {
	f.name = in.Name
	return f.reply, f.err
}
