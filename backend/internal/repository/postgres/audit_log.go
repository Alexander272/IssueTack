package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type auditRepo struct {
	db *pgxpool.Pool
	Transaction
}

func NewAuditRepo(db *pgxpool.Pool, tr Transaction) *auditRepo {
	return &auditRepo{
		db:          db,
		Transaction: tr,
	}
}

type AuditLogs interface {
	Get(ctx context.Context, req *models.GetAuditLogsDTO) ([]*models.AuditLog, error)
	GetByRealm(ctx context.Context, req *models.GetAuditLogsByRealmDTO) ([]*models.AuditLog, error)
	Create(ctx context.Context, tx Tx, dto *models.AuditLogDTO) error
	CreateSeveral(ctx context.Context, tx Tx, dto []*models.AuditLogDTO) error
}

func (r *auditRepo) Get(ctx context.Context, req *models.GetAuditLogsDTO) ([]*models.AuditLog, error) {
	// query := fmt.Sprintf(``)

	return nil, fmt.Errorf("not implemented")
}

func (r *auditRepo) GetByRealm(ctx context.Context, req *models.GetAuditLogsByRealmDTO) ([]*models.AuditLog, error) {
	// query := fmt.Sprintf(``)

	return nil, fmt.Errorf("not implemented")
}

func (r *auditRepo) Create(ctx context.Context, tx Tx, dto *models.AuditLogDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, changed_by, action, role_id, rule_id, realm_id, user_id, old_values, new_values) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		Tables.AuditLogs,
	)
	dto.ID = uuid.New()

	_, err := r.getExec(tx).Exec(ctx, query,
		dto.ID, dto.ChangedBy, dto.Action, dto.RoleID, dto.RuleID, dto.RealmID, dto.UserID, dto.OldValues, dto.NewValues,
	)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *auditRepo) CreateSeveral(ctx context.Context, tx Tx, dto []*models.AuditLogDTO) error {
	if len(dto) == 0 {
		return nil
	}

	rows := make([][]interface{}, len(dto))

	for i, v := range dto {
		if v.ID == uuid.Nil {
			v.ID = uuid.New()
		}

		rows[i] = []interface{}{
			v.ID,
			v.ChangedBy,
			v.Action,
			v.RoleID,
			v.RuleID,
			v.RealmID,
			v.UserID,
			v.OldValues,
			v.NewValues,
		}
	}

	columns := []string{"id", "changed_by", "action", "role_id", "rule_id", "realm_id", "user_id", "old_values", "new_values"}
	_, err := r.getExec(tx).CopyFrom(
		ctx,
		pgx.Identifier{Tables.AuditLogs},
		columns,
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		return MapError(fmt.Errorf("failed to execute query. error: %w", err))
	}
	return nil
}
