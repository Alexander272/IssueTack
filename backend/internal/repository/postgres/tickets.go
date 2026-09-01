package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var sortMapping = map[string]string{
	"dueDate":      "t.due_date",
	"closedAt":     "t.closed_at",
	"priority":     "CASE t.priority WHEN 'urgent' THEN 4 WHEN 'high' THEN 3 WHEN 'medium' THEN 2 WHEN 'low' THEN 1 END",
	"status":       "t.status",
	"ticketNumber": "t.ticket_number",
	"title":        "t.title",
	"owner":        "u_owner.last_name",
	"category":     "c.name",
	"site":         "s.name",
	"assignee":     "COALESCE(u_assignee.last_name, g.name)",
	"createdAt":    "t.created_at",
}

type TicketRepo struct {
	db *pgxpool.Pool
	Transaction
}

func NewTicketRepo(db *pgxpool.Pool, tr Transaction) *TicketRepo {
	return &TicketRepo{
		db:          db,
		Transaction: tr,
	}
}

type Tickets interface {
	Get(ctx context.Context, req *models.TicketFilter) ([]*models.Ticket, int, error)
	GetByID(ctx context.Context, req *models.GetTicketByIdDTO) (*models.Ticket, error)
	Create(ctx context.Context, tx Tx, dto *models.TicketDTO) error
	Update(ctx context.Context, tx Tx, dto *models.TicketDTO) error
	Delete(ctx context.Context, tx Tx, dto *models.DeleteTicketDTO) error
	CloseResolved(ctx context.Context, cutoff time.Time) (int64, error)
	CountNotClosedByGroup(ctx context.Context, groupID uuid.UUID) (int, error)
	CountNotClosedByCategory(ctx context.Context, categoryID uuid.UUID) (int, error)
}

type nullableTicketAssoc struct {
	OwnerID, AssigneeID, ManagerID                                                *uuid.UUID
	OwnerUsername, OwnerFirstName, OwnerLastName, OwnerInternalNumber             *string
	AssigneeUsername, AssigneeFirstName, AssigneeLastName, AssigneeInternalNumber *string
	ManagerUsername, ManagerFirstName, ManagerLastName, ManagerInternalNumber     *string
	GroupID                                                                       *uuid.UUID
	GroupName                                                                     *string
}

func (a *nullableTicketAssoc) assign(ticket *models.Ticket) {
	if a.OwnerID != nil {
		ticket.Owner = &models.UserShort{
			ID: *a.OwnerID, Username: *a.OwnerUsername,
			FirstName: *a.OwnerFirstName, LastName: *a.OwnerLastName,
			InternalNumber: *a.OwnerInternalNumber,
		}
	}
	if a.AssigneeID != nil {
		ticket.Assignee = &models.UserShort{
			ID: *a.AssigneeID, Username: *a.AssigneeUsername,
			FirstName: *a.AssigneeFirstName, LastName: *a.AssigneeLastName,
			InternalNumber: *a.AssigneeInternalNumber,
		}
	}
	if a.ManagerID != nil {
		ticket.Manager = &models.UserShort{
			ID: *a.ManagerID, Username: *a.ManagerUsername,
			FirstName: *a.ManagerFirstName, LastName: *a.ManagerLastName,
			InternalNumber: *a.ManagerInternalNumber,
		}
	}
	if a.GroupID != nil {
		ticket.Group = &models.GroupShort{ID: *a.GroupID, Name: *a.GroupName}
	}
}

var (
	activeStatuses  = []models.TicketStatus{"open", "in_progress", "pending", "on_hold", "resolved"}
	archiveStatuses = []models.TicketStatus{"closed", "cancelled"}
)

// whereBuilder аккумулирует WHERE-предложения с автонумерацией позиционных
// аргументов, избавляя от ручного ведения argIdx/args в построителе запроса.
type whereBuilder struct {
	clauses []string
	args    []interface{}
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

func (w *whereBuilder) statusMode(archived *bool) {
	list := activeStatuses
	if archived != nil && *archived {
		list = archiveStatuses
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

// inList генерит "column IN ($1,$2,..)" и добавляет значения в аргументы.
func inList[T any](w *whereBuilder, column string, values []T) {
	if len(values) == 0 {
		return
	}
	w.args = append(w.args, toAny(values)...)
	w.add(column + " IN (" + w.place(len(values)) + ")")
}

// toAny конвертирует типизированный срез в срез пустых интерфейсов.
func toAny[T any](vals []T) []interface{} {
	out := make([]interface{}, len(vals))
	for i := range vals {
		out[i] = vals[i]
	}
	return out
}

func (r *TicketRepo) Get(ctx context.Context, req *models.TicketFilter) ([]*models.Ticket, int, error) {
	base := fmt.Sprintf(`SELECT 
			t.id, t.title, t.description, t.status, t.priority, t.ticket_number, t.realm_id, t.due_date, t.closed_at, t.resolved_at, t.created_at, t.updated_at,
			u_creator.id, u_creator.username AS creator_username, u_creator.first_name AS creator_first_name, u_creator.last_name AS creator_last_name, u_creator.internal_number AS creator_internal_number,
			u_owner.id, u_owner.username AS owner_username, u_owner.first_name AS owner_first_name, u_owner.last_name AS owner_last_name, u_owner.internal_number AS owner_internal_number,
			u_assignee.id, u_assignee.username AS assignee_username, u_assignee.first_name AS assignee_first_name, u_assignee.last_name AS assignee_last_name, u_assignee.internal_number AS assignee_internal_number,
			u_manager.id, u_manager.username AS manager_username, u_manager.first_name AS manager_first_name, u_manager.last_name AS manager_last_name, u_manager.internal_number AS manager_internal_number,
			g.id, g.name,
			c.id, c.name,
			s.id, s.name,
			COUNT(*) OVER() AS total_count
		FROM %s t
		JOIN %s u_creator ON t.creator_id = u_creator.id
		LEFT JOIN %s u_owner ON t.owner_id = u_owner.id
		LEFT JOIN %s u_assignee ON t.assignee_id = u_assignee.id
		LEFT JOIN %s u_manager ON t.manager_id = u_manager.id
		LEFT JOIN %s g ON t.group_id = g.id
		JOIN %s c ON t.category_id = c.id
		JOIN %s s ON t.site_id = s.id`,
		Tables.Tickets, Tables.Users, Tables.Users, Tables.Users, Tables.Users,
		Tables.Groups, Tables.Categories, Tables.Sites,
	)

	w := &whereBuilder{}
	w.sites(req.SiteIDs)
	w.statuses(req.Status, req.Statuses)
	eq(w, "t.owner_id", req.OwnerID)
	eq(w, "t.assignee_id", req.AssigneeID)
	w.groups(req.GroupIDs, req.IncludeUngroupedAssignedTo)
	eq(w, "t.creator_id", req.CreatorID)
	eq(w, "t.ticket_number", req.Number)
	eq(w, "t.realm_id", req.RealmID)
	w.search(req.Search)
	w.dueDate(req.DueDateFrom, req.DueDateTo)
	w.priorities(req.Priorities)
	w.statusMode(req.Archived)
	w.favorites(req.FavoritesByUser, req.FavoriteType)

	query := base
	if len(w.clauses) > 0 {
		query += " WHERE " + strings.Join(w.clauses, " AND ")
	}

	if req.Sort != nil && *req.Sort != "" {
		field := *req.Sort
		dir := "ASC"
		if rest, ok := strings.CutSuffix(field, "_desc"); ok {
			field = rest
			dir = "DESC"
		} else if rest, ok := strings.CutSuffix(field, "_asc"); ok {
			field = rest
		}
		if col, ok := sortMapping[field]; ok {
			query += fmt.Sprintf(" ORDER BY %s %s", col, dir)
		} else {
			query += " ORDER BY t.created_at DESC"
		}
	} else {
		query += " ORDER BY t.created_at DESC"
	}

	if req.Archived != nil && *req.Archived {
		limit := req.Limit
		if limit <= 0 {
			limit = 20
		}
		w.idx++
		query += fmt.Sprintf(" LIMIT $%d", w.idx)
		w.args = append(w.args, limit)

		offset := req.Offset
		if offset < 0 {
			offset = 0
		}
		w.idx++
		query += fmt.Sprintf(" OFFSET $%d", w.idx)
		w.args = append(w.args, offset)
	}

	rows, err := r.db.Query(ctx, query, w.args...)
	if err != nil {
		return nil, 0, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	var data []*models.Ticket
	total := 0
	for rows.Next() {
		assoc := nullableTicketAssoc{}
		ticket := &models.Ticket{
			Site:     &models.SiteShort{},
			Category: &models.CategoryShort{},
			Creator:  models.UserShort{},
		}
		if err := rows.Scan(
			&ticket.ID, &ticket.Title, &ticket.Description,
			&ticket.Status, &ticket.Priority,
			&ticket.TicketNumber, &ticket.RealmID,
			&ticket.DueDate, &ticket.ClosedAt, &ticket.ResolvedAt, &ticket.CreatedAt, &ticket.UpdatedAt,
			&ticket.Creator.ID, &ticket.Creator.Username, &ticket.Creator.FirstName, &ticket.Creator.LastName, &ticket.Creator.InternalNumber,
			&assoc.OwnerID, &assoc.OwnerUsername, &assoc.OwnerFirstName, &assoc.OwnerLastName, &assoc.OwnerInternalNumber,
			&assoc.AssigneeID, &assoc.AssigneeUsername, &assoc.AssigneeFirstName, &assoc.AssigneeLastName, &assoc.AssigneeInternalNumber,
			&assoc.ManagerID, &assoc.ManagerUsername, &assoc.ManagerFirstName, &assoc.ManagerLastName, &assoc.ManagerInternalNumber,
			&assoc.GroupID, &assoc.GroupName,
			&ticket.Category.ID, &ticket.Category.Name,
			&ticket.Site.ID, &ticket.Site.Name,
			&total,
		); err != nil {
			return nil, 0, MapError(fmt.Errorf("scan row error: %w", err))
		}
		assoc.assign(ticket)
		data = append(data, ticket)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	if data == nil {
		return []*models.Ticket{}, 0, nil
	}
	return data, total, nil
}

func (r *TicketRepo) GetByID(ctx context.Context, req *models.GetTicketByIdDTO) (*models.Ticket, error) {
	query := fmt.Sprintf(`SELECT 
			t.id, t.title, t.description, t.status, t.priority, t.ticket_number, t.realm_id, t.due_date, t.closed_at, t.resolved_at, t.created_at, t.updated_at,
			-- Данные владельца
			u_owner.id, u_owner.username AS owner_username, u_owner.first_name AS owner_first_name, u_owner.last_name AS owner_last_name, u_owner.internal_number AS owner_internal_number,
			-- Данные создателя
			u_creator.id, u_creator.username AS creator_username, u_creator.first_name AS creator_first_name, u_creator.last_name AS creator_last_name, u_creator.internal_number AS creator_internal_number,
			-- Данные исполнителя (может быть null)
			u_assignee.id, u_assignee.username AS assignee_username, u_assignee.first_name AS assignee_first_name, u_assignee.last_name AS assignee_last_name, u_assignee.internal_number AS assignee_internal_number,
			-- Данные менеджера (может быть null)
			u_manager.id, u_manager.username AS manager_username, u_manager.first_name AS manager_first_name, u_manager.last_name AS manager_last_name, u_manager.internal_number AS manager_internal_number,
			-- Данные группы
			g.id, g.name,
			-- Данные категории
			c.id, c.name,
			-- Данные площадки
			s.id, s.name
		FROM %s t
		LEFT JOIN %s u_owner ON t.owner_id = u_owner.id
		JOIN %s u_creator ON t.creator_id = u_creator.id
		LEFT JOIN %s u_assignee ON t.assignee_id = u_assignee.id
		LEFT JOIN %s u_manager ON t.manager_id = u_manager.id
		LEFT JOIN %s g ON t.group_id = g.id
		JOIN %s c ON t.category_id = c.id
		JOIN %s s ON t.site_id = s.id
		WHERE t.id = $1;`,
		Tables.Tickets, Tables.Users, Tables.Users, Tables.Users, Tables.Users,
		Tables.Groups, Tables.Categories, Tables.Sites,
	)

	ticket := &models.Ticket{
		Site:     &models.SiteShort{},
		Category: &models.CategoryShort{},
		Creator:  models.UserShort{},
	}

	assoc := nullableTicketAssoc{}
	if err := r.db.QueryRow(ctx, query, req.ID).Scan(
		&ticket.ID, &ticket.Title, &ticket.Description, &ticket.Status, &ticket.Priority,
		&ticket.TicketNumber, &ticket.RealmID,
		&ticket.DueDate, &ticket.ClosedAt, &ticket.ResolvedAt, &ticket.CreatedAt, &ticket.UpdatedAt,
		&assoc.OwnerID, &assoc.OwnerUsername, &assoc.OwnerFirstName, &assoc.OwnerLastName, &assoc.OwnerInternalNumber,
		&ticket.Creator.ID, &ticket.Creator.Username, &ticket.Creator.FirstName, &ticket.Creator.LastName, &ticket.Creator.InternalNumber,
		&assoc.AssigneeID, &assoc.AssigneeUsername, &assoc.AssigneeFirstName, &assoc.AssigneeLastName, &assoc.AssigneeInternalNumber,
		&assoc.ManagerID, &assoc.ManagerUsername, &assoc.ManagerFirstName, &assoc.ManagerLastName, &assoc.ManagerInternalNumber,
		&assoc.GroupID, &assoc.GroupName,
		&ticket.Category.ID, &ticket.Category.Name,
		&ticket.Site.ID, &ticket.Site.Name,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	assoc.assign(ticket)

	return ticket, nil
}

func (r *TicketRepo) Create(ctx context.Context, tx Tx, dto *models.TicketDTO) error {
	id := uuid.New()
	dto.ID = &id

	numberQuery := fmt.Sprintf(`INSERT INTO %s (realm_id, last_number) VALUES ($1, 1)
		ON CONFLICT (realm_id) DO UPDATE SET last_number = %s.last_number + 1
		RETURNING last_number`,
		Tables.TicketCounters, Tables.TicketCounters,
	)

	var ticketNumber int
	if err := r.getExec(tx).QueryRow(ctx, numberQuery, dto.RealmID).Scan(&ticketNumber); err != nil {
		return MapError(fmt.Errorf("failed to get next ticket number: %w", err))
	}
	dto.TicketNumber = ticketNumber

	query := fmt.Sprintf(`INSERT INTO %s (id, title, description, status, priority, site_id, category_id,
		creator_id, owner_id, group_id, assignee_id, manager_id, due_date, ticket_number, realm_id) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		Tables.Tickets,
	)

	_, err := r.getExec(tx).Exec(
		ctx, query, dto.ID, dto.Title, dto.Description, dto.Status, dto.Priority, dto.SiteID, dto.CategoryID,
		dto.CreatorID, dto.OwnerID, dto.GroupID, dto.AssigneeID, dto.ManagerID, dto.DueDate, ticketNumber, dto.RealmID,
	)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *TicketRepo) Update(ctx context.Context, tx Tx, dto *models.TicketDTO) error {
	sets := make([]string, 0, 12)
	args := []interface{}{dto.ID}
	n := 1

	add := func(jsonKey, column string, value interface{}) {
		if !dto.HasField(jsonKey) {
			return
		}
		n++
		sets = append(sets, fmt.Sprintf("%s=$%d", column, n))
		args = append(args, value)
	}

	add("title", "title", dto.Title)
	add("description", "description", dto.Description)
	add("status", "status", dto.Status)
	add("priority", "priority", dto.Priority)
	add("siteId", "site_id", dto.SiteID)
	add("categoryId", "category_id", dto.CategoryID)
	add("groupId", "group_id", dto.GroupID)
	add("assigneeId", "assignee_id", dto.AssigneeID)
	add("managerId", "manager_id", dto.ManagerID)
	add("ownerId", "owner_id", dto.OwnerID)
	add("dueDate", "due_date", dto.DueDate)
	add("closedAt", "closed_at", dto.ClosedAt)
	add("resolvedAt", "resolved_at", dto.ResolvedAt)

	if len(sets) == 0 {
		return nil
	}

	query := fmt.Sprintf(`UPDATE %s SET %s, updated_at=NOW() WHERE id=$1`,
		Tables.Tickets, strings.Join(sets, ", "),
	)

	_, err := r.getExec(tx).Exec(ctx, query, args...)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *TicketRepo) Delete(ctx context.Context, tx Tx, dto *models.DeleteTicketDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, Tables.Tickets)

	_, err := r.getExec(tx).Exec(ctx, query, dto.ID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *TicketRepo) CloseResolved(ctx context.Context, cutoff time.Time) (int64, error) {
	query := fmt.Sprintf(`UPDATE %s SET status='closed', closed_at=COALESCE(closed_at, NOW()), updated_at=NOW()
		WHERE status='resolved' AND resolved_at IS NOT NULL AND resolved_at <= $1`, Tables.Tickets)

	cmd, err := r.db.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, MapError(fmt.Errorf("failed to auto-close resolved tickets: %w", err))
	}
	return cmd.RowsAffected(), nil
}

// CountNotClosedByGroup возвращает количество незакрытых заявок в группе.
func (r *TicketRepo) CountNotClosedByGroup(ctx context.Context, groupID uuid.UUID) (int, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s
		WHERE group_id = $1 AND status NOT IN ('closed', 'cancelled')`, Tables.Tickets)

	var count int
	if err := r.db.QueryRow(ctx, query, groupID).Scan(&count); err != nil {
		return 0, MapError(fmt.Errorf("failed to count not closed tickets by group: %w", err))
	}
	return count, nil
}

// CountNotClosedByCategory возвращает количество незакрытых заявок в категории.
func (r *TicketRepo) CountNotClosedByCategory(ctx context.Context, categoryID uuid.UUID) (int, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s
		WHERE category_id = $1 AND status NOT IN ('closed', 'cancelled')`, Tables.Tickets)

	var count int
	if err := r.db.QueryRow(ctx, query, categoryID).Scan(&count); err != nil {
		return 0, MapError(fmt.Errorf("failed to count not closed tickets by category: %w", err))
	}
	return count, nil
}
