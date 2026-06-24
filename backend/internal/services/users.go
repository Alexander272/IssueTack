package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/pkg/auth"
	"github.com/google/uuid"
)

type userService struct {
	repo      repository.Users
	tm        TransactionManager
	keycloak  *auth.KeycloakClient
	userRealm UserRealms
	eventBus  *events.PolicyEventManager
}

type UsersDeps struct {
	Repo      repository.Users
	TxManager TransactionManager
	UserRealm UserRealms
	Keycloak  *auth.KeycloakClient
	EventBus  *events.PolicyEventManager
}

func NewUserService(deps *UsersDeps) *userService {
	return &userService{
		repo:      deps.Repo,
		tm:        deps.TxManager,
		userRealm: deps.UserRealm,
		keycloak:  deps.Keycloak,
		eventBus:  deps.EventBus,
	}
}

type Users interface {
	LoadPolicy(ctx context.Context) ([]*models.UserRole, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.UserData, error)
	GetByLogin(ctx context.Context, login string) (*models.UserData, error)
	GetAll(ctx context.Context, realmID *uuid.UUID) ([]*models.UserData, error)
	Sync(ctx context.Context, actor *models.Actor) error
	UpdateAccount(ctx context.Context, dto *models.UpdateAccountDTO) error
}

func (s *userService) LoadPolicy(ctx context.Context) ([]*models.UserRole, error) {
	data, err := s.repo.LoadPolicy(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load policy: %w", err)
	}
	return data, nil
}

func (s *userService) GetByID(ctx context.Context, id uuid.UUID) (*models.UserData, error) {
	data, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id. error: %w", err)
	}
	return data, nil
}

func (s *userService) GetByLogin(ctx context.Context, login string) (*models.UserData, error) {
	data, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by login. error: %w", err)
	}
	return data, nil
}

func (s *userService) GetAll(ctx context.Context, realmID *uuid.UUID) ([]*models.UserData, error) {
	data, err := s.repo.GetAll(ctx, realmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users. error: %w", err)
	}
	return data, nil
}
