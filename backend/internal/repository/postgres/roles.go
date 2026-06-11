package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoleRepo struct {
	db *pgxpool.Pool
	Transaction
}

func NewRoleRepo(db *pgxpool.Pool, tr Transaction) *RoleRepo {
	return &RoleRepo{
		db:          db,
		Transaction: tr,
	}
}

type Roles interface {
	GetOne(ctx context.Context, req *models.GetRoleDTO) (*models.Role, error)
	GetAll(ctx context.Context) ([]*models.Role, error)
	GetUserCount(ctx context.Context, roleIDs []string) (map[string]int, error)
	GetIDsBySlugs(ctx context.Context, realmID uuid.UUID, slugs []string) (map[string]uuid.UUID, error)
	IsExists(ctx context.Context, realmID uuid.UUID, roleName string) (bool, error)
	IsExistsById(ctx context.Context, id uuid.UUID) (bool, error)
	Create(ctx context.Context, tx Tx, dto *models.RoleDTO) error
	Update(ctx context.Context, tx Tx, dto *models.RoleDTO) error
	Delete(ctx context.Context, tx Tx, dto *models.DeleteRoleDTO) error

	AssignPermission(ctx context.Context, tx Tx, dto *models.RolePermissionDTO) error
	AssignPermissions(ctx context.Context, tx Tx, roleID uuid.UUID, permissionIDs []uuid.UUID) error
	DeletePermission(ctx context.Context, tx Tx, dto *models.RolePermissionDTO) error
}

func (r *RoleRepo) GetOne(ctx context.Context, req *models.GetRoleDTO) (*models.Role, error) {
	condition := ""
	params := []interface{}{}
	if req.ID != uuid.Nil {
		params = append(params, req.ID)
		condition = fmt.Sprintf("WHERE id = $%d", len(params))
	}
	if req.Slug != "" {
		params = append(params, req.Slug, req.Realm)
		condition = fmt.Sprintf("WHERE slug = $%d AND realm_id = $%d", len(params)-1, len(params))
	}
	if condition == "" {
		return nil, MapError(models.ErrInvalidInput)
	}

	query := fmt.Sprintf(`SELECT id, slug, name, description, level, is_active, is_system, is_editable, created_at, updated_at FROM %s %s`,
		Tables.Roles, condition,
	)
	data := &models.Role{}

	err := r.db.QueryRow(ctx, query, params...).Scan(
		&data.ID,
		&data.Slug,
		&data.Name,
		&data.Description,
		&data.Level,
		&data.IsActive,
		&data.IsSystem,
		&data.IsEditable,
		&data.CreatedAt,
		&data.UpdatedAt,
	)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return data, nil
}

func (r *RoleRepo) IsExists(ctx context.Context, realmID uuid.UUID, roleName string) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE name = $1 AND realm_id = $2)`, Tables.Roles)
	var exists bool

	err := r.db.QueryRow(ctx, query, roleName, realmID).Scan(&exists)
	if err != nil {
		return false, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return exists, nil
}
func (r *RoleRepo) IsExistsById(ctx context.Context, id uuid.UUID) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1 AND is_active = true)`, Tables.Roles)
	var exists bool

	err := r.db.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return exists, nil
}

func (r *RoleRepo) GetAll(ctx context.Context) ([]*models.Role, error) {
	query := fmt.Sprintf(`SELECT id, slug, name, realm_id, description, level, is_active, is_system, is_editable, created_at, updated_at 
		FROM %s ORDER BY realm_id, level, slug`,
		Tables.Roles,
	)

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	var data []*models.Role
	for rows.Next() {
		item := &models.Role{}
		if err := rows.Scan(
			&item.ID, &item.Slug, &item.Name, &item.Realm, &item.Description,
			&item.Level, &item.IsActive, &item.IsSystem, &item.IsEditable,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

func (r *RoleRepo) GetUserCount(ctx context.Context, roleIDs []string) (map[string]int, error) {
	if len(roleIDs) == 0 {
		return make(map[string]int), nil
	}

	query := fmt.Sprintf(`SELECT role_id, COUNT(*) FROM %s 
		WHERE role_id = ANY($1)
		GROUP BY role_id`,
		Tables.UserRoles,
	)

	rows, err := r.db.Query(ctx, query, roleIDs)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to get user counts: %w", err))
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var roleID string
		var count int
		if err := rows.Scan(&roleID, &count); err != nil {
			return nil, MapError(fmt.Errorf("scan count error: %w", err))
		}
		counts[roleID] = count
	}

	return counts, nil
}

func (r *RoleRepo) GetIDsBySlugs(ctx context.Context, realmID uuid.UUID, slugs []string) (map[string]uuid.UUID, error) {
	if len(slugs) == 0 {
		return make(map[string]uuid.UUID), nil
	}

	query := fmt.Sprintf(`SELECT slug, id FROM %s WHERE slug = ANY($1) AND realm_id = $2`,
		Tables.Roles,
	)

	rows, err := r.db.Query(ctx, query, slugs, realmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	result := make(map[string]uuid.UUID)
	for rows.Next() {
		var slug string
		var id uuid.UUID
		if err := rows.Scan(&slug, &id); err != nil {
			return nil, MapError(fmt.Errorf("failed to scan row: %w", err))
		}
		result[slug] = id
	}

	return result, nil
}

func (r *RoleRepo) Create(ctx context.Context, tx Tx, dto *models.RoleDTO) error {
	if dto.Slug == "root" || dto.Slug == "superadmin" {
		return models.ErrReservedRole
	}

	query := fmt.Sprintf(`INSERT INTO %s (slug, name, realm_id, level, is_system)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		Tables.Roles,
	)

	err := r.getExec(tx).QueryRow(
		ctx, query, dto.Slug, dto.Name, dto.RealmID, dto.Level, dto.IsSystem,
	).Scan(&dto.ID, &dto.CreatedAt)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *RoleRepo) Update(ctx context.Context, tx Tx, dto *models.RoleDTO) error {
	if dto.Slug == "root" || dto.Slug == "superadmin" {
		return models.ErrReservedRole
	}

	query := fmt.Sprintf(`UPDATE %s SET name=$1, realm_id=$2, level=$3, slug=$4, is_system=$5, updated_at=NOW() WHERE id=$6`,
		Tables.Roles,
	)

	_, err := r.getExec(tx).Exec(ctx, query, dto.Name, dto.RealmID, dto.Level, dto.Slug, dto.IsSystem, dto.ID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *RoleRepo) Delete(ctx context.Context, tx Tx, dto *models.DeleteRoleDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id=$1 AND NOT is_system`, Tables.Roles)

	_, err := r.getExec(tx).Exec(ctx, query, dto.ID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *RoleRepo) AssignPermission(ctx context.Context, tx Tx, dto *models.RolePermissionDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (role_id, permission_id) VALUES ($1, $2)`, Tables.RolePermissions)

	_, err := r.getExec(tx).Exec(ctx, query, dto.RoleID, dto.PermissionID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *RoleRepo) AssignPermissions(ctx context.Context, tx Tx, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	if len(permissionIDs) == 0 {
		return nil
	}

	values := make([]string, 0, len(permissionIDs))
	args := make([]interface{}, 0, len(permissionIDs)*2)
	for i, permID := range permissionIDs {
		values = append(values, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		args = append(args, roleID, permID)
	}

	query := fmt.Sprintf(`INSERT INTO %s (role_id, permission_id) VALUES %s`, Tables.RolePermissions, strings.Join(values, ", "))

	_, err := r.getExec(tx).Exec(ctx, query, args...)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *RoleRepo) DeletePermission(ctx context.Context, tx Tx, dto *models.RolePermissionDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE role_id=$1 AND permission_id=$2`, Tables.RolePermissions)

	_, err := r.getExec(tx).Exec(ctx, query, dto.RoleID, dto.PermissionID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
