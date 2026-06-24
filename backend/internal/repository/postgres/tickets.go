package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
	Get(ctx context.Context, req *models.TicketFilter) ([]*models.Ticket, error)
	GetByID(ctx context.Context, req *models.GetTicketByIdDTO) (*models.Ticket, error)
	Create(ctx context.Context, tx Tx, dto *models.TicketDTO) error
	Update(ctx context.Context, tx Tx, dto *models.TicketDTO) error
	Delete(ctx context.Context, tx Tx, dto *models.DeleteTicketDTO) error
}

func (r *TicketRepo) Get(ctx context.Context, req *models.TicketFilter) ([]*models.Ticket, error) {
	base := fmt.Sprintf(`SELECT 
			t.id, t.title, t.description, t.status, t.priority, t.ticket_number, t.realm_id, t.due_date, t.closed_at, t.created_at, t.updated_at,
			u_creator.id, u_creator.username AS creator_username, u_creator.first_name AS creator_first_name, u_creator.last_name AS creator_last_name, u_creator.internal_number AS creator_internal_number,
			u_owner.id, u_owner.username AS owner_username, u_owner.first_name AS owner_first_name, u_owner.last_name AS owner_last_name, u_owner.internal_number AS owner_internal_number,
			u_assignee.id, u_assignee.username AS assignee_username, u_assignee.first_name AS assignee_first_name, u_assignee.last_name AS assignee_last_name, u_assignee.internal_number AS assignee_internal_number,
			u_manager.id, u_manager.username AS manager_username, u_manager.first_name AS manager_first_name, u_manager.last_name AS manager_last_name, u_manager.internal_number AS manager_internal_number,
			g.id, g.name,
			c.id, c.name,
			s.id, s.name
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

	where := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.SiteID != nil {
		where = append(where, fmt.Sprintf("t.site_id = $%d", argIdx))
		args = append(args, *req.SiteID)
		argIdx++
	}
	if req.Status != nil {
		where = append(where, fmt.Sprintf("t.status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.OwnerID != nil {
		where = append(where, fmt.Sprintf("t.owner_id = $%d", argIdx))
		args = append(args, *req.OwnerID)
		argIdx++
	}
	if req.AssigneeID != nil {
		where = append(where, fmt.Sprintf("t.assignee_id = $%d", argIdx))
		args = append(args, *req.AssigneeID)
		argIdx++
	}
	if req.GroupID != nil {
		where = append(where, fmt.Sprintf("t.group_id = $%d", argIdx))
		args = append(args, *req.GroupID)
		argIdx++
	}
	if req.Number != nil {
		where = append(where, fmt.Sprintf("t.ticket_number = $%d", argIdx))
		args = append(args, *req.Number)
		argIdx++
	}
	if req.RealmID != nil {
		where = append(where, fmt.Sprintf("t.realm_id = $%d", argIdx))
		args = append(args, *req.RealmID)
		argIdx++
	}
	if len(req.GroupIDs) > 0 {
		ids := make([]string, len(req.GroupIDs))
		for i, gid := range req.GroupIDs {
			ids[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, gid)
			argIdx++
		}
		where = append(where, "t.group_id IN ("+strings.Join(ids, ",")+")")
	}

	query := base
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY t.created_at DESC"

	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, limit)
	argIdx++

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}
	query += fmt.Sprintf(" OFFSET $%d", argIdx)
	args = append(args, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	var data []*models.Ticket
	for rows.Next() {
		var (
			assigneeID            *uuid.UUID
			assigneeUsername      *string
			assigneeFirstName     *string
			assigneeLastName      *string
			assigneeInternalNumber *string
			managerID              *uuid.UUID
			managerUsername        *string
			managerFirstName       *string
			managerLastName        *string
			managerInternalNumber  *string
			groupID                *uuid.UUID
			groupName              *string
		)
		ticket := &models.Ticket{
			Site:     &models.SiteShort{},
			Category: &models.CategoryShort{},
			Creator:  models.UserShort{},
			Owner:    &models.UserShort{},
		}
		if err := rows.Scan(
			&ticket.ID, &ticket.Title, &ticket.Description,
			&ticket.Status, &ticket.Priority,
			&ticket.TicketNumber, &ticket.RealmID,
			&ticket.DueDate, &ticket.ClosedAt, &ticket.CreatedAt, &ticket.UpdatedAt,
			&ticket.Creator.ID, &ticket.Creator.Username, &ticket.Creator.FirstName, &ticket.Creator.LastName, &ticket.Creator.InternalNumber,
			&ticket.Owner.ID, &ticket.Owner.Username, &ticket.Owner.FirstName, &ticket.Owner.LastName, &ticket.Owner.InternalNumber,
			&assigneeID, &assigneeUsername, &assigneeFirstName, &assigneeLastName, &assigneeInternalNumber,
			&managerID, &managerUsername, &managerFirstName, &managerLastName, &managerInternalNumber,
			&groupID, &groupName,
			&ticket.Category.ID, &ticket.Category.Name,
			&ticket.Site.ID, &ticket.Site.Name,
		); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		if assigneeID != nil {
			ticket.Assignee = &models.UserShort{ID: *assigneeID, Username: *assigneeUsername, FirstName: *assigneeFirstName, LastName: *assigneeLastName, InternalNumber: *assigneeInternalNumber}
		}
		if managerID != nil {
			ticket.Manager = &models.UserShort{ID: *managerID, Username: *managerUsername, FirstName: *managerFirstName, LastName: *managerLastName, InternalNumber: *managerInternalNumber}
		}
		if groupID != nil {
			ticket.Group = &models.GroupShort{ID: *groupID, Name: *groupName}
		}
		data = append(data, ticket)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	if data == nil {
		return []*models.Ticket{}, nil
	}
	return data, nil
}

func (r *TicketRepo) GetByID(ctx context.Context, req *models.GetTicketByIdDTO) (*models.Ticket, error) {
	query := fmt.Sprintf(`SELECT 
			t.id, t.title, t.description, t.status, t.priority, t.ticket_number, t.realm_id, t.due_date, t.closed_at, t.created_at, t.updated_at,
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

	var ownerID, assigneeID, managerID *uuid.UUID
	var ownerUsername, ownerFirstName, ownerLastName *string
	var assigneeUsername, assigneeFirstName, assigneeLastName *string
	var managerUsername, managerFirstName, managerLastName *string
	var ownerInternalNumber, assigneeInternalNumber, managerInternalNumber *string
	var groupID *uuid.UUID
	var groupName *string

	if err := r.db.QueryRow(ctx, query, req.ID).Scan(
		&ticket.ID, &ticket.Title, &ticket.Description, &ticket.Status, &ticket.Priority,
		&ticket.TicketNumber, &ticket.RealmID,
		&ticket.DueDate, &ticket.ClosedAt, &ticket.CreatedAt, &ticket.UpdatedAt,
		&ownerID, &ownerUsername, &ownerFirstName, &ownerLastName, &ownerInternalNumber,
		&ticket.Creator.ID, &ticket.Creator.Username, &ticket.Creator.FirstName, &ticket.Creator.LastName, &ticket.Creator.InternalNumber,
		&assigneeID, &assigneeUsername, &assigneeFirstName, &assigneeLastName, &assigneeInternalNumber,
		&managerID, &managerUsername, &managerFirstName, &managerLastName, &managerInternalNumber,
		&groupID, &groupName,
		&ticket.Category.ID, &ticket.Category.Name,
		&ticket.Site.ID, &ticket.Site.Name,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	if ownerID != nil {
		ticket.Owner = &models.UserShort{ID: *ownerID, Username: *ownerUsername, FirstName: *ownerFirstName, LastName: *ownerLastName, InternalNumber: *ownerInternalNumber}
	}
	if assigneeID != nil {
		ticket.Assignee = &models.UserShort{ID: *assigneeID, Username: *assigneeUsername, FirstName: *assigneeFirstName, LastName: *assigneeLastName, InternalNumber: *assigneeInternalNumber}
	}
	if managerID != nil {
		ticket.Manager = &models.UserShort{ID: *managerID, Username: *managerUsername, FirstName: *managerFirstName, LastName: *managerLastName, InternalNumber: *managerInternalNumber}
	}
	if groupID != nil {
		ticket.Group = &models.GroupShort{ID: *groupID, Name: *groupName}
	}

	return ticket, nil
}

func (r *TicketRepo) Create(ctx context.Context, tx Tx, dto *models.TicketDTO) error {
	if dto.ID == uuid.Nil {
		dto.ID = uuid.New()
	}

	numberQuery := fmt.Sprintf(`INSERT INTO %s (realm_id, last_number) VALUES ($1, 1)
		ON CONFLICT (realm_id) DO UPDATE SET last_number = %s.last_number + 1
		RETURNING last_number`,
		Tables.TicketCounters, Tables.TicketCounters,
	)

	var ticketNumber int
	if err := r.getExec(tx).QueryRow(ctx, numberQuery, dto.RealmID).Scan(&ticketNumber); err != nil {
		return MapError(fmt.Errorf("failed to get next ticket number: %w", err))
	}

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
	query := fmt.Sprintf(`UPDATE %s 
		SET title=$2, description=$3, status=$4, priority=$5, site_id=$6, assignee_id=$7, 
		due_date=$8, closed_at=$9, category_id=$10, group_id=$11, owner_id=$12, updated_at=NOW()
		WHERE id=$1`,
		Tables.Tickets,
	)

	_, err := r.getExec(tx).Exec(
		ctx, query, dto.ID, dto.Title, dto.Description, dto.Status, dto.Priority, dto.SiteID, dto.AssigneeID,
		dto.DueDate, dto.ClosedAt, dto.CategoryID, dto.GroupID, dto.OwnerID,
	)
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
