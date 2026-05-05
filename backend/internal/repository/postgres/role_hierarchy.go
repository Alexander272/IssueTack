package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
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
	GetInheritedRoles(ctx context.Context, req *models.GetRoleInheritance) ([]string, error)
	SyncRoleInheritance(ctx context.Context, req *models.GetRoleInheritance) ([]*models.SyncRoleInheritance, error)
	LoadPolicy(ctx context.Context, req *models.GetPoliciesDTO) ([]*models.SyncRoleInheritance, error)
	AddInheritance(ctx context.Context, tx Tx, dto *models.RoleHierarchyDTO) error
	RemoveInheritance(ctx context.Context, tx Tx, dto *models.RoleHierarchyDTO) error
}

// GetInheritedRoles — получить все родительские роли (прямые + цепочки)
// Используется для предрасчёта прав при синхронизации с Casbin
func (r *RoleHierarchyRepo) GetInheritedRoles(ctx context.Context, req *models.GetRoleInheritance) ([]string, error) {
	query := fmt.Sprintf(`WITH RECURSIVE inheritance_tree AS (
            -- Базовый случай: прямые родители
            SELECT r2.slug as parent_code
            FROM %s ri
            JOIN %s r1 ON ri.role_id = r1.id
            JOIN %s r2 ON ri.parent_role_id = r2.id
            WHERE r1.slug = $1 AND ri.realm_id = $2 AND r2.is_active = true
            
            UNION
            
            -- Рекурсия: родители родителей
            SELECT r3.slug
            FROM inheritance_tree it
            JOIN %s ri ON ri.role_id = (SELECT id FROM %s WHERE slug = it.parent_code)
            JOIN %s r3 ON ri.parent_role_id = r3.id
            WHERE ri.realm_id = $2 AND r3.is_active = true
        )
        SELECT DISTINCT parent_code FROM inheritance_tree`,
		Tables.RoleHierarchy, Tables.Roles, Tables.Roles,
		Tables.RoleHierarchy, Tables.Roles, Tables.Roles,
	)

	rows, err := r.db.Query(ctx, query, req.Role, req.Realm)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	var parents []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		parents = append(parents, code)
	}
	return parents, nil
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

func (r *RoleHierarchyRepo) LoadPolicy(ctx context.Context, req *models.GetPoliciesDTO) ([]*models.SyncRoleInheritance, error) {
	condition := ""
	args := make([]any, 0, 1)
	if req.RealmId != "" {
		condition = "rh.realm_id = $1 AND"
		args = append(args, req.RealmId)
	}

	query := fmt.Sprintf(`SELECT r1.slug, r2.slug, rh.realm_id
        FROM %s rh
        JOIN %s r1 ON rh.role_id = r1.id
        JOIN %s r2 ON rh.parent_role_id = r2.id
        WHERE %s r2.is_active = true`,
		Tables.RoleHierarchy, Tables.Roles, Tables.Roles, condition,
	)

	rows, err := r.db.Query(ctx, query, args...)
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
		// // g(дочерняя_роль, родительская_роль, домен)
		// casbin.AddGroupingPolicy(roleCode, parentCode, domain)
		data = append(data, item)
	}

	return data, nil
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
