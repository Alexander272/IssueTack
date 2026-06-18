package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RealmRepo struct {
	db *pgxpool.Pool
	Transaction
}

func NewRealmRepo(db *pgxpool.Pool, tr Transaction) *RealmRepo {
	return &RealmRepo{
		db:          db,
		Transaction: tr,
	}
}

type Realm interface {
	GetAll(ctx context.Context) ([]*models.Realm, error)
	GetByID(ctx context.Context, req *models.GetRealmByIdDTO) (*models.Realm, error)
	Create(ctx context.Context, tx Tx, dto *models.RealmDTO) error
	Update(ctx context.Context, tx Tx, dto *models.RealmDTO) error
	Delete(ctx context.Context, tx Tx, dto *models.DeleteRealmDTO) error
}

func (r *RealmRepo) GetAll(ctx context.Context) ([]*models.Realm, error) {
	query := fmt.Sprintf(`SELECT id, name, code, description, is_active, created_at, updated_at FROM %s ORDER BY name`, Tables.Realms)

	var data []*models.Realm
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	for rows.Next() {
		item := &models.Realm{}
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Code, &item.Description, &item.IsActive,
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

func (d *RealmRepo) GetByID(ctx context.Context, req *models.GetRealmByIdDTO) (*models.Realm, error) {
	query := fmt.Sprintf(`SELECT id, code, name, description, is_active, created_at, updated_at FROM %s WHERE id = $1`, Tables.Realms)
	data := &models.Realm{}

	err := d.getExec(nil).QueryRow(ctx, query, req.ID).Scan(
		&data.ID, &data.Code, &data.Name, &data.Description, &data.IsActive,
		&data.CreatedAt, &data.UpdatedAt,
	)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return data, nil
}

func (d *RealmRepo) Create(ctx context.Context, tx Tx, dto *models.RealmDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, code, name, description, is_active) VALUES ($1, $2, $3, $4, $5)`, Tables.Realms)
	dto.ID = uuid.New()

	_, err := d.getExec(tx).Exec(ctx, query, dto.ID, dto.Code, dto.Name, dto.Description, dto.IsActive)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (d *RealmRepo) Update(ctx context.Context, tx Tx, dto *models.RealmDTO) error {
	query := fmt.Sprintf(`UPDATE %s SET code=$1, name=$2, description=$3, is_active=$4, updated_at=now() WHERE id=$5`, Tables.Realms)

	_, err := d.getExec(tx).Exec(ctx, query, dto.Code, dto.Name, dto.Description, dto.IsActive, dto.ID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (d *RealmRepo) Delete(ctx context.Context, tx Tx, dto *models.DeleteRealmDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, Tables.Realms)

	_, err := d.getExec(tx).Exec(ctx, query, dto.ID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
