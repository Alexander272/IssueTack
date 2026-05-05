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
type ActivityLog interface {
	postgres.ActivityLog
}

type Repository struct {
	Transaction
	Realms
	Roles
	RoleHierarchy
	Permissions
	AuditLogs
	Users
	Groups
	Categories
	Sites
	Tickets
	ActivityLog
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
		Groups:        postgres.NewGroupRepo(pool),
		Categories:    postgres.NewCategoryRepo(pool),
		Sites:         postgres.NewSiteRepo(pool),
		Tickets:       postgres.NewTicketRepo(pool, transaction),
		ActivityLog:   postgres.NewActivityRepo(pool, transaction),
	}
}
