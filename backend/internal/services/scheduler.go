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

// SchedulerService управляет фоновыми cron-заданиями, в частности автозакрытием resolved-тикетов.
type SchedulerService struct {
	cron    gocron.Scheduler
	tickets Tickets
}

// SchedulerDeps содержит зависимости для создания SchedulerService.
type SchedulerDeps struct {
	Tickets Tickets
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
		cron:    cron,
		tickets: deps.Tickets,
	}
}

// Start регистрирует фоновые задания и запускает планировщик.
func (s *SchedulerService) Start(conf *config.TicketConfig) error {
	if conf.ResolvedToClosedAfter <= 0 {
		logger.Info("auto-close of resolved tickets is disabled")
		return nil
	}

	_, err := s.cron.NewJob(
		gocron.CronJob(conf.AutoCloseSchedule, false),
		gocron.NewTask(s.job, conf.ResolvedToClosedAfter),
	)
	if err != nil {
		return fmt.Errorf("failed to create auto-close job. error: %w", err)
	}

	s.cron.Start()
	logger.Info("auto-close scheduler started, schedule: " + conf.AutoCloseSchedule)
	return nil
}

// Stop останавливает планировщик.
func (s *SchedulerService) Stop() error {
	if err := s.cron.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown cron scheduler. error: %w", err)
	}
	return nil
}

// job — фоновая задача cron: автоматически закрывает resolved-тикеты, которые "зависли"
// дольше conf.ResolvedToClosedAfter после принятия решения. Автозакрытие нужно, чтобы
// очередь не засорялась давно решёнными заявками. Ошибку не пробрасываем наверх, а только
// логируем: единичный сбой (например, недоступность БД) не должен ронять планировщик,
// следующий запуск cron повторит попытку.
func (s *SchedulerService) job(ctx context.Context, delay time.Duration) {
	n, err := s.tickets.AutoCloseResolved(ctx, delay)
	if err != nil {
		logger.Error("failed to auto-close resolved tickets:", logger.ErrAttr(err))
		return
	}
	if n > 0 {
		logger.Info(fmt.Sprintf("Auto-closed %d resolved tickets", n))
	}
}
