package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PermissionRepo struct {
	db *pgxpool.Pool
}

func NewPermissionRepo(db *pgxpool.Pool) *PermissionRepo {
	return &PermissionRepo{
		db: db,
	}
}

type Permissions interface {
	GetByRole(ctx context.Context, req *models.GetPermsByRoleDTO) ([]*models.Permission, error)
	Create(ctx context.Context, dto *models.PermissionDTO) error
	Delete(ctx context.Context, dto *models.DeletePermissionDTO) error
}

func (r *PermissionRepo) GetByRole(ctx context.Context, req *models.GetPermsByRoleDTO) ([]*models.Permission, error) {
	query := fmt.Sprintf(`SELECT r.slug, d.code, p.object, p.action
		FROM %s rp
		JOIN %s r ON r.id = rp.role_id
		JOIN %s d ON d.id = r.realm_id
		JOIN %s p ON p.id = rp.permission_id
		WHERE r.slug = $1`,
		Tables.RolePermissions, Tables.Roles, Tables.Realms, Tables.Permissions,
	)

	data := make([]*models.Permission, 0, 50)
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	for rows.Next() {
		item := &models.Permission{}
		if err := rows.Scan(&item.ID, &item.Role, &item.Realm, &item.Object, &item.Action); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

func (r *PermissionRepo) GetForCasbin(ctx context.Context) ([]*models.Permission, error) {
	query := fmt.Sprintf(`SELECT r.slug, d.code, p.object, p.action
		FROM %s rp
		JOIN %s r ON r.id = rp.role_id
		JOIN %s d ON d.id = r.realm_id
		JOIN %s p ON p.id = rp.permission_id`,
		Tables.RolePermissions, Tables.Roles, Tables.Realms, Tables.Permissions,
	)

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	permissions := make([]*models.Permission, 0, 50)
	for rows.Next() {
		item := &models.Permission{}
		if err := rows.Scan(&item.Role, &item.Realm, &item.Object, &item.Action); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		permissions = append(permissions, item)
	}

	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return permissions, nil
}

func (r *PermissionRepo) Create(ctx context.Context, dto *models.PermissionDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, realm_id, object, action) VALUES ($1, $2, $3, $4)`,
		Tables.Permissions,
	)
	dto.ID = uuid.New()

	_, err := r.db.Exec(ctx, query, dto.ID, dto.RealmID, dto.Object, dto.Action)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *PermissionRepo) Delete(ctx context.Context, dto *models.DeletePermissionDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id=$1`, Tables.Permissions)

	_, err := r.db.Exec(ctx, query, dto.ID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
