package postgres

import (
	"context"
	"fmt"

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
	Get(ctx context.Context, req *models.GetRoleDTO) (*models.Role, error)
	IsExists(ctx context.Context, roleName string) (bool, error)
	IsExistsById(ctx context.Context, id uuid.UUID) (bool, error)
	Create(ctx context.Context, tx Tx, dto *models.RoleDTO) error
	Update(ctx context.Context, tx Tx, dto *models.RoleDTO) error
	Delete(ctx context.Context, tx Tx, dto *models.DeleteRoleDTO) error

	AssignPermission(ctx context.Context, tx Tx, dto *models.RolePermissionDTO) error
	DeletePermission(ctx context.Context, tx Tx, dto *models.RolePermissionDTO) error
}

func (r *RoleRepo) Get(ctx context.Context, req *models.GetRoleDTO) (*models.Role, error) {
	condition := ""
	params := []interface{}{}
	if req.ID != uuid.Nil {
		params = append(params, req.ID)
		condition = fmt.Sprintf("WHERE id = $%d", len(params))
	}
	if req.Name != "" && req.Realm != "" {
		params = append(params, req.Name, req.Realm)
		condition = fmt.Sprintf("WHERE name = $%d AND realm_id = $%d", len(params)-1, len(params))
	}
	if condition == "" {
		return nil, models.ErrInvalidInput
	}

	query := fmt.Sprintf(`SELECT id, slug, name, realm_id, level, is_system, created_at, updated_at 
		FROM %s %s ORDER BY realm_id, level, slug`,
		Tables.Roles, condition,
	)
	data := &models.Role{}

	err := r.db.QueryRow(ctx, query, req.ID).Scan(
		&data.ID,
		&data.Slug,
		&data.Name,
		&data.Realm,
		&data.Level,
		&data.IsSystem,
		&data.CreatedAt,
		&data.UpdatedAt,
	)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return data, nil
}

func (r *RoleRepo) GetForCasbin(ctx context.Context) ([][3]string, error) {
	query := fmt.Sprintf(`SELECT u.id, r.slug, d.code
		FROM %s ur
		JOIN %s u ON u.id = ur.user_id
		JOIN %s r ON r.id = ur.role_id
		JOIN %s d ON d.id = r.realm_id
	`, Tables.UserRoles, Tables.Users, Tables.Roles, Tables.Realms)

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	var userRoles [][3]string
	for rows.Next() {
		var userID, role, realm string
		if err := rows.Scan(&userID, &role, &realm); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		userRoles = append(userRoles, [3]string{userID, role, realm})
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return userRoles, nil
}

func (r *RoleRepo) IsExists(ctx context.Context, roleName string) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE name = $1)`, Tables.Roles)
	var exists bool

	err := r.db.QueryRow(ctx, query, roleName).Scan(&exists)
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

// func (r *RoleRepo) GetList(ctx context.Context, req *models.GetRoleDTO) ([]*models.Role, error) {
// 	query := fmt.Sprintf(`SELECT id, name, realm_id, level, created_at FROM %s WHERE realm_id = $1 ORDER BY level, name`, Tables.Roles)
// 	roles := []*Role{}
// 	if err := r.db.SelectContext(ctx, &roles, query, req.Realm); err != nil {
// 		return nil, err
// 	}
// 	return roles, nil
// }

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

func (r *RoleRepo) DeletePermission(ctx context.Context, tx Tx, dto *models.RolePermissionDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE role_id=$1 AND permission_id=$2`, Tables.RolePermissions)

	_, err := r.getExec(tx).Exec(ctx, query, dto.RoleID, dto.PermissionID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
