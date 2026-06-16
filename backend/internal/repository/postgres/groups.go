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

	GetMembers(ctx context.Context, req *models.GetGroupDTO) ([]*models.User, error)
	GetMemberCount(ctx context.Context, groupID uuid.UUID) (int, error)
	GetManagedGroups(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetMemberGroups(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	IsMember(ctx context.Context, groupID, userID uuid.UUID) (bool, error)
	AddMember(ctx context.Context, dto *models.GroupMemberDTO) error
	RemoveMember(ctx context.Context, dto *models.GroupMemberDTO) error
}

func (r *groupRepo) GetByID(ctx context.Context, req *models.GetGroupDTO) (*models.Group, error) {
	query := fmt.Sprintf(`
		SELECT g.id, g.name, g.description, g.created_at, g.updated_at,
			g.default_assignee_id, g.manager_id,
			da.id AS da_id, da.name AS da_name,
			m.id AS m_id, m.name AS m_name
		FROM %s g
		LEFT JOIN %s da ON da.id = g.default_assignee_id
		LEFT JOIN %s m ON m.id = g.manager_id
		WHERE g.id = $1
	`, Tables.Groups, Tables.Users, Tables.Users)

	group := &models.Group{}
	var daID, mID *uuid.UUID
	var daName, mName *string
	err := r.db.QueryRow(ctx, query, req.ID).Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&group.CreatedAt,
		&group.UpdatedAt,
		&group.DefaultAssigneeID,
		&group.ManagerID,
		&daID,
		&daName,
		&mID,
		&mName,
	)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	if daID != nil {
		group.DefaultAssignee = &models.UserShort{ID: *daID, FullName: *daName}
	}
	if mID != nil {
		group.Manager = &models.UserShort{ID: *mID, FullName: *mName}
	}
	return group, nil
}

func (r *groupRepo) Get(ctx context.Context, req *models.GetGroupsDTO) ([]*models.Group, error) {
	query := fmt.Sprintf(`
		SELECT g.id, g.name, g.description, g.created_at, g.updated_at,
			g.default_assignee_id, g.manager_id,
			da.id AS da_id, da.name AS da_name,
			m.id AS m_id, m.name AS m_name
		FROM %s g
		LEFT JOIN %s da ON da.id = g.default_assignee_id
		LEFT JOIN %s m ON m.id = g.manager_id
	`, Tables.Groups, Tables.Users, Tables.Users)

	var data []*models.Group
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	for rows.Next() {
		item := &models.Group{}
		var daID, mID *uuid.UUID
		var daName, mName *string
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.DefaultAssigneeID,
			&item.ManagerID,
			&daID,
			&daName,
			&mID,
			&mName,
		); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		if daID != nil {
			item.DefaultAssignee = &models.UserShort{ID: *daID, FullName: *daName}
		}
		if mID != nil {
			item.Manager = &models.UserShort{ID: *mID, FullName: *mName}
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
	query := fmt.Sprintf(`UPDATE %s SET name=$2, description=$3, default_assignee_id=$4, manager_id=$5 WHERE id=$1`, Tables.Groups)

	_, err := r.db.Exec(ctx, query, dto.ID, dto.Name, dto.Description, dto.DefaultAssigneeID, dto.ManagerID)
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

func (r *groupRepo) GetMembers(ctx context.Context, req *models.GetGroupDTO) ([]*models.User, error) {
	query := fmt.Sprintf(`
		SELECT u.id, u.email, u.name, u.created_at, u.updated_at
		FROM %s gm
		JOIN %s u ON u.id = gm.user_id
		WHERE gm.group_id = $1
	`, Tables.GroupMembers, Tables.Users)

	rows, err := r.db.Query(ctx, query, req.ID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	var data []*models.User
	for rows.Next() {
		item := &models.User{}
		if err := rows.Scan(&item.ID, &item.Email, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

func (r *groupRepo) GetMemberCount(ctx context.Context, groupID uuid.UUID) (int, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE group_id = $1`, Tables.GroupMembers)

	var count int
	err := r.db.QueryRow(ctx, query, groupID).Scan(&count)
	if err != nil {
		return 0, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return count, nil
}

func (r *groupRepo) GetManagedGroups(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	query := fmt.Sprintf(`SELECT id FROM %s WHERE manager_id = $1`, Tables.Groups)

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	var data []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, id)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

func (r *groupRepo) GetMemberGroups(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	query := fmt.Sprintf(`SELECT group_id FROM %s WHERE user_id = $1`, Tables.GroupMembers)

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	var data []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, id)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

func (r *groupRepo) IsMember(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE group_id = $1 AND user_id = $2)`, Tables.GroupMembers)

	var exists bool
	err := r.db.QueryRow(ctx, query, groupID, userID).Scan(&exists)
	if err != nil {
		return false, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return exists, nil
}
