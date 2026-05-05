package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepo struct {
	db *pgxpool.Pool
	Transaction
}

func NewUserRepo(db *pgxpool.Pool, tr Transaction) *userRepo {
	return &userRepo{
		db:          db,
		Transaction: tr,
	}
}

type Users interface {
	LoadPolicy(ctx context.Context, req *models.GetPoliciesDTO) ([]*models.UserRole, error)
	AssignRole(ctx context.Context, tx Tx, dto *models.UserRoleDTO) error
	DeleteRole(ctx context.Context, tx Tx, dto *models.UserRoleDTO) error
}

func (r *userRepo) LoadPolicy(ctx context.Context, req *models.GetPoliciesDTO) ([]*models.UserRole, error) {
	condition := ""
	args := make([]any, 0, 1)
	if req.RealmId != "" {
		condition = "WHERE r.realm_id = $1"
		args = append(args, req.RealmId)
	}

	query := fmt.Sprintf(`SELECT u.id, r.name, r.realm_id
        FROM %s ur
        JOIN %s u ON u.id = ur.user_id
        JOIN %s r ON r.id = ur.role_id
		%s`,
		Tables.UserRoles, Tables.Users, Tables.Roles, condition,
	)

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := make([]*models.UserRole, 0, 50)
	for rows.Next() {
		item := &models.UserRole{}
		if err := rows.Scan(&item.UserID, &item.RoleName, &item.Realm); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

func (r *userRepo) AssignRole(ctx context.Context, tx Tx, dto *models.UserRoleDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (user_id, role_id) VALUES ($1, $2)`, Tables.UserRoles)

	_, err := r.getExec(tx).Exec(ctx, query, dto.UserID, dto.RoleID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *userRepo) DeleteRole(ctx context.Context, tx Tx, dto *models.UserRoleDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE user_id=$1 AND role_id=$2`, Tables.UserRoles)

	_, err := r.getExec(tx).Exec(ctx, query, dto.UserID, dto.RoleID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
