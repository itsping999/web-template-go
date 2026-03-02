package data

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/wyuhsin/web-template-go/internal/biz"
	"github.com/wyuhsin/web-template-go/internal/pkg/dbx"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"
)

type greeterRepo struct {
	mysqlDB *gorm.DB
	redis   *redis.Client
	log     *logrus.Entry
}

// NewGreeterRepo .
func NewGreeterRepo(mysqlDB dbx.MySQLDB, redisClient *redis.Client, logger *logrus.Entry) biz.GreeterRepo {
	return &greeterRepo{
		mysqlDB: mysqlDB.Conn,
		redis:   redisClient,
		log:     logger.WithField("module", "data/greeter"),
	}
}

func (r *greeterRepo) Save(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
	_, span := otel.Tracer("data.greeter").Start(ctx, "GreeterRepo.Save")
	span.SetAttributes(attribute.String("greeter.hello", g.Hello))
	defer span.End()
	return g, nil
}

func (r *greeterRepo) Update(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
	_, span := otel.Tracer("data.greeter").Start(ctx, "GreeterRepo.Update")
	span.SetAttributes(attribute.String("greeter.hello", g.Hello))
	defer span.End()
	return g, nil
}

func (r *greeterRepo) FindByID(ctx context.Context, id int64) (*biz.Greeter, error) {
	_, span := otel.Tracer("data.greeter").Start(ctx, "GreeterRepo.FindByID")
	span.SetAttributes(attribute.Int64("greeter.id", id))
	defer span.End()
	return nil, nil
}

func (r *greeterRepo) ListByHello(ctx context.Context, hello string) ([]*biz.Greeter, error) {
	_, span := otel.Tracer("data.greeter").Start(ctx, "GreeterRepo.ListByHello")
	span.SetAttributes(attribute.String("greeter.hello", hello))
	defer span.End()
	return nil, nil
}

func (r *greeterRepo) ListAll(ctx context.Context) ([]*biz.Greeter, error) {
	_, span := otel.Tracer("data.greeter").Start(ctx, "GreeterRepo.ListAll")
	defer span.End()
	return nil, nil
}
