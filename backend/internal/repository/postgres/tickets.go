package postgres

import (
	"context"
	"fmt"

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
	return nil, fmt.Errorf("not implemented")
}

func (r *TicketRepo) GetByID(ctx context.Context, req *models.GetTicketByIdDTO) (*models.Ticket, error) {
	query := fmt.Sprintf(`SELECT 
			t.id, t.title, t.description, t.status, t.priority, t.due_date, t.created_at,
			-- Данные владельца
			u_owner.id, CONCAT_WS(' ', u_owner.last_name, u_owner.first_name) AS owner_full_name,
			-- Данные создателя
			u_creator.id, CONCAT_WS(' ', u_creator.last_name, u_creator.first_name) AS creator_full_name,
			-- Данные исполнителя (может быть null)
			u_assignee.id, CONCAT_WS(' ', u_assignee.last_name, u_assignee.first_name) AS assignee_full_name,
			-- Данные менеджера (может быть null)
			u_manager.id, CONCAT_WS(' ', u_manager.last_name, u_manager.first_name) AS manager_full_name,
			-- Данные группы
			g.id, g.name,
			-- Данные категории
			c.id, c.name,
			-- Данные площадки
			s.id, s.name
		FROM %s t
		JOIN %s u_owner ON t.owner_id = u_owner.id
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

	//TODO это может не заработать. потому что могут вернуться null
	ticket := &models.Ticket{
		Site:     &models.SiteShort{},
		Category: &models.CategoryShort{},
		Creator:  models.UserShort{},
		Owner:    &models.UserShort{},
		Assignee: &models.UserShort{},
		Manager:  &models.UserShort{},
		Group:    &models.GroupShort{},
	}
	if err := r.db.QueryRow(ctx, query, req.ID).Scan(
		&ticket.ID, &ticket.Title, &ticket.Description, &ticket.Status, &ticket.Priority, &ticket.DueDate, &ticket.CreatedAt,
		&ticket.Owner.ID, &ticket.Owner.FullName,
		&ticket.Creator.ID, &ticket.Creator.FullName,
		&ticket.Assignee.ID, &ticket.Assignee.FullName,
		&ticket.Manager.ID, &ticket.Manager.FullName,
		&ticket.Group.ID, &ticket.Group.Name,
		&ticket.Category.ID, &ticket.Category.Name,
		&ticket.Site.ID, &ticket.Site.Name,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	return ticket, nil
}

func (r *TicketRepo) Create(ctx context.Context, tx Tx, dto *models.TicketDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, title, description, status, priority, site_id, category_id,
		creator_id, owner_id, group_id, assignee_id, manager_id, due_date) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		Tables.Tickets,
	)
	if dto.ID == uuid.Nil {
		dto.ID = uuid.New()
	}

	_, err := r.getExec(tx).Exec(
		ctx, query, dto.ID, dto.Title, dto.Description, dto.Status, dto.Priority, dto.SiteID, dto.CategoryID,
		dto.CreatorID, dto.OwnerID, dto.GroupID, dto.AssigneeID, dto.ManagerID, dto.DueDate,
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
