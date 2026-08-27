package services

import (
	"context"

	"github.com/Alexander272/IssueTrack/backend/internal/repository"
)

// SessionCacher описывает кэш прав пользователя по сессии.
type SessionCacher interface {
	Get(ctx context.Context, userID string) map[string][]string
	Set(ctx context.Context, userID string, perms map[string][]string)
	Flush(ctx context.Context)
}

type sessionCacheService struct {
	repo repository.SessionCache
}

// NewSessionCacheService создаёт сервис кэширования прав сессий.
func NewSessionCacheService(repo repository.SessionCache) *sessionCacheService {
	return &sessionCacheService{repo: repo}
}

// Get возвращает кэшированные права пользователя.
func (s *sessionCacheService) Get(ctx context.Context, userID string) map[string][]string {
	return s.repo.Get(ctx, userID)
}

// Set сохраняет права пользователя в кэш.
func (s *sessionCacheService) Set(ctx context.Context, userID string, perms map[string][]string) {
	s.repo.Set(ctx, userID, perms)
}

// Flush очищает весь кэш прав сессий.
func (s *sessionCacheService) Flush(ctx context.Context) {
	s.repo.Flush(ctx)
}
