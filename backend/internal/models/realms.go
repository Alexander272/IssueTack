package models

import (
	"time"

	"github.com/google/uuid"
)

type Realm struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Code      string    `json:"code" db:"code"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type GetRealmDTO struct {
}

type GetRealmByIdDTO struct {
	ID uuid.UUID `json:"id" db:"id"`
}

type RealmDTO struct {
	ID   uuid.UUID `json:"id" db:"id"`
	Code string    `json:"code" db:"code"`
	Name string    `json:"name" db:"name"`
}

type DeleteRealmDTO struct {
	ID uuid.UUID `json:"id" db:"id"`
}
