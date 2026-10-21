package services

import (
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSchedulerService_Start_Disabled(t *testing.T) {
	_, _, _, _, _, _, _, ticketSvc := ticketServiceFixtures()
	svc := NewSchedulerService(&SchedulerDeps{Tickets: ticketSvc})

	err := svc.Start(&config.TicketConfig{ResolvedToClosedAfter: 0})
	assert.NoError(t, err)
	assert.Len(t, svc.cron.Jobs(), 0)
}

func TestSchedulerService_Start_Enabled(t *testing.T) {
	_, _, _, _, _, _, _, ticketSvc := ticketServiceFixtures()
	svc := NewSchedulerService(&SchedulerDeps{Tickets: ticketSvc})

	err := svc.Start(&config.TicketConfig{
		ResolvedToClosedAfter: 168 * time.Hour,
		AutoCloseSchedule:     "@every 1h",
	})
	assert.NoError(t, err)
	assert.Len(t, svc.cron.Jobs(), 1)
	assert.NoError(t, svc.Stop())
}

func TestSchedulerService_Start_InvalidSchedule(t *testing.T) {
	_, _, _, _, _, _, _, ticketSvc := ticketServiceFixtures()
	svc := NewSchedulerService(&SchedulerDeps{Tickets: ticketSvc})

	err := svc.Start(&config.TicketConfig{
		ResolvedToClosedAfter: 168 * time.Hour,
		AutoCloseSchedule:     "not-a-cron",
	})
	assert.Error(t, err)
	assert.NoError(t, svc.Stop())
}

// TestSchedulerService_Start_PriorityReminders проверяет, что расписания напоминаний
// по приоритету регистрируются отдельными задачами и что пустое расписание отключает задачу.
func TestSchedulerService_Start_PriorityReminders(t *testing.T) {
	_, _, _, _, _, _, _, ticketSvc := ticketServiceFixtures()
	svc := NewSchedulerService(&SchedulerDeps{Tickets: ticketSvc})

	err := svc.Start(&config.TicketConfig{
		NotifyHighPriorityInterval:   3 * time.Hour,
		NotifyUrgentPriorityInterval: time.Hour,
	})
	assert.NoError(t, err)
	assert.Len(t, svc.cron.Jobs(), 2)
	assert.NoError(t, svc.Stop())

	svc = NewSchedulerService(&SchedulerDeps{Tickets: ticketSvc})
	err = svc.Start(&config.TicketConfig{
		NotifyHighPriorityInterval:   0,
		NotifyUrgentPriorityInterval: 0,
	})
	assert.NoError(t, err)
	assert.Len(t, svc.cron.Jobs(), 0)
	assert.NoError(t, svc.Stop())
}

// TestSchedulerService_NotifyUrgentPriorityJob проверяет, что задача берёт активные
// заявки с критичным приоритетом и отправляет по каждой напоминание через сервис.
func TestSchedulerService_NotifyUrgentPriorityJob(t *testing.T) {
	mockTickets := new(MockTicketsService)
	mockNotifications := new(MockNotificationService)
	svc := NewSchedulerService(&SchedulerDeps{Tickets: mockTickets, Notifications: mockNotifications})

	ticketID := uuid.New()
	ticket := &models.Ticket{ID: ticketID, Priority: models.PriorityUrgent}

	mockNotifications.On("GetActiveByPriorityGlobal", mock.Anything, models.PriorityUrgent).Return([]uuid.UUID{ticketID}, nil).Once()
	mockTickets.On("GetSummary", mock.Anything, ticketID).Return(ticket, nil).Once()
	mockNotifications.On("NotifyReminderUrgent", mock.Anything, ticket, mock.AnythingOfType("time.Duration")).Return(nil).Once()

	svc.notifyUrgentPriorityJob(t.Context(), time.Hour)

	mockNotifications.AssertExpectations(t)
	mockTickets.AssertExpectations(t)
}

// TestSchedulerService_Start_WorkingHours проверяет, что окно рабочего времени
// разбирается в Start и не меняет число зарегистрированных задач.
func TestSchedulerService_Start_WorkingHours(t *testing.T) {
	_, _, _, _, _, _, _, ticketSvc := ticketServiceFixtures()
	svc := NewSchedulerService(&SchedulerDeps{Tickets: ticketSvc})

	err := svc.Start(&config.TicketConfig{
		NotifyUrgentPriorityInterval: time.Hour,
		WorkingHours: config.WorkingHoursConfig{
			Start: "08:00", End: "18:00", Timezone: "UTC", Days: []string{"mon"},
		},
	})
	assert.NoError(t, err)
	assert.True(t, svc.workingHours.enabled)
	assert.Equal(t, 8*time.Hour, svc.workingHours.start)
	assert.Equal(t, 18*time.Hour, svc.workingHours.end)
	assert.Len(t, svc.cron.Jobs(), 1)
	assert.NoError(t, svc.Stop())
}

// TestSchedulerService_Jobs_SkippedOutsideWorkingHours — вне рабочего окна джобы
// выходят до выборки: сервис уведомлений не вызывается, значит дедуп-слот
// (reminder_last_sent) не занимается и напоминание не сдвигается на пол интервала.
// Окно берётся «на завтрашний день» — сегодняшний weekday гарантированно вне
// списка, результат не зависит от часа запуска теста.
func TestSchedulerService_Jobs_SkippedOutsideWorkingHours(t *testing.T) {
	tomorrow := time.Weekday((int(time.Now().Weekday()) + 1) % 7)
	window := newWorkingHours(config.WorkingHoursConfig{
		Start: "08:00", End: "18:00", Days: []string{weekdayNames[tomorrow]},
	})
	assert.False(t, window.allows(time.Now()), "окно должно быть вне сегодняшнего дня")

	t.Run("priority reminder", func(t *testing.T) {
		mockNotifications := new(MockNotificationService)
		svc := NewSchedulerService(&SchedulerDeps{Tickets: new(MockTicketsService), Notifications: mockNotifications})
		svc.workingHours = window

		svc.notifyUrgentPriorityJob(t.Context(), time.Hour)
		svc.notifyHighPriorityJob(t.Context(), 3*time.Hour)

		mockNotifications.AssertNotCalled(t, "GetActiveByPriorityGlobal", mock.Anything, mock.Anything)
	})

	t.Run("overdue", func(t *testing.T) {
		mockNotifications := new(MockNotificationService)
		svc := NewSchedulerService(&SchedulerDeps{Tickets: new(MockTicketsService), Notifications: mockNotifications})
		svc.workingHours = window

		svc.notifyOverdueJob(t.Context())

		mockNotifications.AssertNotCalled(t, "GetOverdueTicketIDs", mock.Anything, mock.Anything)
	})
}
