package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoleHierarchyRepo struct {
	db *pgxpool.Pool
	Transaction
}

func NewRoleHierarchyRepo(db *pgxpool.Pool, tr Transaction) *RoleHierarchyRepo {
	return &RoleHierarchyRepo{
		db:          db,
		Transaction: tr,
	}
}

type RoleHierarchy interface {
	LoadPolicy(ctx context.Context) ([]*models.SyncRoleInheritance, error)
	GetInheritedRoles(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error)
	GetRoleDescendants(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error)
	GetDirectChildren(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error)
	SyncRoleInheritance(ctx context.Context, req *models.GetRoleInheritance) ([]*models.SyncRoleInheritance, error)
	AddInheritance(ctx context.Context, tx Tx, dto *models.RoleHierarchyDTO) error
	AddInheritances(ctx context.Context, tx Tx, realmID uuid.UUID, roleID uuid.UUID, parentRoleIDs []uuid.UUID) error
	RemoveInheritance(ctx context.Context, tx Tx, dto *models.RoleHierarchyDTO) error
	RemoveInheritances(ctx context.Context, tx Tx, roleID uuid.UUID, parentRoleIDs []uuid.UUID) error
}

func (r *RoleHierarchyRepo) LoadPolicy(ctx context.Context) ([]*models.SyncRoleInheritance, error) {
	query := fmt.Sprintf(`SELECT r1.slug, r2.slug, rh.realm_id
        FROM %s rh
        JOIN %s r1 ON rh.role_id = r1.id
        JOIN %s r2 ON rh.parent_role_id = r2.id
        WHERE r2.is_active = true`,
		Tables.RoleHierarchy, Tables.Roles, Tables.Roles,
	)

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()
	data := make([]*models.SyncRoleInheritance, 0, 5)

	for rows.Next() {
		item := &models.SyncRoleInheritance{}
		if err := rows.Scan(&item.Role, &item.ParentRole, &item.Realm); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}

	return data, nil
}

func (r *RoleHierarchyRepo) GetInheritedRoles(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error) {
	if len(req.Roles) == 0 {
		return make(map[string][]string), nil
	}

	query := fmt.Sprintf(`WITH RECURSIVE inheritance_tree AS (
			SELECT 
				r1.id as root_id,
				r1.slug as root_slug,
				r2.id as parent_id,
				r2.slug as parent_slug
			FROM %s ri
			JOIN %s r1 ON ri.parent_role_id = r1.id
			JOIN %s r2 ON ri.role_id = r2.id
			WHERE r1.slug = ANY($1)
			AND r2.is_active = true

			UNION ALL

			SELECT 
				it.root_id,
				it.root_slug,
				r3.id,
				r3.slug
			FROM inheritance_tree it
			JOIN %s ri ON ri.parent_role_id = it.parent_id
			JOIN %s r3 ON ri.role_id = r3.id
			WHERE r3.is_active = true
		)
		SELECT DISTINCT root_slug, parent_slug
		FROM inheritance_tree`,
		Tables.RoleHierarchy, Tables.Roles, Tables.Roles,
		Tables.RoleHierarchy, Tables.Roles,
	)

	rows, err := r.db.Query(ctx, query, req.Roles)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var root, parent string
		if err := rows.Scan(&root, &parent); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		result[root] = append(result[root], parent)
	}

	return result, nil
}

// SyncRoleInheritance — используется для синхронизации наследования ролей с Casbin
func (r *RoleHierarchyRepo) SyncRoleInheritance(ctx context.Context, req *models.GetRoleInheritance) ([]*models.SyncRoleInheritance, error) {
	query := fmt.Sprintf(`SELECT r2.slug 
        FROM %s ri
        JOIN %s r1 ON ri.role_id = r1.id
        JOIN %s r2 ON ri.parent_role_id = r2.id
        WHERE r1.slug = $1 AND ri.realm_id = $2 AND r2.is_active = true`,
		Tables.RoleHierarchy, Tables.Roles, Tables.Roles,
	)

	rows, err := r.db.Query(ctx, query, req.Role, req.Realm)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()
	data := make([]*models.SyncRoleInheritance, 0, 5)

	for rows.Next() {
		var parentCode string
		if err := rows.Scan(&parentCode); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		// // g(дочерняя_роль, родительская_роль, домен)
		// casbin.AddGroupingPolicy(roleCode, parentCode, domain)
		data = append(data, &models.SyncRoleInheritance{Role: req.Role, ParentRole: parentCode, Realm: req.Realm})
	}

	return data, nil
}

func (r *RoleHierarchyRepo) GetRoleDescendants(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error) {
	if len(req.Roles) == 0 {
		return make(map[string][]string), nil
	}

	query := fmt.Sprintf(`WITH RECURSIVE descendants_tree AS (
			SELECT
				r1.id as root_id,
				r1.slug as root_slug,
				r2.id as child_id,
				r2.slug as child_slug
			FROM %s ri
			JOIN %s r1 ON ri.parent_role_id = r1.id
			JOIN %s r2 ON ri.role_id = r2.id
			WHERE r1.slug = ANY($1)
			AND r2.is_active = true

			UNION ALL

			SELECT
				dt.root_id,
				dt.root_slug,
				r3.id,
				r3.slug
			FROM descendants_tree dt
			JOIN %s ri ON ri.parent_role_id = dt.child_id
			JOIN %s r3 ON ri.role_id = r3.id
			WHERE r3.is_active = true
		)
		SELECT DISTINCT root_slug, child_slug
		FROM descendants_tree`,
		Tables.RoleHierarchy, Tables.Roles, Tables.Roles,
		Tables.RoleHierarchy, Tables.Roles,
	)

	rows, err := r.db.Query(ctx, query, req.Roles)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	result := make(map[string][]string)
	for _, slug := range req.Roles {
		result[slug] = []string{}
	}

	for rows.Next() {
		var root, child string
		if err := rows.Scan(&root, &child); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		result[root] = append(result[root], child)
	}

	return result, nil
}

func (r *RoleHierarchyRepo) GetDirectChildren(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error) {
	if len(req.Roles) == 0 {
		return make(map[string][]string), nil
	}

	query := fmt.Sprintf(`SELECT r1.slug, r2.slug
		FROM %s ri
		JOIN %s r1 ON ri.parent_role_id = r1.id
		JOIN %s r2 ON ri.role_id = r2.id
		WHERE r1.slug = ANY($1) AND r2.is_active = true`,
		Tables.RoleHierarchy, Tables.Roles, Tables.Roles,
	)

	rows, err := r.db.Query(ctx, query, req.Roles)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	result := make(map[string][]string)
	for _, slug := range req.Roles {
		result[slug] = []string{}
	}

	for rows.Next() {
		var parent, child string
		if err := rows.Scan(&parent, &child); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		result[parent] = append(result[parent], child)
	}

	return result, nil
}

func (r *RoleHierarchyRepo) AddInheritance(ctx context.Context, tx Tx, dto *models.RoleHierarchyDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (role_id, parent_role_id, realm_id) 
		VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		Tables.RoleHierarchy,
	)

	// Вставка (уникальность обеспечена БД с помощью trigger)
	_, err := r.getExec(tx).Exec(ctx, query, dto.RoleID, dto.ParentRoleID, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *RoleHierarchyRepo) AddInheritances(ctx context.Context, tx Tx, realmID uuid.UUID, roleID uuid.UUID, parentRoleIDs []uuid.UUID) error {
	if len(parentRoleIDs) == 0 {
		return nil
	}

	values := make([]string, 0, len(parentRoleIDs))
	args := make([]interface{}, 0, len(parentRoleIDs)*3)
	for i, parentID := range parentRoleIDs {
		values = append(values, fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3))
		args = append(args, roleID, parentID, realmID)
	}

	query := fmt.Sprintf(`INSERT INTO %s (parent_role_id, role_id, realm_id) VALUES %s ON CONFLICT DO NOTHING`,
		Tables.RoleHierarchy, strings.Join(values, ", "))

	_, err := r.getExec(tx).Exec(ctx, query, args...)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *RoleHierarchyRepo) RemoveInheritances(ctx context.Context, tx Tx, roleID uuid.UUID, parentRoleIDs []uuid.UUID) error {
	if len(parentRoleIDs) == 0 {
		return nil
	}

	placeholders := make([]string, 0, len(parentRoleIDs))
	args := []interface{}{roleID}
	for _, parentID := range parentRoleIDs {
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)+1))
		args = append(args, parentID)
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE parent_role_id = $1 AND role_id IN (%s)`,
		Tables.RoleHierarchy, strings.Join(placeholders, ", "))

	_, err := r.getExec(tx).Exec(ctx, query, args...)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *RoleHierarchyRepo) RemoveInheritance(ctx context.Context, tx Tx, dto *models.RoleHierarchyDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE role_id = $1 AND parent_role_id = $2`,
		Tables.RoleHierarchy,
	)

	_, err := r.getExec(tx).Exec(ctx, query, dto.RoleID, dto.ParentRoleID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

// func (s *RBACService) AddRoleInheritance(ctx context.Context, child, parent, location string) error {
//     // 1. Сохраняем в бизнес-таблицу (для отображения в админке)
//     query := `INSERT INTO role_hierarchy (child_role, parent_role, location_code) VALUES ($1, $2, $3)`
//     _, err := s.pool.Exec(ctx, query, child, parent, location)
//     if err != nil {
//         return err
//     }

//     return err
// }
