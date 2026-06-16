package postgres

import (
	"context"
	"fmt"
	"strings"

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
	LoadPolicy(ctx context.Context) ([]*models.Permission, error)
	Sync(ctx context.Context, tx Tx, dto []*models.PermissionDTO) error
	GetById(ctx context.Context, id uuid.UUID) (*models.Permission, error)
	GetAll(ctx context.Context) ([]*models.Permission, error)
	GetByRole(ctx context.Context, req *models.GetPermsByRoleDTO) ([]*models.Permission, error)
	GetInheritedByRole(ctx context.Context, roleID uuid.UUID) (map[uuid.UUID]struct{}, error)
	GetRolePermissionsMap(ctx context.Context, tx Tx, roleID uuid.UUID) (map[uuid.UUID]bool, error)
	ReplacePermissions(ctx context.Context, tx Tx, roleID uuid.UUID, permissionIDs []uuid.UUID) error
	Count(ctx context.Context, req *models.GetPermsCountDTO) (*models.PermsWithCount, error)
	CountForAll(ctx context.Context, roleToDescendants map[string][]string) (map[string]models.PermsWithCount, error)
	Create(ctx context.Context, tx Tx, dto *models.PermissionDTO) error
	Delete(ctx context.Context, tx Tx, dto *models.DeletePermissionDTO) error
	DeleteByKeys(ctx context.Context, tx Tx, dto []*models.PermissionDTO) error
}

func (r *PermissionRepo) LoadPolicy(ctx context.Context) ([]*models.Permission, error) {
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

func (r *PermissionRepo) Sync(ctx context.Context, tx Tx, dto []*models.PermissionDTO) error {
	if len(dto) == 0 {
		return nil
	}
	values := []string{}
	args := []interface{}{}

	for _, v := range dto {
		values = append(values, fmt.Sprintf("($%d, $%d, $%d)", len(args)+1, len(args)+2, len(args)+3))
		args = append(args, v.Object, v.Action, v.Description)
	}

	query := fmt.Sprintf(`INSERT INTO %s (object, action, description)
			VALUES %s
			ON CONFLICT (object, action) 
			DO UPDATE SET description = EXCLUDED.description`,
		Tables.Permissions, strings.Join(values, ", "),
	)

	_, err := r.getExec(tx).Exec(ctx, query, args...)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *PermissionRepo) GetById(ctx context.Context, id uuid.UUID) (*models.Permission, error) {
	query := fmt.Sprintf(`SELECT id, p.object, p.action
		FROM %s p WHERE id=$1`,
		Tables.Permissions,
	)
	data := &models.Permission{}
	err := r.db.QueryRow(ctx, query, id).Scan(&data.ID, &data.Object, &data.Action)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return data, nil
}

func (r *PermissionRepo) GetByRole(ctx context.Context, req *models.GetPermsByRoleDTO) ([]*models.Permission, error) {
	query := fmt.Sprintf(`SELECT p.id, r.slug, d.code, p.object, p.action
		FROM %s rp
		JOIN %s r ON r.id = rp.role_id
		JOIN %s d ON d.id = r.realm_id
		JOIN %s p ON p.id = rp.permission_id
		WHERE r.slug = $1`,
		Tables.RolePermissions, Tables.Roles, Tables.Realms, Tables.Permissions,
	)

	data := make([]*models.Permission, 0, 50)
	rows, err := r.db.Query(ctx, query, req.Role)
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

func (r *PermissionRepo) Count(ctx context.Context, req *models.GetPermsCountDTO) (*models.PermsWithCount, error) {
	query := fmt.Sprintf(`SELECT 
			array_agg(*) FILTER (WHERE role_id = $1) AS own_permissions,
			array_agg(DISTINCT permission_id) FILTER (WHERE role_id = ANY($2)) AS inherited_permissions
		FROM %s
		WHERE role_id = $1 OR role_id = ANY($2)`,
		Tables.RolePermissions,
	)

	data := &models.PermsWithCount{}
	err := r.db.QueryRow(ctx, query, req.Role, req.Inherited).Scan(&data.Own.Items, &data.Inherited.Items)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	data.Own.Count = len(data.Own.Items)
	data.Inherited.Count = len(data.Inherited.Items)

	data.Total = models.Perm{
		Items: append(data.Own.Items, data.Inherited.Items...),
		Count: len(data.Total.Items),
	}

	return data, nil
}
func (r *PermissionRepo) CountForAll(ctx context.Context, roleToDescendants map[string][]string) (map[string]models.PermsWithCount, error) {
	if len(roleToDescendants) == 0 {
		return make(map[string]models.PermsWithCount), nil
	}

	res := make(map[string]models.PermsWithCount)

	// Для каждой роли считаем её собственные permissions
	for roleSlug := range roleToDescendants {
		c := models.PermsWithCount{}

		// Считаем собственные permissions роли
		ownQuery := fmt.Sprintf(`
			SELECT array_agg(rp.permission_id)
			FROM %s rp
			JOIN %s r ON rp.role_id = r.id
			WHERE r.slug = $1`,
			Tables.RolePermissions, Tables.Roles,
		)

		err := r.db.QueryRow(ctx, ownQuery, roleSlug).Scan(&c.Own.Items)
		if err != nil {
			return nil, MapError(fmt.Errorf("failed to count own perms for role %s: %w", roleSlug, err))
		}
		c.Own.Count = len(c.Own.Items)

		c.Total = c.Own
		res[roleSlug] = c
	}

	// Собираем все уникальные descendant slug'и
	allDescendants := make([]string, 0, len(roleToDescendants))
	descendantSet := make(map[string]struct{})
	for _, descendants := range roleToDescendants {
		for _, d := range descendants {
			if _, exists := descendantSet[d]; !exists {
				descendantSet[d] = struct{}{}
				allDescendants = append(allDescendants, d)
			}
		}
	}

	// Считаем permissions для каждого descendant
	descendantPerms := make(map[string]models.Perm)
	if len(allDescendants) > 0 {
		descQuery := fmt.Sprintf(`
			SELECT r.slug, array_agg(rp.permission_id)
			FROM %s rp
			JOIN %s r ON rp.role_id = r.id
			WHERE r.slug = ANY($1)
			GROUP BY r.slug`,
			Tables.RolePermissions, Tables.Roles,
		)

		rows, err := r.db.Query(ctx, descQuery, allDescendants)
		if err != nil {
			return nil, MapError(fmt.Errorf("failed to count descendant perms: %w", err))
		}
		defer rows.Close()

		for rows.Next() {
			var slug string
			var perms []string
			if err := rows.Scan(&slug, &perms); err != nil {
				return nil, err
			}
			descendantPerms[slug] = models.Perm{Items: perms, Count: len(perms)}
		}
		if err := rows.Err(); err != nil {
			return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
		}
	}

	// Для каждой роли суммируем permissions всех её descendants
	for roleSlug, descendants := range roleToDescendants {
		c := res[roleSlug]
		for _, d := range descendants {
			c.Inherited.Items = append(c.Inherited.Items, descendantPerms[d].Items...)
			c.Inherited.Count += descendantPerms[d].Count
		}
		c.Total = models.Perm{
			Items: append(c.Own.Items, c.Inherited.Items...),
			Count: c.Own.Count + c.Inherited.Count,
		}
		res[roleSlug] = c
	}

	return res, nil
}

func (r *PermissionRepo) GetAll(ctx context.Context) ([]*models.Permission, error) {
	//TODO можно еще добавить уровни для сортировки
	query := fmt.Sprintf(`SELECT id, object, action, description FROM %s ORDER BY object, action`, Tables.Permissions)

	data := make([]*models.Permission, 0, 50)
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	for rows.Next() {
		item := &models.Permission{}
		if err := rows.Scan(&item.ID, &item.Object, &item.Action, &item.Description); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

func (r *PermissionRepo) GetRolePermissionsMap(ctx context.Context, tx Tx, roleID uuid.UUID) (map[uuid.UUID]bool, error) {
	query := fmt.Sprintf(`SELECT permission_id FROM %s WHERE role_id = $1`, Tables.RolePermissions)

	rows, err := r.getExec(tx).Query(ctx, query, roleID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to get role permissions: %w", err))
	}
	defer rows.Close()

	result := make(map[uuid.UUID]bool)
	for rows.Next() {
		var permID uuid.UUID
		if err := rows.Scan(&permID); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		result[permID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return result, nil
}

func (r *PermissionRepo) GetInheritedByRole(ctx context.Context, roleID uuid.UUID) (map[uuid.UUID]struct{}, error) {
	query := fmt.Sprintf(`WITH RECURSIVE sub_roles AS (
			-- Базовый случай: берем только ПРЯМЫХ детей роли $1
			SELECT role_id 
			FROM %s 
			WHERE parent_role_id = $1
			
			UNION ALL
			
			-- Рекурсия: спускаемся дальше ко всем внукам, правнукам и т.д.
			SELECT rh.role_id 
			FROM %s rh
			JOIN sub_roles sr ON rh.parent_role_id = sr.role_id
		)
		SELECT DISTINCT rp.permission_id 
		FROM %s rp 
		WHERE rp.role_id IN (SELECT role_id FROM sub_roles)`,
		Tables.RoleHierarchy, Tables.RoleHierarchy, Tables.RolePermissions,
	)

	rows, err := r.db.Query(ctx, query, roleID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	result := make(map[uuid.UUID]struct{})
	for rows.Next() {
		var permID uuid.UUID
		if err := rows.Scan(&permID); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		result[permID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return result, nil
}

func (r *PermissionRepo) Create(ctx context.Context, tx Tx, dto *models.PermissionDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, object, action, description) VALUES ($1, $2, $3, $4)`,
		Tables.Permissions,
	)
	dto.ID = uuid.New()

	_, err := r.getExec(tx).Exec(ctx, query, dto.ID, dto.Object, dto.Action, dto.Description)
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

func (r *PermissionRepo) DeleteByKeys(ctx context.Context, tx Tx, dto []*models.PermissionDTO) error {
	if len(dto) == 0 {
		return nil
	}

	placeholders := make([]string, 0, len(dto)*2)
	args := make([]interface{}, 0, len(dto)*2)
	for _, v := range dto {
		placeholders = append(placeholders, fmt.Sprintf("($%d::text, $%d::text)", len(args)+1, len(args)+2))
		args = append(args, v.Object, v.Action)
	}

	// НО проще и надежнее использовать расширение unnest или values:
	query := fmt.Sprintf(`DELETE FROM %s 
        WHERE (object, action) NOT IN (
            SELECT * FROM (VALUES %s) AS t(obj, act)
        )`,
		Tables.Permissions,
		strings.Join(placeholders, ","),
	)

	_, err := r.getExec(tx).Exec(ctx, query, args...)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *PermissionRepo) ReplacePermissions(ctx context.Context, tx Tx, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	exec := r.getExec(tx)

	query := fmt.Sprintf(`DELETE FROM %s WHERE role_id = $1`, Tables.RolePermissions)
	_, err := exec.Exec(ctx, query, roleID)
	if err != nil {
		return MapError(fmt.Errorf("failed to delete old permissions: %w", err))
	}

	if len(permissionIDs) == 0 {
		return nil
	}

	values := make([]string, 0, len(permissionIDs))
	args := make([]interface{}, 0, len(permissionIDs)*2)
	for i, id := range permissionIDs {
		values = append(values, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		args = append(args, roleID, id)
	}

	query = fmt.Sprintf(`INSERT INTO %s (role_id, permission_id) VALUES %s`, Tables.RolePermissions, strings.Join(values, ", "))

	_, err = exec.Exec(ctx, query, args...)
	if err != nil {
		return MapError(fmt.Errorf("failed to insert permissions: %w", err))
	}

	return nil
}
