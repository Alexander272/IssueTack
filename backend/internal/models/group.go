package models

import (
	"time"

	"github.com/google/uuid"
)

// type Group struct {
//     ID          uint     `json:"id"`
//     Name        string   `json:"name"`
//     Description string   `json:"description"`
//     Members     []User   `json:"members"`
//     CreatedAt   time.Time
// }

type Group struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
	Members     []*User   `json:"members,omitempty"`
}

type GroupShort struct {
	ID   uuid.UUID `json:"id" db:"id"`
	Name string    `json:"name" db:"name"`
}

type GetGroupDTO struct {
	ID uuid.UUID `json:"id" db:"id"`
}

type GetGroupsDTO struct{}

type GroupDTO struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
}

type DelGroupDTO struct {
	ID uuid.UUID `json:"id" db:"id"`
}

// GroupMember — вспомогательная структура для работы со связями в БД
type GroupMember struct {
	GroupID uuid.UUID `json:"groupId" db:"group_id"`
	UserID  uuid.UUID `json:"userId" db:"user_id"`
}

type GroupMemberDTO struct {
	GroupID uuid.UUID `json:"groupId" db:"group_id"`
	UserID  uuid.UUID `json:"userId" db:"user_id"`
}
