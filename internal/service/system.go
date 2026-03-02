package service

import "time"

// SystemService provides basic service runtime endpoints.
type SystemService struct {
	startedAt time.Time
}

func NewSystemService() *SystemService {
	return &SystemService{
		startedAt: time.Now(),
	}
}

func (s *SystemService) Liveness() map[string]any {
	return map[string]any{
		"status": "ok",
	}
}

func (s *SystemService) Readiness() map[string]any {
	return map[string]any{
		"status": "ready",
	}
}

func (s *SystemService) Metadata() map[string]any {
	return map[string]any{
		"status":     "running",
		"started_at": s.startedAt.UTC().Format(time.RFC3339),
	}
}
