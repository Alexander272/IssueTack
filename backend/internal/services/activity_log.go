package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
)

// ActivityLogService — сервис журнала активности (аудит изменений тикетов, подзадач и вложений).
type ActivityLogService struct {
	repo      repository.ActivityLog
	txManager TransactionManager
}

// NewActivityLogService создаёт ActivityLogService.
func NewActivityLogService(repo repository.ActivityLog, txManager TransactionManager) *ActivityLogService {
	return &ActivityLogService{
		repo:      repo,
		txManager: txManager,
	}
}

// ActivityLog — интерфейс работы с журналом активности.
type ActivityLog interface {
	// Get возвращает записи журнала активности по критериям запроса.
	Get(ctx context.Context, req *models.GetLogsDTO) ([]*models.ActivityLog, error)
	// Create создаёт записи журнала активности.
	Create(ctx context.Context, tx postgres.Tx, dto []*models.ActivityLogDTO) error
}

// Get возвращает записи журнала активности по критериям запроса.
func (s *ActivityLogService) Get(ctx context.Context, req *models.GetLogsDTO) ([]*models.ActivityLog, error) {
	data, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity log. error: %w", err)
	}
	return data, nil
}

// Create создаёт записи журнала активности; если транзакция не передана, открывает новую.
func (s *ActivityLogService) Create(ctx context.Context, tx postgres.Tx, dto []*models.ActivityLogDTO) error {
	if len(dto) == 0 {
		return nil
	}

	if tx == nil {
		// Если транзакция не передана, создаем новую
		return s.txManager.WithinTransaction(ctx, func(newTx postgres.Tx) error {
			return s.executeCreate(ctx, newTx, dto)
		})
	}
	// Если транзакция передана, используем её
	return s.executeCreate(ctx, tx, dto)
}

// executeCreate — общая реализация создания записей журнала: вызывается как внутри
// переданной транзакции, так и внутри новой, открываемой методом Create, чтобы
// не дублировать логику записи и обработки ошибок.
func (s *ActivityLogService) executeCreate(ctx context.Context, tx postgres.Tx, dto []*models.ActivityLogDTO) error {
	// Журнал обязан нести realm_id: GET /activity-log фильтрует по авторизованному
	// реалму, «бесхозные» записи (NULL) не видит никто. Дозаполняем реалм по
	// сущности (тикет/подзадача); записи, которым реалм уже задан, не трогаем.
	for _, v := range dto {
		if v == nil || v.RealmID != nil {
			continue
		}
		realmID, realmName, ok, err := s.repo.GetRealmByEntity(ctx, tx, v.EntityType, v.EntityID)
		if err != nil {
			return fmt.Errorf("resolve realm for %s/%s: %w", v.EntityType, v.EntityID, err)
		}
		if !ok {
			// Неизвестный тип или осиротевшая сущность: реалма нет, запись не
			// попадёт ни в одну realm-выборку, но операцию журнала не роняем.
			continue
		}
		v.RealmID = &realmID
		if v.RealmName == "" {
			v.RealmName = realmName
		}
	}

	if err := s.repo.Create(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to create activity log. error: %w", err)
	}
	return nil
}
