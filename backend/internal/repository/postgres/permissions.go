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
	Transaction
}

func NewPermissionRepo(db *pgxpool.Pool, tr Transaction) *PermissionRepo {
	return &PermissionRepo{
		db:          db,
		Transaction: tr,
	}
}

type Permissions interface {
	GetById(ctx context.Context, id uuid.UUID) (*models.Permission, error)
	GetByRole(ctx context.Context, req *models.GetPermsByRoleDTO) ([]*models.Permission, error)
	LoadPolicy(ctx context.Context, req *models.GetPoliciesDTO) ([]*models.Permission, error)
	Create(ctx context.Context, tx Tx, dto *models.PermissionDTO) error
	Delete(ctx context.Context, tx Tx, dto *models.DeletePermissionDTO) error
}

func (r *PermissionRepo) GetById(ctx context.Context, id uuid.UUID) (*models.Permission, error) {
	query := fmt.Sprintf(`SELECT id, d.code, p.object, p.action
		FROM %s p JOIN %s d ON d.id = p.realm_id
		WHERE id=$1`,
		Tables.Realms, Tables.Permissions,
	)
	data := &models.Permission{}
	err := r.db.QueryRow(ctx, query, id).Scan(&data.ID, &data.Realm, &data.Object, &data.Action)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return data, nil
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

func (r *PermissionRepo) LoadPolicy(ctx context.Context, req *models.GetPoliciesDTO) ([]*models.Permission, error) {
	condition := ""
	args := make([]any, 0, 1)
	if req.RealmId != "" {
		condition = "WHERE d.realm_id = $1"
		args = append(args, req.RealmId)
	}

	query := fmt.Sprintf(`SELECT r.slug, d.code, p.object, p.action
		FROM %s rp
		JOIN %s r ON r.id = rp.role_id
		JOIN %s d ON d.id = r.realm_id
		JOIN %s p ON p.id = rp.permission_id
		%s`,
		Tables.RolePermissions, Tables.Roles, Tables.Realms, Tables.Permissions, condition,
	)

	rows, err := r.db.Query(ctx, query, args...)
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

func (r *PermissionRepo) Create(ctx context.Context, tx Tx, dto *models.PermissionDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, realm_id, object, action) VALUES ($1, $2, $3, $4)`,
		Tables.Permissions,
	)
	dto.ID = uuid.New()

	_, err := r.getExec(tx).Exec(ctx, query, dto.ID, dto.RealmID, dto.Object, dto.Action)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *PermissionRepo) Delete(ctx context.Context, tx Tx, dto *models.DeletePermissionDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id=$1`, Tables.Permissions)

	_, err := r.getExec(tx).Exec(ctx, query, dto.ID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
