package repository

import (
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Transaction interface {
	postgres.Transaction
}

type Realms interface {
	postgres.Realm
}
type Roles interface {
	postgres.Roles
}
type RoleHierarchy interface {
	postgres.RoleHierarchy
}
type Permissions interface {
	postgres.Permissions
}
type AuditLogs interface {
	postgres.AuditLogs
}
type Users interface {
	postgres.Users
}
type UserRealms interface {
	postgres.UserRealms
}

type Groups interface {
	postgres.Groups
}
type Categories interface {
	postgres.Categories
}
type Sites interface {
	postgres.Sites
}
type Tickets interface {
	postgres.Tickets
}
type Subtasks interface {
	postgres.Subtasks
}
type Attachments interface {
	postgres.Attachments
}
type Checklists interface {
	postgres.Checklists
}
type ActivityLog interface {
	postgres.ActivityLog
}
type Notifications interface {
	postgres.Notifications
}

type Repository struct {
	Transaction
	Realms
	Roles
	RoleHierarchy
	Permissions
	AuditLogs
	Users
	UserRealms
	Groups
	Categories
	Sites
	Tickets
	Subtasks
	Attachments
	Checklists
	ActivityLog
	Notifications
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	transaction := postgres.NewTransactionRepo(pool)

	return &Repository{
		Transaction:   transaction,
		Realms:        postgres.NewRealmRepo(pool, transaction),
		Roles:         postgres.NewRoleRepo(pool, transaction),
		RoleHierarchy: postgres.NewRoleHierarchyRepo(pool, transaction),
		Permissions:   postgres.NewPermissionRepo(pool, transaction),
		AuditLogs:     postgres.NewAuditRepo(pool, transaction),
		Users:         postgres.NewUserRepo(pool, transaction),
		UserRealms:    postgres.NewUserRealmRepo(pool, transaction),
		Groups:        postgres.NewGroupRepo(pool),
		Categories:    postgres.NewCategoryRepo(pool),
		Sites:         postgres.NewSiteRepo(pool),
		Tickets:       postgres.NewTicketRepo(pool, transaction),
		Subtasks:      postgres.NewSubtaskRepo(pool, transaction),
		Attachments:   postgres.NewAttachmentRepo(pool, transaction),
		Checklists:    postgres.NewChecklistRepo(pool, transaction),
		ActivityLog:   postgres.NewActivityRepo(pool, transaction),
		Notifications: postgres.NewNotificationRepo(pool, transaction),
	}
}
