package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTicketDTO_GetChanges_ManagerChanged(t *testing.T) {
	oldManagerID := uuid.New()
	newManagerID := uuid.New()

	old := &Ticket{
		Manager: &UserShort{ID: oldManagerID},
	}
	dto := &TicketDTO{
		ManagerID: &newManagerID,
		Provided:  map[string]bool{"managerId": true},
	}

	changes := dto.GetChanges(old)
	assert.Len(t, changes, 1)
	assert.Equal(t, ActionManagerChanged, changes[0].Tag)
	assert.Equal(t, oldManagerID.String(), changes[0].OldVal)
	assert.Equal(t, newManagerID.String(), changes[0].NewVal)
}

func TestTicketDTO_GetChanges_ManagerRemoved(t *testing.T) {
	oldManagerID := uuid.New()

	old := &Ticket{
		Manager: &UserShort{ID: oldManagerID},
	}
	dto := &TicketDTO{
		ManagerID: nil,
		Provided:  map[string]bool{"managerId": true},
	}

	changes := dto.GetChanges(old)
	assert.Len(t, changes, 1)
	assert.Equal(t, ActionManagerChanged, changes[0].Tag)
	assert.Equal(t, oldManagerID.String(), changes[0].OldVal)
	assert.Equal(t, "none", changes[0].NewVal)
}

func TestTicketDTO_GetChanges_ManagerAssigned(t *testing.T) {
	newManagerID := uuid.New()

	old := &Ticket{}
	dto := &TicketDTO{
		ManagerID: &newManagerID,
		Provided:  map[string]bool{"managerId": true},
	}

	changes := dto.GetChanges(old)
	assert.Len(t, changes, 1)
	assert.Equal(t, ActionManagerChanged, changes[0].Tag)
	assert.Equal(t, "none", changes[0].OldVal)
	assert.Equal(t, newManagerID.String(), changes[0].NewVal)
}

func TestTicketDTO_GetChanges_ManagerUnchanged(t *testing.T) {
	managerID := uuid.New()

	old := &Ticket{
		Manager: &UserShort{ID: managerID},
	}
	dto := &TicketDTO{
		ManagerID: &managerID,
		Provided:  map[string]bool{"managerId": true},
	}

	changes := dto.GetChanges(old)
	assert.Empty(t, changes)
}
