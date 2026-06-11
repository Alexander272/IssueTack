package models

import (
	"time"

	"github.com/google/uuid"
)

type Realm struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Code        string    `json:"code" db:"code"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	IsActive    bool      `json:"isActive" db:"is_active"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
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

type UserRealm struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"userId" db:"user_id"`
	RealmID   uuid.UUID `json:"realmId" db:"realm_id"`
	RoleID    uuid.UUID `json:"roleId" db:"role_id"`
	Role      *Role     `json:"role,omitempty"`
	Realm     *Realm    `json:"realm,omitempty"`
	IsActive  bool      `json:"isActive" db:"is_active"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

type UserRealmDTO struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"userId" db:"user_id" binding:"required"`
	RealmID   uuid.UUID  `json:"realmId" db:"realm_id" binding:"required"`
	RoleID    *uuid.UUID `json:"roleId" db:"role_id" binding:"required"`
	IsActive  bool       `json:"isActive" db:"is_active"`
	CreatedAt string     `json:"createdAt" db:"created_at"`
}

type UserRealmsDTO struct {
	UserID uuid.UUID `json:"userId" db:"user_id" binding:"required"`
}
