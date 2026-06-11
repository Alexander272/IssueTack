package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
)

type UserRealmService struct {
	repo      repository.UserRealms
	txManager TransactionManager
}

func NewUserRealmService(repo repository.UserRealms, txManager TransactionManager) *UserRealmService {
	return &UserRealmService{
		repo:      repo,
		txManager: txManager,
	}
}

type UserRealms interface {
	GetAll(ctx context.Context) ([]*models.UserRealm, error)
	GetByUserId(ctx context.Context, userId string) ([]*models.UserRealm, error)
	Create(ctx context.Context, tx postgres.Tx, dto *models.UserRealmDTO) error
	CreateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.UserRealmDTO) error
	Update(ctx context.Context, tx postgres.Tx, dto *models.UserRealmDTO) error
	UpdateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.UserRealmDTO) error
	Delete(ctx context.Context, tx postgres.Tx, userId, realmId string) error
	DeleteSeveral(ctx context.Context, tx postgres.Tx, dto []*models.UserRealmDTO) error
}

func (s *UserRealmService) GetAll(ctx context.Context) ([]*models.UserRealm, error) {
	data, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all user realms. error: %w", err)
	}
	return data, nil
}

func (s *UserRealmService) GetByUserId(ctx context.Context, userId string) ([]*models.UserRealm, error) {
	data, err := s.repo.GetByUserId(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user realms by user id. error: %w", err)
	}
	return data, nil
}

func (s *UserRealmService) Create(ctx context.Context, tx postgres.Tx, dto *models.UserRealmDTO) error {
	if tx != nil {
		if err := s.repo.Create(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to create user realm. error: %w", err)
		}
		return nil
	}

	return s.txManager.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.Create(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to create user realm. error: %w", err)
		}
		return nil
	})
}

func (s *UserRealmService) CreateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.UserRealmDTO) error {
	if tx != nil {
		if err := s.repo.CreateSeveral(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to create several user realms. error: %w", err)
		}
		return nil
	}

	return s.txManager.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.CreateSeveral(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to create several user realms. error: %w", err)
		}
		return nil
	})
}

func (s *UserRealmService) Update(ctx context.Context, tx postgres.Tx, dto *models.UserRealmDTO) error {
	if tx != nil {
		if err := s.repo.Update(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to update user realm. error: %w", err)
		}
		return nil
	}

	return s.txManager.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.Update(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to update user realm. error: %w", err)
		}
		return nil
	})
}

func (s *UserRealmService) UpdateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.UserRealmDTO) error {
	if tx != nil {
		if err := s.repo.UpdateSeveral(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to update several user realms. error: %w", err)
		}
		return nil
	}

	return s.txManager.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.UpdateSeveral(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to update several user realms. error: %w", err)
		}
		return nil
	})
}

func (s *UserRealmService) Delete(ctx context.Context, tx postgres.Tx, userId, realmId string) error {
	if tx != nil {
		if err := s.repo.DeleteByUserAndRealm(ctx, tx, userId, realmId); err != nil {
			return fmt.Errorf("failed to delete user realm. error: %w", err)
		}
		return nil
	}

	return s.txManager.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.DeleteByUserAndRealm(ctx, tx, userId, realmId); err != nil {
			return fmt.Errorf("failed to delete user realm. error: %w", err)
		}
		return nil
	})
}

func (s *UserRealmService) DeleteSeveral(ctx context.Context, tx postgres.Tx, dto []*models.UserRealmDTO) error {
	if tx != nil {
		if err := s.repo.DeleteSeveral(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to delete several user realms. error: %w", err)
		}
		return nil
	}

	return s.txManager.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.DeleteSeveral(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to delete several user realms. error: %w", err)
		}
		return nil
	})
}
