package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
)

type RealmService struct {
	repo      repository.Realms
	txManager TransactionManager
}

func NewRealmService(repo repository.Realms, txManager TransactionManager) *RealmService {
	return &RealmService{
		repo:      repo,
		txManager: txManager,
	}
}

type Realms interface{}

func (s *RealmService) GetByID(ctx context.Context, req *models.GetRealmByIdDTO) (*models.Realm, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get realm by id. error: %w", err)
	}
	return data, nil
}

func (s *RealmService) Create(ctx context.Context, dto *models.RealmDTO) error {
	return s.txManager.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.Create(ctx, nil, dto); err != nil {
			return fmt.Errorf("failed to create realm. error: %w", err)
		}

		//TODO создать несколько системных ролей, надо бы еще наверное вынести их куда-нибудь

		return nil
	})
}

func (s *RealmService) Update(ctx context.Context, dto *models.RealmDTO) error {
	if err := s.repo.Update(ctx, nil, dto); err != nil {
		return fmt.Errorf("failed to update realm. error: %w", err)
	}
	return nil
}

func (s *RealmService) Delete(ctx context.Context, dto *models.DeleteRealmDTO) error {
	//? может надо бы как-то ограничить удаление realm
	if err := s.repo.Delete(ctx, nil, dto); err != nil {
		return fmt.Errorf("failed to delete realm. error: %w", err)
	}
	return nil
}
