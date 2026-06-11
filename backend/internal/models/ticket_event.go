package models

import "github.com/google/uuid"

type TicketEvent struct {
	Event    string         `json:"event"`
	TicketID uuid.UUID      `json:"ticketId"`
	Ticket   *Ticket        `json:"ticket,omitempty"`
	Changes  []*FieldChange `json:"changes,omitempty"`
}
