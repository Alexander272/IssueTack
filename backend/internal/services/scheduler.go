package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/go-co-op/gocron/v2"
)

// SchedulerService управляет фоновыми cron-заданиями, в частности автозакрытием resolved-тикетов,
// уведомлениями о просроченных задачах и очисткой «закреплённых» (temporary) избранных.
type SchedulerService struct {
	cron          gocron.Scheduler
	tickets       Tickets
	notifications Notifications
	favorites     TicketFavorites
}

// SchedulerDeps содержит зависимости для создания SchedulerService.
type SchedulerDeps struct {
	Tickets       Tickets
	Notifications Notifications
	Favorites     TicketFavorites
}

// Scheduler описывает сервис планировщика фоновых заданий.
type Scheduler interface {
	// Start регистрирует фоновые задания и запускает планировщик.
	Start(conf *config.TicketConfig) error
	// Stop останавливает планировщик.
	Stop() error
}

// NewSchedulerService создаёт SchedulerService с инициализированным планировщиком.
func NewSchedulerService(deps *SchedulerDeps) *SchedulerService {
	cron, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalf("failed to create scheduler. error: %s", err.Error())
	}

	return &SchedulerService{
		cron:          cron,
		tickets:       deps.Tickets,
		notifications: deps.Notifications,
		favorites:     deps.Favorites,
	}
}

// Start регистрирует фоновые задания и запускает планировщик.
func (s *SchedulerService) Start(conf *config.TicketConfig) error {
	registered := false

	if conf.ResolvedToClosedAfter > 0 {
		_, err := s.cron.NewJob(
			gocron.CronJob(conf.AutoCloseSchedule, false),
			gocron.NewTask(s.autoCloseJob, conf.ResolvedToClosedAfter),
		)
		if err != nil {
			return fmt.Errorf("failed to create auto-close job. error: %w", err)
		}
		registered = true
		logger.Info("auto-close scheduler started, schedule: " + conf.AutoCloseSchedule)
	} else {
		logger.Info("auto-close of resolved tickets is disabled")
	}

	if conf.NotifyOverdueSchedule != "" {
		_, err := s.cron.NewJob(
			gocron.CronJob(conf.NotifyOverdueSchedule, false),
			gocron.NewTask(s.notifyOverdueJob),
		)
		if err != nil {
			return fmt.Errorf("failed to create overdue-notify job. error: %w", err)
		}
		registered = true
		logger.Info("overdue-notify scheduler started, schedule: " + conf.NotifyOverdueSchedule)
	}

	if conf.FavoriteCleanupSchedule != "" {
		_, err := s.cron.NewJob(
			gocron.CronJob(conf.FavoriteCleanupSchedule, false),
			gocron.NewTask(s.favoriteCleanupJob),
		)
		if err != nil {
			return fmt.Errorf("failed to create favorite-cleanup job. error: %w", err)
		}
		registered = true
		logger.Info("favorite-cleanup scheduler started, schedule: " + conf.FavoriteCleanupSchedule)
	}

	if registered {
		s.cron.Start()
	}
	return nil
}

// Stop останавливает планировщик.
func (s *SchedulerService) Stop() error {
	if err := s.cron.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown cron scheduler. error: %w", err)
	}
	return nil
}

// autoCloseJob — фоновая задача cron: автоматически закрывает resolved-тикеты, которые "зависли"
// дольше conf.ResolvedToClosedAfter после принятия решения.
func (s *SchedulerService) autoCloseJob(ctx context.Context, delay time.Duration) {
	n, err := s.tickets.AutoCloseResolved(ctx, delay)
	if err != nil {
		logger.Error("failed to auto-close resolved tickets:", logger.ErrAttr(err))
		return
	}
	if n > 0 {
		logger.Info(fmt.Sprintf("Auto-closed %d resolved tickets", n))
	}
}

// notifyOverdueJob — фоновая задача cron: ищет активные тикеты с просроченным сроком и оповещает
// о них заинтересованных пользователей (дедупликация выполняется в сервисе уведомлений).
func (s *SchedulerService) notifyOverdueJob(ctx context.Context) {
	ids, err := s.notifications.GetOverdueTicketIDs(ctx, time.Now())
	if err != nil {
		logger.Error("failed to get overdue tickets:", logger.ErrAttr(err))
		return
	}
	for _, id := range ids {
		ticket, err := s.tickets.GetSummary(ctx, id)
		if err != nil {
			logger.Error("failed to load overdue ticket", logger.StringAttr("ticket_id", id.String()), logger.ErrAttr(err))
			continue
		}
		if err := s.notifications.NotifyOverdue(ctx, ticket); err != nil {
			logger.Error("failed to notify overdue ticket:", logger.StringAttr("ticket_id", id.String()), logger.ErrAttr(err))
		}
	}
	if len(ids) > 0 {
		logger.Info(fmt.Sprintf("Notified overdue for %d tickets", len(ids)))
	}
}

// favoriteCleanupJob — фоновая задача cron: удаляет автоматически истёкшие «закреплённые»
// (temporary) избранные по правилам CleanupTemporary.
func (s *SchedulerService) favoriteCleanupJob(ctx context.Context) {
	if s.favorites == nil {
		return
	}
	n, err := s.favorites.CleanupTemporary(ctx)
	if err != nil {
		logger.Error("failed to cleanup temporary favorites:", logger.ErrAttr(err))
		return
	}
	if n > 0 {
		logger.Info(fmt.Sprintf("Cleaned up %d temporary favorites", n))
	}
}
