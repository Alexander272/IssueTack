package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

// statisticsBucketLimit — предел числа строк в разрезах «топ-N» (категории, группы,
// площадки). Хвост длинного распределения не отбрасывается: остальные заявки учтены
// в KPI-карточках и динамике. Разрезы «заказчики»/«исполнители» идут без лимита:
// фронт сворачивает хвост в «Прочие», и для этого ему нужен полный хвост.
const statisticsBucketLimit = 20

// Statistics — сервис статистики заявок. Срез видимых заявок вычисляется по роли
// актора (StatisticsScope): начальник области видит реалм, менеджер — управляемые
// группы (и свои назначения), исполнитель — свои назначения, заявитель — свои заявки.
type Statistics interface {
	// Get возвращает агрегаты статистики заявок за период с учётом среза актора.
	Get(ctx context.Context, filter *models.StatisticsFilter) (*models.TicketStatistics, error)
	// GetTickets — drill-down статистики: заявки, стоящие за сектором диаграммы
	// (нагрузка исполнителя / задачи от заказчика) с учётом того же среза и периода.
	GetTickets(ctx context.Context, query models.StatisticsTicketsQuery) ([]*models.Ticket, int, error)
}

// StatisticsService реализует Statistics.
type StatisticsService struct {
	repo   repository.Statistics
	groups Groups
	access TicketAccessChecker
}

// NewStatisticsService создаёт сервис статистики.
func NewStatisticsService(repo repository.Statistics, groups Groups, access TicketAccessChecker) *StatisticsService {
	return &StatisticsService{
		repo:   repo,
		groups: groups,
		access: access,
	}
}

// Get собирает полный ответ страницы статистики. Все секции инициализируются
// пустыми срезами: фронт считает их массивами, nil превратился бы в JSON null.
//
// Запросы к БД независимы (у каждого свой whereBuilder), поэтому выполняются
// параллельно через errgroup. Горутины пишут в разные поля stats — общих ячеек
// нет, гонок не возникает. Контекст отменяет остальные запросы при первой ошибке.
func (s *StatisticsService) Get(ctx context.Context, filter *models.StatisticsFilter) (*models.TicketStatistics, error) {
	if filter.Actor == nil {
		return nil, models.ErrPermissionDenied
	}

	scope, err := s.resolveScope(ctx, filter.Actor.ID, filter.RealmID)
	if err != nil {
		return nil, err
	}

	granularity := filter.Granularity
	if granularity == "" {
		granularity = autoGranularity(filter.From, filter.To)
	}

	query := models.StatisticsQuery{
		Scope:       scope,
		Filter:      filter,
		Limit:       statisticsBucketLimit,
		Granularity: granularity,
	}

	stats := &models.TicketStatistics{
		ByGroup:  []*models.StatisticsBucket{},
		ByOwner:  []*models.StatisticsBreakdownBucket{},
		Workload: []*models.WorkloadBucket{},
	}

	// Разрезы «заказчики» и «исполнители» на фронте показываются как топ-N
	// (максимум 20) + «Прочие»: хвост нужен для точного «Прочие», поэтому
	// Limit = 0 — репозиторий не добавляет LIMIT в SQL.
	breakdownQuery := query
	breakdownQuery.Limit = 0

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		summary, err := s.repo.GetStatisticsSummary(gctx, query)
		if err != nil {
			return fmt.Errorf("failed to get statistics summary: %w", err)
		}
		stats.Summary = *summary
		return nil
	})
	g.Go(func() error {
		byCategory, err := s.repo.GetStatisticsByDimension(gctx, query.WithDimension("category"))
		if err != nil {
			return fmt.Errorf("failed to get statistics by category: %w", err)
		}
		stats.ByCategory = byCategory
		return nil
	})
	g.Go(func() error {
		bySite, err := s.repo.GetStatisticsByDimension(gctx, query.WithDimension("site"))
		if err != nil {
			return fmt.Errorf("failed to get statistics by site: %w", err)
		}
		stats.BySite = bySite
		return nil
	})
	g.Go(func() error {
		byStatus, err := s.repo.GetStatisticsByStatus(gctx, query)
		if err != nil {
			return fmt.Errorf("failed to get statistics by status: %w", err)
		}
		stats.ByStatus = byStatus
		return nil
	})
	g.Go(func() error {
		trend, err := s.repo.GetStatisticsTrend(gctx, query)
		if err != nil {
			return fmt.Errorf("failed to get statistics trend: %w", err)
		}
		stats.Trend = trend
		return nil
	})

	// Управленческие разрезы — только начальнику области и менеджерам групп.
	if scope.IsManagerial() {
		g.Go(func() error {
			byGroup, err := s.repo.GetStatisticsByDimension(gctx, query.WithDimension("group"))
			if err != nil {
				return fmt.Errorf("failed to get statistics by group: %w", err)
			}
			stats.ByGroup = byGroup
			return nil
		})
		g.Go(func() error {
			byOwner, err := s.repo.GetStatisticsByOwner(gctx, breakdownQuery)
			if err != nil {
				return fmt.Errorf("failed to get statistics by owner: %w", err)
			}
			stats.ByOwner = byOwner
			return nil
		})
		g.Go(func() error {
			workload, err := s.repo.GetStatisticsWorkload(gctx, breakdownQuery)
			if err != nil {
				return fmt.Errorf("failed to get statistics workload: %w", err)
			}
			stats.Workload = workload
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return stats, nil
}

// GetTickets — drill-down статистики: заявки, стоящие за сектором диаграммы
// «Нагрузка исполнителей» / «Задачи от заказчиков». Срез актора и уточнения
// фильтра те же, что у агрегатов, поэтому список совпадает с числом в секции.
// Как и Get, требует актора: без него срез доступа не вычислить.
func (s *StatisticsService) GetTickets(ctx context.Context, query models.StatisticsTicketsQuery) ([]*models.Ticket, int, error) {
	if query.Filter == nil || query.Filter.Actor == nil {
		return nil, 0, models.ErrPermissionDenied
	}

	scope, err := s.resolveScope(ctx, query.Filter.Actor.ID, query.Filter.RealmID)
	if err != nil {
		return nil, 0, err
	}
	query.Scope = scope

	return s.repo.GetStatisticsTickets(ctx, query)
}

// resolveScope определяет срез видимых актору заявок. Приоритет ролей:
// начальник области → менеджер управляемых групп → участник/исполнитель → заявитель.
// Менеджер, помимо управляемых групп, видит и личные назначения: он часто остаётся
// исполнителем, и «статистика группы» не должна прятать его собственную работу.
func (s *StatisticsService) resolveScope(ctx context.Context, userID uuid.UUID, realmID uuid.UUID) (models.StatisticsScope, error) {
	scope := models.StatisticsScope{RealmID: realmID}

	supervisor, err := s.access.IsRealmSupervisor(ctx, userID, realmID.String())
	if err != nil {
		return scope, fmt.Errorf("realm supervisor check failed: %w", err)
	}
	if supervisor {
		scope.AllRealm = true
		return scope, nil
	}

	managed, err := s.groups.GetManagedGroups(ctx, userID, &realmID)
	if err != nil {
		return scope, fmt.Errorf("failed to get managed groups: %w", err)
	}
	if len(managed) > 0 {
		scope.GroupIDs = managed
		scope.AssigneeID = &userID
		return scope, nil
	}

	member, err := s.groups.GetMemberGroups(ctx, userID, &realmID)
	if err != nil {
		return scope, fmt.Errorf("failed to get member groups: %w", err)
	}
	if len(member) > 0 {
		scope.AssigneeID = &userID
		return scope, nil
	}

	scope.RequesterID = &userID
	return scope, nil
}

// autoGranularity выбирает шаг бакетов динамики по длине периода.
func autoGranularity(from, to time.Time) string {
	days := to.Sub(from).Hours() / 24
	switch {
	case days <= 31:
		return "day"
	case days <= 180:
		return "week"
	default:
		return "month"
	}
}
