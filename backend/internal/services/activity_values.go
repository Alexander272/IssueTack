package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

// logPersonName собирает «Фамилия Имя (username)» для журнала активности — тот же
// формат, что в веб-форме выбора заказчика (см. customerLabel), но без пустых скобок.
// При неполных данных — то, что есть; при полностью пустых (запись удалена) — сырой id.
func logPersonName(firstName, lastName, username string, id uuid.UUID) string {
	name := strings.TrimSpace(lastName + " " + firstName)
	switch {
	case name == "" && username == "":
		return id.String()
	case name == "":
		return username
	case username == "" || strings.EqualFold(username, name):
		return name
	default:
		return fmt.Sprintf("%s (%s)", name, username)
	}
}

// logEntityName — имя справочной сущности (группа/площадка/категория) для журнала;
// при пустом имени — сырой id.
func logEntityName(name string, id uuid.UUID) string {
	if name != "" {
		return name
	}
	return id.String()
}

// userOrNoneLabel — имя пользователя тикета/подзадачи для журнала; nil — "none".
func userOrNoneLabel(u *models.UserShort) string {
	if u == nil {
		return "none"
	}
	return logPersonName(u.FirstName, u.LastName, u.Username, u.ID)
}

// resolveUserLabel резолвит ФИО пользователя из сырого значения журнала (id или
// "none"). Отсутствие резолвера, ошибка загрузки и не-идентификатор не проваливают
// операцию — возвращается исходное значение.
func resolveUserLabel(ctx context.Context, users Users, raw string) string {
	id, err := uuid.Parse(raw)
	if err != nil {
		return raw // "none" или не-идентификатор
	}
	if users == nil {
		return raw
	}
	u, err := users.GetByID(ctx, id)
	if err != nil || u == nil {
		return raw
	}
	return logPersonName(u.FirstName, u.LastName, u.Username, u.ID)
}

// logValues строит old/new-мапы снапшота журнала из изменений тикета: id
// пользователей/групп/площадок/категорий заменяются именами. Старые значения
// берутся из загруженного тикета, новые — резолвятся по id из change. Ошибки
// резолва не проваливают операцию: в лог уходит исходный id. Уведомления и
// Mattermost продолжают получать исходный []*FieldChange с id — хьюманизация
// касается только снапшота, сохраняемого в БД (там «Вам назначена задача» парсит
// NewVal как uuid и не должен сломаться).
func (s *TicketService) logValues(ctx context.Context, old *models.Ticket, changes []*models.FieldChange) (map[string]string, map[string]string) {
	oldMap := make(map[string]string, len(changes))
	newMap := make(map[string]string, len(changes))
	for _, c := range changes {
		var oldVal, newVal string
		switch c.Tag {
		case models.ActionAssignChanged, models.ActionAssigned,
			models.ActionManagerChanged, models.ActionOwnerChanged:
			oldVal = logTicketUser(old, c.Tag)
			newVal = resolveUserLabel(ctx, s.users, c.NewVal)
		case models.ActionGroupChanged, models.ActionGroupAssigned:
			if old.Group == nil {
				oldVal = "none"
			} else {
				oldVal = logEntityName(old.Group.Name, old.Group.ID)
			}
			newVal = s.resolveGroupLabel(ctx, old, c.NewVal)
		case models.ActionSiteChanged:
			if old.Site == nil {
				oldVal = "none"
			} else {
				oldVal = logEntityName(old.Site.Name, old.Site.ID)
			}
			newVal = s.resolveSiteLabel(ctx, c.NewVal)
		case models.ActionCategoryChanged:
			if old.Category == nil {
				oldVal = "none"
			} else {
				oldVal = logEntityName(old.Category.Name, old.Category.ID)
			}
			newVal = s.resolveCategoryLabel(ctx, old, c.NewVal)
		default:
			oldVal = c.OldVal
			newVal = c.NewVal
		}
		oldMap[string(c.Tag)] = oldVal
		newMap[string(c.Tag)] = newVal
	}
	return oldMap, newMap
}

// logTicketUser — имя пользователя тикета по тегу изменения (старая сторона).
func logTicketUser(old *models.Ticket, tag models.ActivityType) string {
	switch tag {
	case models.ActionManagerChanged:
		return userOrNoneLabel(old.Manager)
	case models.ActionOwnerChanged:
		return userOrNoneLabel(old.Owner)
	default: // ActionAssignChanged, ActionAssigned
		return userOrNoneLabel(old.Assignee)
	}
}

func (s *TicketService) resolveGroupLabel(ctx context.Context, old *models.Ticket, raw string) string {
	id, err := uuid.Parse(raw)
	if err != nil {
		return raw
	}
	if s.groups == nil {
		return raw
	}
	g, err := s.groups.GetByID(ctx, &models.GetGroupDTO{ID: id, RealmID: old.RealmID})
	if err != nil || g == nil {
		return raw
	}
	return logEntityName(g.Name, g.ID)
}

func (s *TicketService) resolveSiteLabel(ctx context.Context, raw string) string {
	id, err := uuid.Parse(raw)
	if err != nil {
		return raw
	}
	if s.sites == nil {
		return raw
	}
	site, err := s.sites.GetByID(ctx, &models.GetSiteByIdDTO{ID: id})
	if err != nil || site == nil {
		return raw
	}
	return logEntityName(site.Name, site.ID)
}

func (s *TicketService) resolveCategoryLabel(ctx context.Context, old *models.Ticket, raw string) string {
	id, err := uuid.Parse(raw)
	if err != nil {
		return raw
	}
	if s.categories == nil || old.RealmID == nil {
		return raw
	}
	c, err := s.categories.GetByID(ctx, &models.GetCategoryByIdDTO{ID: id, RealmID: *old.RealmID})
	if err != nil || c == nil {
		return raw
	}
	return logEntityName(c.Name, c.ID)
}

// logUserLabel — ФИО пользователя для точечных лог-мест (снапшот Delete): по id,
// при сбое резолва — сырой id.
func (s *TicketService) logUserLabel(ctx context.Context, id uuid.UUID) string {
	return resolveUserLabel(ctx, s.users, id.String())
}

// subtaskLogValues строит old/new-мапы снапшота журнала подзадачи: единственное
// поле с id — исполнитель (ActionAssigned), его значения заменяются ФИО; прочие
// (заголовок, статус, приоритет, срок) не трогаются.
func (s *SubtaskService) subtaskLogValues(ctx context.Context, old *models.Subtask, changes []*models.FieldChange) (map[string]string, map[string]string) {
	oldMap := make(map[string]string, len(changes))
	newMap := make(map[string]string, len(changes))
	for _, c := range changes {
		oldMap[string(c.Tag)] = c.OldVal
		newMap[string(c.Tag)] = c.NewVal
		if c.Tag == models.ActionAssigned {
			oldMap[string(c.Tag)] = userOrNoneLabel(old.Assignee)
			newMap[string(c.Tag)] = resolveUserLabel(ctx, s.users, c.NewVal)
		}
	}
	return oldMap, newMap
}
