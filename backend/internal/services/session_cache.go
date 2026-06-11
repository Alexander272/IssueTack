package services

import (
	"context"

	"github.com/Alexander272/IssueTrack/backend/internal/repository"
)

type SessionCacher interface {
	Get(ctx context.Context, userID string) map[string][]string
	Set(ctx context.Context, userID string, perms map[string][]string)
	Flush(ctx context.Context)
}

type sessionCacheService struct {
	repo repository.SessionCache
}

func NewSessionCacheService(repo repository.SessionCache) *sessionCacheService {
	return &sessionCacheService{repo: repo}
}

func (s *sessionCacheService) Get(ctx context.Context, userID string) map[string][]string {
	return s.repo.Get(ctx, userID)
}

func (s *sessionCacheService) Set(ctx context.Context, userID string, perms map[string][]string) {
	s.repo.Set(ctx, userID, perms)
}

func (s *sessionCacheService) Flush(ctx context.Context) {
	s.repo.Flush(ctx)
}
