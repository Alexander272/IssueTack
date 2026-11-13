package models

import (
	"time"

	"github.com/google/uuid"
)

// Статусы запроса на перенос срока. Это не TicketStatus: у запроса свой жизненный
// цикл (pending → approved | rejected), поэтому константы отдельные.
const (
	DueDateRequestStatusPending  = "pending"
	DueDateRequestStatusApproved = "approved"
	DueDateRequestStatusRejected = "rejected"
)

// DueDateRequest — запрос исполнителя на перенос срока заявки.
// Жизненный цикл: pending → approved | rejected (один pending на тикет — enforced DB partial unique index).
type DueDateRequest struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	TicketID        uuid.UUID  `json:"ticketId" db:"ticket_id"`
	RequesterID     uuid.UUID  `json:"requesterId" db:"requester_id"`
	Requester       *UserShort `json:"requester" db:"requester"`
	CurrentDueDate  *time.Time `json:"currentDueDate" db:"current_due_date"`
	ProposedDueDate time.Time  `json:"proposedDueDate" db:"proposed_due_date"`
	Reason          string     `json:"reason" db:"reason"`
	Status          string     `json:"status" db:"status"`
	DecidedBy       *uuid.UUID `json:"decidedBy" db:"decided_by"`
	DecidedAt       *time.Time `json:"decidedAt" db:"decided_at"`
	DecisionComment string     `json:"decisionComment" db:"decision_comment"`
	CreatedAt       time.Time  `json:"createdAt" db:"created_at"`
}

// CreateDueDateRequestDTO — запрос на перенос срока от исполнителя.
type CreateDueDateRequestDTO struct {
	ID              uuid.UUID `json:"id" db:"id"`
	TicketID        uuid.UUID `json:"ticketId" db:"ticketId"`
	ProposedDueDate time.Time `json:"proposedDueDate" db:"proposedDueDate"`
	Reason          string    `json:"reason" db:"reason"`
	Actor           *Actor    `json:"actor" db:"actor"`
	RealmID         string    `json:"realmId" db:"realmId"`
}

// GetDueDateRequestDTO — чтение активного (pending) запроса тикета. TicketID
// приходит из пути, Actor нужен для проверки доступа к самой заявке.
type GetDueDateRequestDTO struct {
	TicketID uuid.UUID `json:"ticketId" db:"ticketId"`
	Actor    *Actor    `json:"actor" db:"actor"`
}

// DecideDueDateRequestDTO — решение менеджера по запросу (утверждение/отклонение).
type DecideDueDateRequestDTO struct {
	ID              uuid.UUID `json:"id" db:"id"`
	TicketID        uuid.UUID `json:"ticketId" db:"ticketId"`
	RequestID       uuid.UUID `json:"requestId" db:"requestId"`
	Approved        bool      `json:"approved" db:"approved"`
	DecisionComment string    `json:"decisionComment" db:"decisionComment"`
	Actor           *Actor    `json:"actor" db:"actor"`
	RealmID         string    `json:"realmId" db:"realmId"`
}
