package models

import (
	"time"

	"github.com/google/uuid"
)

type UserCapabilities struct {
	ManagedGroupIDs []uuid.UUID `json:"managedGroupIds"`
	MemberGroupIDs  []uuid.UUID `json:"memberGroupIds"`
	IsRealmAdmin    bool        `json:"isRealmAdmin"`
}

// MembershipFilter задаёт фильтр списка пользователей realm по членству в группах.
type MembershipFilter string

const (
	// MembershipAll — все пользователи realm (без фильтра).
	MembershipAll MembershipFilter = "all"
	// MembershipCustomers — пользователи, не состоящие ни в одной группе («заказчики»).
	MembershipCustomers MembershipFilter = "customers"
	// MembershipExecutors — пользователи, состоящие хотя бы в одной группе («исполнители»).
	MembershipExecutors MembershipFilter = "executors"
)

type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	MattermostID *string    `json:"mattermostId" db:"mattermost_id"`
	Email        string     `json:"email" db:"email"`
	Name         string     `json:"name" db:"name"`
	SiteID       *uuid.UUID `json:"siteId" db:"site_id"`
	CreatedAt    time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time  `json:"updatedAt" db:"updated_at"`

	Permissions  map[string][]string         `json:"permissions"`
	Realms       []*UserRealm                `json:"realms,omitempty"`
	Capabilities map[string]*UserCapabilities `json:"capabilities"`

	AccessToken  string `json:"token"`
	RefreshToken string `json:"-"`
}

type UserDTO struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	MattermostID *string    `json:"mattermostId" db:"mattermost_id"`
	Username     string     `json:"username" db:"username" binding:"required"`
	Email        string     `json:"email" db:"email"`
	FirstName    string     `json:"firstName" db:"first_name"`
	LastName     string     `json:"lastName" db:"last_name"`
	SiteID       *uuid.UUID `json:"siteId" db:"site_id"`
}

type Actor struct {
	ID   uuid.UUID
	Name string
}

type UserShort struct {
	ID             uuid.UUID `json:"id"`
	Username       string    `json:"username" db:"username"`
	FirstName      string    `json:"firstName" db:"first_name"`
	LastName       string    `json:"lastName" db:"last_name"`
	Email          string    `json:"email" db:"email"`
	InternalNumber string    `json:"internalNumber,omitempty"`
}

type UserData struct {
	ID           uuid.UUID `json:"id" db:"id"`
	MattermostID *string   `json:"mattermostId" db:"mattermost_id"`
	Username     string    `json:"username" db:"username"`
	FirstName    string    `json:"firstName" db:"first_name"`
	LastName     string    `json:"lastName" db:"last_name"`
	Email        string    `json:"email" db:"email"`
	// RoleId       string  `json:"roleId" db:"role_id"`
	SiteID         *string   `json:"siteId" db:"site_id"`
	IsActive       bool      `json:"isActive" db:"is_active"`
	IsSystem       bool      `json:"isSystem" db:"is_system"`
	InternalNumber string    `json:"internalNumber" db:"internal_number"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`

	Realms []*UserRealm `json:"realms,omitempty"`
}

type UserDataDTO struct {
	ID             uuid.UUID `json:"id" db:"id"`
	MattermostID   *string   `json:"mattermostId" db:"mattermost_id"`
	Username       string     `json:"username" db:"username"`
	FirstName      string     `json:"firstName" db:"first_name"`
	LastName       string     `json:"lastName" db:"last_name"`
	Email          string     `json:"email" db:"email"`
	SiteID         *uuid.UUID `json:"siteId" db:"site_id"`
	IsActive       bool       `json:"isActive" db:"is_active"`
	IsSystem       bool       `json:"isSystem" db:"is_system"`
	InternalNumber string     `json:"internalNumber" db:"internal_number"`
	Actor          *Actor
	Realms         []*UserRealmDTO `json:"realms"`
}

type UpdateAccountDTO struct {
	ID             uuid.UUID `json:"id"`
	IsActive       bool      `json:"isActive"`
	MattermostID   *string   `json:"mattermostId"`
	InternalNumber string    `json:"internalNumber"`
	Actor          *Actor
	Realms         []*UserRealmDTO `json:"realms,omitempty"`
}

type UserRole struct {
	UserID   uuid.UUID
	RoleName string
	Realm    string
}

type UserRoleDTO struct {
	UserID  uuid.UUID `json:"userId" db:"user_id"`
	RoleID  uuid.UUID `json:"roleId" db:"role_id"`
	ActorID uuid.UUID `json:"actorId" db:"actor_id"`
}
