package models

import (
	"time"

	"github.com/google/uuid"
)

type Site struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Address   string    `json:"address" db:"address"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type SiteShort struct {
	ID   uuid.UUID `json:"id" db:"id"`
	Name string    `json:"name" db:"name"`
}

type GetSiteByIdDTO struct {
	ID uuid.UUID `json:"id" db:"id"`
}

type GetSitesDTO struct{}

type SiteDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	Name    string    `json:"name" db:"name"`
	Address string    `json:"address" db:"address"`
}

type DelSiteDTO struct {
	ID uuid.UUID `json:"id" db:"id"`
}
