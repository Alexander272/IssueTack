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
	Transaction
}

func NewGroupRepo(db *pgxpool.Pool, tr Transaction) *groupRepo {
	return &groupRepo{
		db: db, Transaction: tr,
	}
}

type Groups interface {
	GetByID(ctx context.Context, req *models.GetGroupDTO) (*models.Group, error)
	Get(ctx context.Context, req *models.GetGroupsDTO) ([]*models.Group, error)
	Create(ctx context.Context, tx Tx, dto *models.GroupDTO) error
	Update(ctx context.Context, tx Tx, dto *models.GroupDTO) error
	Delete(ctx context.Context, dto *models.DelGroupDTO) error

	GetMembers(ctx context.Context, req *models.GetGroupDTO) ([]*models.UserShort, error)
	GetMembersMap(ctx context.Context, groupIDs []uuid.UUID) (map[uuid.UUID][]*models.UserShort, error)
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
			da.id AS da_id, da.username AS da_username, da.first_name AS da_first_name, da.last_name AS da_last_name, da.internal_number AS da_internal_number,
			m.id AS m_id, m.username AS m_username, m.first_name AS m_first_name, m.last_name AS m_last_name, m.internal_number AS m_internal_number
		FROM %s g
		LEFT JOIN %s da ON da.id = g.default_assignee_id
		LEFT JOIN %s m ON m.id = g.manager_id
		WHERE g.id = $1
	`, Tables.Groups, Tables.Users, Tables.Users)

	group := &models.Group{}
	var daID, mID *uuid.UUID
	var daUsername, daFirstName, daLastName *string
	var mUsername, mFirstName, mLastName *string
	var daInternalNumber, mInternalNumber *string
	err := r.db.QueryRow(ctx, query, req.ID).Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&group.CreatedAt,
		&group.UpdatedAt,
		&group.DefaultAssigneeID,
		&group.ManagerID,
		&daID,
		&daUsername,
		&daFirstName,
		&daLastName,
		&daInternalNumber,
		&mID,
		&mUsername,
		&mFirstName,
		&mLastName,
		&mInternalNumber,
	)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	if daID != nil {
		group.DefaultAssignee = &models.UserShort{ID: *daID, Username: *daUsername, FirstName: *daFirstName, LastName: *daLastName, InternalNumber: *daInternalNumber}
	}
	if mID != nil {
		group.Manager = &models.UserShort{ID: *mID, Username: *mUsername, FirstName: *mFirstName, LastName: *mLastName, InternalNumber: *mInternalNumber}
	}
	return group, nil
}

func (r *groupRepo) Get(ctx context.Context, req *models.GetGroupsDTO) ([]*models.Group, error) {
	query := fmt.Sprintf(`
		SELECT g.id, g.name, g.description, g.created_at, g.updated_at,
			g.default_assignee_id, g.manager_id,
			da.id AS da_id, da.username AS da_username, da.first_name AS da_first_name, da.last_name AS da_last_name, 
			da.internal_number AS da_internal_number, da.email AS da_email,
			m.id AS m_id, m.username AS m_username, m.first_name AS m_first_name, m.last_name AS m_last_name, 
			m.internal_number AS m_internal_number, m.email AS m_email
		FROM %s g
		LEFT JOIN %s da ON da.id = g.default_assignee_id
		LEFT JOIN %s m ON m.id = g.manager_id
	`, Tables.Groups, Tables.Users, Tables.Users)

	data := []*models.Group{}
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	for rows.Next() {
		item := &models.Group{}
		var daID, mID *uuid.UUID
		var daUsername, daFirstName, daLastName, daEmail *string
		var mUsername, mFirstName, mLastName, mEmail *string
		var daInternalNumber, mInternalNumber *string
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.DefaultAssigneeID,
			&item.ManagerID,
			&daID,
			&daUsername,
			&daFirstName,
			&daLastName,
			&daInternalNumber,
			&daEmail,
			&mID,
			&mUsername,
			&mFirstName,
			&mLastName,
			&mInternalNumber,
			&mEmail,
		); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		if daID != nil {
			item.DefaultAssignee = &models.UserShort{
				ID: *daID, Username: *daUsername, FirstName: *daFirstName, LastName: *daLastName,
				InternalNumber: *daInternalNumber, Email: *daEmail,
			}
		}
		if mID != nil {
			item.Manager = &models.UserShort{
				ID: *mID, Username: *mUsername, FirstName: *mFirstName, LastName: *mLastName,
				InternalNumber: *mInternalNumber, Email: *mEmail,
			}
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

func (r *groupRepo) Create(ctx context.Context, tx Tx, dto *models.GroupDTO) error {
	exec := r.getExec(tx)

	query := fmt.Sprintf(`INSERT INTO %s (id, realm_id, name, description, manager_id, default_assignee_id) 
		VALUES ($1, $2, $3, $4, $5, $6)`,
		Tables.Groups,
	)
	dto.ID = uuid.New()

	if _, err := exec.Exec(ctx, query, dto.ID, dto.RealmID, dto.Name, dto.Description, dto.ManagerID, dto.DefaultAssigneeID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	if err := r.setMembers(ctx, exec, dto.ID, dto.MemberIDs); err != nil {
		return err
	}

	return nil
}

func (r *groupRepo) Update(ctx context.Context, tx Tx, dto *models.GroupDTO) error {
	exec := r.getExec(tx)

	query := fmt.Sprintf(`UPDATE %s SET name=$2, description=$3, default_assignee_id=$4, manager_id=$5 WHERE id=$1`, Tables.Groups)

	if _, err := exec.Exec(ctx, query, dto.ID, dto.Name, dto.Description, dto.DefaultAssigneeID, dto.ManagerID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	deleteQuery := fmt.Sprintf(`DELETE FROM %s WHERE group_id = $1`, Tables.GroupMembers)
	if _, err := exec.Exec(ctx, deleteQuery, dto.ID); err != nil {
		return MapError(fmt.Errorf("failed to delete members: %w", err))
	}

	if err := r.setMembers(ctx, exec, dto.ID, dto.MemberIDs); err != nil {
		return err
	}

	return nil
}

func (r *groupRepo) setMembers(ctx context.Context, exec QueryExecutor, groupID uuid.UUID, memberIDs []uuid.UUID) error {
	if len(memberIDs) == 0 {
		return nil
	}

	query := fmt.Sprintf(`INSERT INTO %s (group_id, user_id) VALUES ($1, $2)`, Tables.GroupMembers)
	for _, mid := range memberIDs {
		if _, err := exec.Exec(ctx, query, groupID, mid); err != nil {
			return MapError(fmt.Errorf("failed to insert member: %w", err))
		}
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

func (r *groupRepo) GetMembers(ctx context.Context, req *models.GetGroupDTO) ([]*models.UserShort, error) {
	query := fmt.Sprintf(`
		SELECT u.id, u.username, u.first_name, u.last_name, u.email, u.internal_number
		FROM %s gm
		JOIN %s u ON u.id = gm.user_id
		WHERE gm.group_id = $1
	`, Tables.GroupMembers, Tables.Users)

	rows, err := r.db.Query(ctx, query, req.ID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.UserShort{}
	for rows.Next() {
		item := &models.UserShort{}
		if err := rows.Scan(&item.ID, &item.Username, &item.FirstName, &item.LastName, &item.Email, &item.InternalNumber); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

func (r *groupRepo) GetMembersMap(ctx context.Context, groupIDs []uuid.UUID) (map[uuid.UUID][]*models.UserShort, error) {
	if len(groupIDs) == 0 {
		return make(map[uuid.UUID][]*models.UserShort), nil
	}

	query := fmt.Sprintf(`
		SELECT gm.group_id, u.id, u.username, u.first_name, u.last_name, u.email, u.internal_number
		FROM %s gm
		JOIN %s u ON u.id = gm.user_id
		WHERE gm.group_id = ANY($1)
	`, Tables.GroupMembers, Tables.Users)

	rows, err := r.db.Query(ctx, query, groupIDs)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]*models.UserShort)
	for rows.Next() {
		var groupID uuid.UUID
		item := &models.UserShort{}
		if err := rows.Scan(&groupID, &item.ID, &item.Username, &item.FirstName, &item.LastName, &item.Email, &item.InternalNumber); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		result[groupID] = append(result[groupID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return result, nil
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
