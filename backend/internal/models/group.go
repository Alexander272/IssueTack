package models

import (
	"time"

	"github.com/google/uuid"
)

type Group struct {
	ID          uuid.UUID    `json:"id" db:"id"`
	RealmID     *uuid.UUID   `json:"realmId" db:"realm_id"`
	Name        string       `json:"name" db:"name"`
	Description string       `json:"description" db:"description"`
	CreatedAt   time.Time    `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time    `json:"updatedAt" db:"updated_at"`
	Members     []*UserShort `json:"members,omitempty"`

	DefaultAssigneeID *uuid.UUID `json:"defaultAssigneeId" db:"default_assignee_id"`
	ManagerID         *uuid.UUID `json:"managerId" db:"manager_id"`
	DefaultAssignee   *UserShort `json:"defaultAssignee,omitempty"`
	Manager           *UserShort `json:"manager,omitempty"`
}

type GroupShort struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	Name             string     `json:"name" db:"name"`
	DefaultAssigneeID *uuid.UUID `json:"defaultAssigneeId,omitempty" db:"default_assignee_id"`
}

// GetGroupDTO — запрос одной группы. RealmID заполняется сервером из
// авторизованного realm (constants.CtxRealm) и не принимается от клиента: группа
// ищется по id, поэтому без предиката realm запрос «мой realm + чужой uuid»
// проходил бы к чужой группе. json:"-" исключает подстановку realm из тела.
type GetGroupDTO struct {
	ID      uuid.UUID  `json:"id" db:"id"`
	RealmID *uuid.UUID `json:"-" db:"-"`
}

// GetGroupsDTO — запрос списка групп. RealmID заполняется сервером (см. GetGroupDTO).
type GetGroupsDTO struct {
	RealmID *uuid.UUID `json:"-" db:"-"`
}

type GroupDTO struct {
	ID                uuid.UUID   `json:"id" db:"id"`
	RealmID           uuid.UUID   `json:"realmId" db:"realm_id"`
	Name              string      `json:"name" db:"name"`
	Description       string      `json:"description" db:"description"`
	DefaultAssigneeID *uuid.UUID  `json:"defaultAssigneeId" db:"default_assignee_id"`
	ManagerID         *uuid.UUID  `json:"managerId" db:"manager_id"`
	MemberIDs         []uuid.UUID `json:"memberIds"`
}

type GroupManagerReq struct {
	GroupID           uuid.UUID  `json:"groupId"`
	ManagerID         *uuid.UUID `json:"managerId"`
	DefaultAssigneeID *uuid.UUID `json:"defaultAssigneeId"`
}

type DelGroupDTO struct {
	ID      uuid.UUID  `json:"id" db:"id"`
	RealmID *uuid.UUID `json:"-" db:"-"`
}

// GroupMember — вспомогательная структура для работы со связями в БД
type GroupMember struct {
	GroupID uuid.UUID `json:"groupId" db:"group_id"`
	UserID  uuid.UUID `json:"userId" db:"user_id"`
}

// GroupMemberDTO — добавление/удаление участника группы. GroupID приходит от
// клиента, поэтому RealmID (подставленный сервером) обязателен: без него
// участника можно было бы добавить в группу чужого реалма.
type GroupMemberDTO struct {
	GroupID uuid.UUID  `json:"groupId" db:"group_id"`
	UserID  uuid.UUID  `json:"userId" db:"user_id"`
	RealmID *uuid.UUID `json:"-" db:"-"`
}
