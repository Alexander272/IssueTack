package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type groupRepo struct {
	db *pgxpool.Pool
}

func NewGroupRepo(db *pgxpool.Pool) *groupRepo {
	return &groupRepo{
		db: db,
	}
}

type Groups interface {
	GetByID(ctx context.Context, req *models.GetGroupDTO) (*models.Group, error)
	Get(ctx context.Context, req *models.GetGroupsDTO) ([]*models.Group, error)
	Create(ctx context.Context, dto *models.GroupDTO) error
	Update(ctx context.Context, dto *models.GroupDTO) error
	Delete(ctx context.Context, dto *models.DelGroupDTO) error

	// Работа с составом группы
	// GetMembers(ctx context.Context, req *models.GetGroupDTO) ([]*models.User, error)
	AddMember(ctx context.Context, dto *models.GroupMemberDTO) error
	RemoveMember(ctx context.Context, dto *models.GroupMemberDTO) error
}

func (r *groupRepo) GetByID(ctx context.Context, req *models.GetGroupDTO) (*models.Group, error) {
	query := fmt.Sprintf(`SELECT id, name, description, created_at, updated_at FROM %s`, Tables.Groups)

	group := &models.Group{}
	err := r.db.QueryRow(ctx, query, req.ID).Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&group.CreatedAt,
		&group.UpdatedAt,
	)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return group, nil
}

func (r *groupRepo) Get(ctx context.Context, req *models.GetGroupsDTO) ([]*models.Group, error) {
	query := fmt.Sprintf(`SELECT id, name, description, created_at, updated_at FROM %s`, Tables.Groups)

	var data []*models.Group
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	for rows.Next() {
		item := &models.Group{}
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

func (r *groupRepo) Create(ctx context.Context, dto *models.GroupDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, name, description) VALUES ($1, $2, $3)`, Tables.Groups)
	dto.ID = uuid.New()

	_, err := r.db.Exec(ctx, query, dto.ID, dto.Name, dto.Description)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *groupRepo) Update(ctx context.Context, dto *models.GroupDTO) error {
	query := fmt.Sprintf(`UPDATE %s SET name=$2, description=$3 WHERE id=$1`, Tables.Groups)

	_, err := r.db.Exec(ctx, query, dto.ID, dto.Name, dto.Description)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *groupRepo) Delete(ctx context.Context, dto *models.DelGroupDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, Tables.Groups)

	_, err := r.db.Exec(ctx, query, dto.ID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *groupRepo) AddMember(ctx context.Context, dto *models.GroupMemberDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (group_id, user_id) VALUES ($1, $2)`, Tables.GroupMembers)

	_, err := r.db.Exec(ctx, query, dto.GroupID, dto.UserID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *groupRepo) RemoveMember(ctx context.Context, dto *models.GroupMemberDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE group_id = $1 AND user_id = $2`, Tables.GroupMembers)

	_, err := r.db.Exec(ctx, query, dto.GroupID, dto.UserID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
