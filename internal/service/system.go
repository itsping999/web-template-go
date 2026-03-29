package service

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"gorm.io/gorm"
)

// SystemService provides basic service runtime endpoints.
type SystemService struct {
	startedAt   time.Time
	postgresDB  *gorm.DB
	redisClient *redis.Client
	mongoClient *mongo.Client
	mu          sync.RWMutex
	cacheAt     time.Time
	cacheTTL    time.Duration
	cacheReady  bool
	cacheChecks map[string]string
}

func NewSystemService(
	postgresDB *gorm.DB,
	redisClient *redis.Client,
	mongoClient *mongo.Client,
) *SystemService {
	return &SystemService{
		startedAt:   time.Now(),
		postgresDB:  postgresDB,
		redisClient: redisClient,
		mongoClient: mongoClient,
		cacheTTL:    2 * time.Second,
	}
}

func (s *SystemService) Liveness() map[string]any {
	return map[string]any{
		"status": "ok",
	}
}

func (s *SystemService) Readiness() map[string]any {
	ready, dbChecks := s.readinessSnapshot()
	status := "ready"
	if !ready {
		status = "not_ready"
	}
	return map[string]any{
		"status": status,
		"checks": dbChecks,
	}
}

func (s *SystemService) Metadata() map[string]any {
	return map[string]any{
		"status":     "running",
		"started_at": s.startedAt.UTC().Format(time.RFC3339),
	}
}

func (s *SystemService) IsReady() bool {
	ready, _ := s.readinessSnapshot()
	return ready
}

func (s *SystemService) ReadinessStatus() (bool, map[string]any) {
	ready, checks := s.readinessSnapshot()
	status := "ready"
	if !ready {
		status = "not_ready"
	}
	return ready, map[string]any{
		"status": status,
		"checks": checks,
	}
}

func (s *SystemService) checkPostgres() string {
	if s.postgresDB == nil {
		return "skipped"
	}
	sqlDB, err := s.postgresDB.DB()
	if err != nil {
		return "down"
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return "down"
	}
	return "up"
}

func (s *SystemService) checkRedis() string {
	if s.redisClient == nil {
		return "skipped"
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.redisClient.Ping(ctx).Err(); err != nil {
		return "down"
	}
	return "up"
}

func (s *SystemService) checkMongoDB() string {
	if s.mongoClient == nil {
		return "skipped"
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.mongoClient.Ping(ctx, readpref.Primary()); err != nil {
		return "down"
	}
	return "up"
}

func (s *SystemService) readinessSnapshot() (bool, map[string]string) {
	now := time.Now()
	s.mu.RLock()
	if !s.cacheAt.IsZero() && now.Sub(s.cacheAt) < s.cacheTTL {
		ready := s.cacheReady
		checks := copyChecks(s.cacheChecks)
		s.mu.RUnlock()
		return ready, checks
	}
	s.mu.RUnlock()

	checks := map[string]string{
		"postgres": s.checkPostgres(),
		"redis":    s.checkRedis(),
		"mongodb":  s.checkMongoDB(),
	}
	ready := checks["postgres"] != "down" && checks["redis"] != "down" && checks["mongodb"] != "down"

	s.mu.Lock()
	s.cacheAt = now
	s.cacheReady = ready
	s.cacheChecks = copyChecks(checks)
	s.mu.Unlock()
	return ready, checks
}

func copyChecks(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
