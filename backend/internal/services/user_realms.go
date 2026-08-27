package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

// UserRealmService управляет привязками пользователей к realm'ам и их ролям в них.
type UserRealmService struct {
	repo      repository.UserRealms
	txManager TransactionManager
}

// NewUserRealmService создаёт сервис привязок пользователей к realm'ам.
func NewUserRealmService(repo repository.UserRealms, txManager TransactionManager) *UserRealmService {
	return &UserRealmService{
		repo:      repo,
		txManager: txManager,
	}
}

// UserRealms описывает сервис управления привязками пользователей к realm'ам.
type UserRealms interface {
	GetAll(ctx context.Context) ([]*models.UserRealm, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.UserRealm, error)
	GetByUserAndRealm(ctx context.Context, userID, realmID uuid.UUID) (*models.UserRealm, error)
	Create(ctx context.Context, tx postgres.Tx, dto *models.UserRealmDTO) error
	CreateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.UserRealmDTO) error
	Update(ctx context.Context, tx postgres.Tx, dto *models.UserRealmDTO) error
	UpdateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.UserRealmDTO) error
	Delete(ctx context.Context, tx postgres.Tx, userID, realmID uuid.UUID) error
	DeleteSeveral(ctx context.Context, tx postgres.Tx, dto []*models.UserRealmDTO) error
}

// GetAll возвращает все привязки пользователей к realm'ам.
func (s *UserRealmService) GetAll(ctx context.Context) ([]*models.UserRealm, error) {
	data, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all user realms. error: %w", err)
	}
	return data, nil
}

// GetByUserID возвращает привязки пользователя ко всем realm'ам.
func (s *UserRealmService) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.UserRealm, error) {
	data, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user realms by user id. error: %w", err)
	}
	return data, nil
}

// GetByUserAndRealm возвращает привязку пользователя к конкретному realm'у.
func (s *UserRealmService) GetByUserAndRealm(ctx context.Context, userID, realmID uuid.UUID) (*models.UserRealm, error) {
	data, err := s.repo.GetByUserAndRealm(ctx, userID, realmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user realm by user and realm. error: %w", err)
	}
	return data, nil
}

// Create создаёт привязку пользователя к realm'у, при необходимости в рамках переданной транзакции.
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

// CreateSeveral создаёт несколько привязок пользователей к realm'ам, при необходимости в рамках переданной транзакции.
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

// Update обновляет привязку пользователя к realm'у, при необходимости в рамках переданной транзакции.
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

// UpdateSeveral обновляет несколько привязок пользователей к realm'ам, при необходимости в рамках переданной транзакции.
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

// Delete удаляет привязку пользователя к realm'у, при необходимости в рамках переданной транзакции.
func (s *UserRealmService) Delete(ctx context.Context, tx postgres.Tx, userID, realmID uuid.UUID) error {
	if tx != nil {
		if err := s.repo.DeleteByUserAndRealm(ctx, tx, userID, realmID); err != nil {
			return fmt.Errorf("failed to delete user realm. error: %w", err)
		}
		return nil
	}

	return s.txManager.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.DeleteByUserAndRealm(ctx, tx, userID, realmID); err != nil {
			return fmt.Errorf("failed to delete user realm. error: %w", err)
		}
		return nil
	})
}

// DeleteSeveral удаляет несколько привязок пользователей к realm'ам, при необходимости в рамках переданной транзакции.
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
