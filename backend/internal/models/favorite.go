package models

import (
	"time"

	"github.com/google/uuid"
)

// FavoriteType — тип избранной заявки: постоянное избранное (звезда) или временное закрепление.
type FavoriteType string

const (
	FavoriteTypePermanent FavoriteType = "permanent"
	FavoriteTypeTemporary FavoriteType = "temporary"
)

// IsValid проверяет, что тип — допустимое значение FavoriteType.
func (t FavoriteType) IsValid() bool {
	switch t {
	case FavoriteTypePermanent, FavoriteTypeTemporary:
		return true
	default:
		return false
	}
}

// TicketFavorite — отметка «в избранном» конкретного пользователя по конкретной заявке.
type TicketFavorite struct {
	ID        uuid.UUID    `json:"id" db:"id"`
	UserID    uuid.UUID    `json:"userId" db:"user_id"`
	TicketID  uuid.UUID    `json:"ticketId" db:"ticket_id"`
	Type      FavoriteType `json:"type" db:"type"`
	CreatedAt time.Time    `json:"createdAt" db:"created_at"`
}

// FavoriteDTO — запрос на добавление/удаление избранного.
type FavoriteDTO struct {
	ActorID  uuid.UUID    `json:"-"`
	UserID   uuid.UUID    `json:"-"`
	TicketID uuid.UUID    `json:"ticketId"`
	Type     FavoriteType `json:"type" binding:"required,enum"`
}

// IsFavoriteDTO — запрос на проверку наличия избранного.
type IsFavoriteDTO struct {
	TicketID uuid.UUID `json:"-"`
	UserID   uuid.UUID `json:"-"`
}
