package postgres

import (
	"fmt"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

var (
	activeStatuses  = []models.TicketStatus{"open", "in_progress", "pending", "on_hold", "resolved"}
	archiveStatuses = []models.TicketStatus{"closed", "cancelled"}
)

// whereBuilder аккумулирует WHERE-предложения с автонумерацией позиционных
// аргументов, избавляя от ручного ведения argIdx/args в построителе запроса.
type whereBuilder struct {
	clauses []string
	args    []any
	idx     int
}

func (w *whereBuilder) add(expr string) {
	w.clauses = append(w.clauses, expr)
}

// place нумерует n плейсхолдеров (продвигая idx) и возвращает их через запятую.
func (w *whereBuilder) place(n int) string {
	parts := make([]string, n)
	for i := range parts {
		w.idx++
		parts[i] = fmt.Sprintf("$%d", w.idx)
	}
	return strings.Join(parts, ",")
}

func (w *whereBuilder) sites(ids []string) {
	inList(w, "t.site_id", ids)
}

func (w *whereBuilder) statuses(st *models.TicketStatus, many []models.TicketStatus) {
	if st != nil {
		w.idx++
		w.add(fmt.Sprintf("t.status = $%d", w.idx))
		w.args = append(w.args, *st)
	} else if len(many) > 0 {
		w.args = append(w.args, toAny(many)...)
		w.add("t.status IN (" + w.place(len(many)) + ")")
	}
}

// statusMode ограничивает список статусов вкладкой: «Активные» — activeStatuses,
// «Архив» — archiveStatuses. withResolved (страница «Мои задачи», mode=assigned)
// переносит resolved в архив: активная вкладка его не показывает, архивная — включает.
// Наборы собираются в новый слайс — append к пакетным мутировал бы их.
func (w *whereBuilder) statusMode(archived *bool, withResolved bool) {
	isArchive := archived != nil && *archived
	list := make([]models.TicketStatus, 0, len(activeStatuses)+1)
	if isArchive {
		if withResolved {
			list = append(list, models.StatusResolved)
		}
		list = append(list, archiveStatuses...)
	} else {
		for _, st := range activeStatuses {
			if withResolved && st == models.StatusResolved {
				continue
			}
			list = append(list, st)
		}
	}
	w.args = append(w.args, toAny(list)...)
	w.add("t.status IN (" + w.place(len(list)) + ")")
}

// eq добавляет "column = $n" если val не nil (типизированный nil-указатель корректно отсекается).
func eq[T any](w *whereBuilder, column string, val *T) {
	if val == nil {
		return
	}
	w.idx++
	w.add(fmt.Sprintf("%s = $%d", column, w.idx))
	w.args = append(w.args, *val)
}

func (w *whereBuilder) groups(ids []uuid.UUID, ungrouped *uuid.UUID) {
	if len(ids) == 0 && ungrouped == nil {
		return
	}
	var clauses []string
	if len(ids) > 0 {
		w.args = append(w.args, toAny(ids)...)
		clauses = append(clauses, "t.group_id IN ("+w.place(len(ids))+")")
	}
	if ungrouped != nil {
		w.idx++
		clauses = append(clauses, fmt.Sprintf("(t.group_id IS NULL AND t.assignee_id = $%d)", w.idx))
		w.args = append(w.args, *ungrouped)
	}
	w.add("(" + strings.Join(clauses, " OR ") + ")")
}

func (w *whereBuilder) search(s *string) {
	if s == nil || *s == "" {
		return
	}
	w.idx++
	w.add(fmt.Sprintf("(LOWER(t.title) LIKE $%d OR t.ticket_number::text LIKE $%d)", w.idx, w.idx+1))
	pattern := "%" + strings.ToLower(*s) + "%"
	w.args = append(w.args, pattern, pattern)
	w.idx++
}

func (w *whereBuilder) dueDate(from, to *time.Time) {
	if from != nil {
		w.idx++
		w.add(fmt.Sprintf("t.due_date >= $%d", w.idx))
		w.args = append(w.args, *from)
	}
	if to != nil {
		w.idx++
		w.add(fmt.Sprintf("t.due_date <= $%d", w.idx))
		w.args = append(w.args, *to)
	}
}

func (w *whereBuilder) priorities(ps []models.Priority) {
	inList(w, "t.priority", ps)
}

func (w *whereBuilder) favorites(userID *uuid.UUID, typ *models.FavoriteType) {
	if userID == nil || typ == nil {
		return
	}
	w.idx++
	w.add(fmt.Sprintf("EXISTS (SELECT 1 FROM %s fav WHERE fav.ticket_id = t.id AND fav.user_id = $%d AND fav.type = $%d)",
		Tables.TicketFavorites, w.idx, w.idx+1))
	w.args = append(w.args, *userID, *typ)
	w.idx++
}

// creatorOrOwner добавляет "(t.creator_id = $n OR t.owner_id = $n)": тикеты, где
// пользователь является автором ИЛИ заказчиком (для Mattermost-плагина).
func (w *whereBuilder) creatorOrOwner(userID *uuid.UUID) {
	if userID == nil {
		return
	}
	w.idx++
	w.add(fmt.Sprintf("(t.creator_id = $%d OR t.owner_id = $%d)", w.idx, w.idx+1))
	w.args = append(w.args, *userID, *userID)
	w.idx++
}

// myWork добавляет "(t.assignee_id = $n OR t.group_id IN (...))" для страницы
// «Мои задачи»: личные назначения пользователя ИЛИ задачи его групп.
func (w *whereBuilder) myWork(f *models.MyWorkFilter) {
	if f == nil {
		return
	}
	disj := make([]string, 0, 2)
	w.idx++
	disj = append(disj, fmt.Sprintf("t.assignee_id = $%d", w.idx))
	w.args = append(w.args, f.UserID)
	if len(f.GroupIDs) > 0 {
		disj = append(disj, "t.group_id IN ("+w.place(len(f.GroupIDs))+")")
		w.args = append(w.args, toAny(f.GroupIDs)...)
	}
	w.add("(" + strings.Join(disj, " OR ") + ")")
}

// inList генерит "column IN ($1,$2,..)" и добавляет значения в аргументы.
func inList[T any](w *whereBuilder, column string, values []T) {
	if len(values) == 0 {
		return
	}
	w.args = append(w.args, toAny(values)...)
	w.add(column + " IN (" + w.place(len(values)) + ")")
}

// toAny конвертирует типизированный срез в срез пустых интерфейсов.
func toAny[T any](vals []T) []any {
	out := make([]any, len(vals))
	for i := range vals {
		out[i] = vals[i]
	}
	return out
}
