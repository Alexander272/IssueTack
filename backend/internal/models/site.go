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

// GetSitesDTO — заготовка под будущие фильтры списка площадок; пока пустой,
// поэтому в вызовы репозитория/сервиса не передаётся.
type GetSitesDTO struct{}

type SiteDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	Name    string    `json:"name" db:"name"`
	Address string    `json:"address" db:"address"`
}

type DelSiteDTO struct {
	ID uuid.UUID `json:"id" db:"id"`
}

// SiteLocation — расположение внутри площадки (комната, стеллаж, склад).
type SiteLocation struct {
	ID        uuid.UUID `json:"id" db:"id"`
	SiteID    uuid.UUID `json:"siteId" db:"site_id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type GetSiteLocationsDTO struct {
	SiteID uuid.UUID `json:"siteId"`
}

type GetSiteLocationDTO struct {
	ID     uuid.UUID `json:"id"`
	SiteID uuid.UUID `json:"siteId"`
}

// SiteLocationDTO — создание/правка расположения в карточке площадки.
type SiteLocationDTO struct {
	ID     uuid.UUID `json:"id"`
	SiteID uuid.UUID `json:"siteId"`
	Name   string    `json:"name" binding:"required"`
}

type DelSiteLocationDTO struct {
	ID     uuid.UUID `json:"id"`
	SiteID uuid.UUID `json:"siteId"`
}
