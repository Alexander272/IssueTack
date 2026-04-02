package services

import (
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/pkg/ws_hub"
)

type Services struct {
	Realms
	Groups
	Categories
	Sites
	Tickets
	ActivityLog
}

type Deps struct {
	Repo *repository.Repository
	Hub  *ws_hub.Hub
}

func NewServices(deps *Deps) *Services {
	transaction := NewTransactionManager(deps.Repo.Transaction)

	realms := NewRealmService(deps.Repo.Realms, transaction)

	groups := NewGroupService(deps.Repo.Groups)
	categories := NewCategoryService(deps.Repo.Categories)
	sites := NewSiteService(deps.Repo.Sites)
	logs := NewActivityLogService(deps.Repo.ActivityLog, transaction)
	tickets := NewTicketService(deps.Repo.Tickets, transaction, logs)

	return &Services{
		Realms:      realms,
		Groups:      groups,
		Categories:  categories,
		Sites:       sites,
		Tickets:     tickets,
		ActivityLog: logs,
	}
}
