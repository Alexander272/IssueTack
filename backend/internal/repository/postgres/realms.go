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
	GetByID(ctx context.Context, req *models.GetRealmByIdDTO) (*models.Realm, error)
	Create(ctx context.Context, tx Tx, dto *models.RealmDTO) error
	Update(ctx context.Context, tx Tx, dto *models.RealmDTO) error
	Delete(ctx context.Context, tx Tx, dto *models.DeleteRealmDTO) error
}

func (d *RealmRepo) GetByID(ctx context.Context, req *models.GetRealmByIdDTO) (*models.Realm, error) {
	query := fmt.Sprintf(`SELECT id, code, name, created_at, updated_at FROM %s WHERE id = $1`, Tables.Realms)
	data := &models.Realm{}

	err := d.getExec(nil).QueryRow(ctx, query, req.ID).Scan(
		&data.ID, &data.Code, &data.Name,
		&data.CreatedAt, &data.UpdatedAt,
	)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return data, nil
}

// func (d *RealmRepo) GetList(ctx context.Context, req *models.GetRealmDTO) ([]*models.Realm, error) {
// 	query := fmt.Sprintf(`SELECT * FROM %s WHERE code LIKE $1 ORDER BY name`, Tables.Realms)

// 	var Realms []Realm
// 	err := d.getExec(nil).Query(
// 		ctx, &Realms, query, fmt.Sprintf("%%%s%%", req.Code),
// 	).Scan(&Realms)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return Realms, nil
// }

func (d *RealmRepo) Create(ctx context.Context, tx Tx, dto *models.RealmDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, code, name) VALUES ($1, $2, $3)`, Tables.Realms)
	dto.ID = uuid.New()

	_, err := d.getExec(tx).Exec(ctx, query, dto.Code, dto.Name)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (d *RealmRepo) Update(ctx context.Context, tx Tx, dto *models.RealmDTO) error {
	query := fmt.Sprintf(`UPDATE %s SET code=$1, name=$2 WHERE id=$3`, Tables.Realms)

	_, err := d.getExec(tx).Exec(ctx, query, dto.Code, dto.Name, dto.ID)
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
