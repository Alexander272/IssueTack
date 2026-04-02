package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	MattermostID *string    `json:"mattermostId" db:"mattermost_id"`
	Email        string     `json:"email" db:"email"`
	FullName     string     `json:"fullName" db:"full_name"`
	Role         string     `json:"role" db:"role"`
	SiteID       *uuid.UUID `json:"siteId" db:"site_id"`
	CreatedAt    time.Time  `json:"createdAt" db:"created_at"`

	Permissions  []string `json:"permissions"`
	AccessToken  string   `json:"token"`
	RefreshToken string   `json:"-"`
}

type UserShort struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"fullName"`
}

type UserData struct {
	ID           uuid.UUID `json:"id" db:"id"`
	SSO_ID       string    `json:"ssoId" db:"sso_id"`
	MattermostID *string   `json:"mattermostId" db:"mattermost_id"`
	Username     string    `json:"username" db:"username"`
	FirstName    string    `json:"firstName" db:"first_name"`
	LastName     string    `json:"lastName" db:"last_name"`
	Email        string    `json:"email" db:"email"`
	// RoleId       string  `json:"roleId" db:"role_id"`
	SiteID *string `json:"siteId" db:"site_id"`
}

type UserRole struct {
	UserID   uuid.UUID
	RoleName string
	Realm    string
}

type UserRoleDTO struct {
	UserID  uuid.UUID `json:"userId" db:"user_id"`
	RoleID  uuid.UUID `json:"roleId" db:"role_id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}
