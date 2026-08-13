package services

import (
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/stretchr/testify/assert"
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
