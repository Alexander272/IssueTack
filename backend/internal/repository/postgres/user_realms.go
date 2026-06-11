package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres/pq_models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRealmRepo struct {
	db *pgxpool.Pool
	Transaction
}

func NewUserRealmRepo(db *pgxpool.Pool, tr Transaction) *UserRealmRepo {
	return &UserRealmRepo{
		db:          db,
		Transaction: tr,
	}
}

type UserRealms interface {
	GetAll(ctx context.Context) ([]*models.UserRealm, error)
	GetByUserId(ctx context.Context, userId string) ([]*models.UserRealm, error)
	GetByUserAndRealm(ctx context.Context, userId, realmId string) (*models.UserRealm, error)
	Create(ctx context.Context, tx Tx, dto *models.UserRealmDTO) error
	CreateSeveral(ctx context.Context, tx Tx, dto []*models.UserRealmDTO) error
	Update(ctx context.Context, tx Tx, dto *models.UserRealmDTO) error
	UpdateSeveral(ctx context.Context, tx Tx, dto []*models.UserRealmDTO) error
	Delete(ctx context.Context, tx Tx, id string) error
	DeleteByUserAndRealm(ctx context.Context, tx Tx, userId, realmId string) error
	DeleteSeveral(ctx context.Context, tx Tx, dto []*models.UserRealmDTO) error
}

func (r *UserRealmRepo) GetAll(ctx context.Context) ([]*models.UserRealm, error) {
	query := fmt.Sprintf(`SELECT ur.id, ur.user_id, ur.realm_id, ur.role_id, ur.is_active, ur.created_at,
		    r.slug as role_slug, r.name as role_name, r.level as role_level,
		    rl.name as realm_name, rl.description as realm_description, rl.created_at as realm_created_at
		FROM %s ur
		LEFT JOIN %s r ON ur.role_id = r.id
		LEFT JOIN %s rl ON ur.realm_id = rl.id`,
		Tables.UserRealms, Tables.Roles, Tables.Realms,
	)

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to get all user realms: %w", err))
	}
	defer rows.Close()

	var userRealms []*pq_models.UserRealm
	for rows.Next() {
		item := &pq_models.UserRealm{}
		if err := rows.Scan(
			&item.Id, &item.UserId, &item.RealmId, &item.RoleId, &item.IsActive, &item.RealmCreatedAt,
			&item.RoleSlug, &item.RoleName, &item.RoleLevel,
			&item.RealmName, &item.RealmDescription,
		); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		userRealms = append(userRealms, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return mapUserRealm(userRealms), nil
}

func (r *UserRealmRepo) GetByUserId(ctx context.Context, userId string) ([]*models.UserRealm, error) {
	query := fmt.Sprintf(`SELECT ur.id, ur.user_id, ur.realm_id, ur.role_id, ur.is_active, ur.created_at,
		    r.slug as role_slug, r.name as role_name, r.level as role_level,
		    rl.name as realm_name, rl.description as realm_description, rl.created_at as realm_created_at
		FROM %s ur
		LEFT JOIN %s r ON ur.role_id = r.id
		LEFT JOIN %s rl ON ur.realm_id = rl.id
		WHERE ur.user_id = $1`,
		Tables.UserRealms, Tables.Roles, Tables.Realms,
	)

	rows, err := r.db.Query(ctx, query, userId)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to get user realms by user id: %w", err))
	}
	defer rows.Close()

	var userRealms []*pq_models.UserRealm
	for rows.Next() {
		item := &pq_models.UserRealm{}
		if err := rows.Scan(
			&item.Id, &item.UserId, &item.RealmId, &item.RoleId, &item.IsActive, &item.RealmCreatedAt,
			&item.RoleSlug, &item.RoleName, &item.RoleLevel,
			&item.RealmName, &item.RealmDescription,
		); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		userRealms = append(userRealms, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return mapUserRealm(userRealms), nil
}

func (r *UserRealmRepo) GetByUserAndRealm(ctx context.Context, userId, realmId string) (*models.UserRealm, error) {
	query := fmt.Sprintf(`SELECT ur.id, ur.user_id, ur.realm_id, ur.role_id, ur.is_active, ur.created_at,
		    r.slug as role_slug, r.name as role_name, r.level as role_level,
		    rl.name as realm_name, rl.description as realm_description, rl.created_at as realm_created_at
		FROM %s ur
		LEFT JOIN %s r ON ur.role_id = r.id
		LEFT JOIN %s rl ON ur.realm_id = rl.id
		WHERE ur.user_id = $1 AND ur.realm_id = $2`,
		Tables.UserRealms, Tables.Roles, Tables.Realms,
	)

	item := &pq_models.UserRealm{}
	if err := r.db.QueryRow(ctx, query, userId, realmId).Scan(
		&item.Id, &item.UserId, &item.RealmId, &item.RoleId, &item.IsActive, &item.RealmCreatedAt,
		&item.RoleSlug, &item.RoleName, &item.RoleLevel,
		&item.RealmName, &item.RealmDescription,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to get user realm: %w", err))
	}

	return mapUserRealm([]*pq_models.UserRealm{item})[0], nil
}

func mapUserRealm(rows []*pq_models.UserRealm) []*models.UserRealm {
	data := make([]*models.UserRealm, 0, len(rows))
	for _, ur := range rows {
		role := &models.Role{
			ID:    uuid.MustParse(ur.RoleId),
			Slug:  ur.RoleSlug,
			Name:  ur.RoleName,
			Level: ur.RoleLevel,
		}

		realm := &models.Realm{
			ID:          uuid.MustParse(ur.RealmId),
			Name:        ur.RealmName,
			Description: ur.RealmDescription,
		}

		data = append(data, &models.UserRealm{
			ID:        uuid.MustParse(ur.Id),
			UserID:    uuid.MustParse(ur.UserId),
			RealmID:   uuid.MustParse(ur.RealmId),
			RoleID:    uuid.MustParse(ur.RoleId),
			IsActive:  ur.IsActive,
			CreatedAt: ur.RealmCreatedAt,
			Role:      role,
			Realm:     realm,
		})
	}
	return data
}

func (r *UserRealmRepo) Create(ctx context.Context, tx Tx, dto *models.UserRealmDTO) error {
	id := uuid.New()
	query := fmt.Sprintf(`INSERT INTO %s (id, user_id, realm_id, role_id, is_active) VALUES ($1, $2, $3, $4, $5)`, Tables.UserRealms)

	_, err := r.getExec(tx).Exec(ctx, query, id, dto.UserID, dto.RealmID, dto.RoleID, dto.IsActive)
	if err != nil {
		return MapError(fmt.Errorf("failed to create user realm: %w", err))
	}
	return nil
}

func (r *UserRealmRepo) CreateSeveral(ctx context.Context, tx Tx, dto []*models.UserRealmDTO) error {
	if len(dto) == 0 {
		return nil
	}

	rows := make([][]interface{}, len(dto))
	for i, v := range dto {
		rows[i] = []interface{}{
			uuid.New(),
			v.UserID,
			v.RealmID,
			v.RoleID,
			v.IsActive,
		}
	}

	columns := []string{"id", "user_id", "realm_id", "role_id", "is_active"}
	_, err := r.getExec(tx).CopyFrom(
		ctx,
		pgx.Identifier{Tables.UserRealms},
		columns,
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		return MapError(fmt.Errorf("failed to create user realms: %w", err))
	}
	return nil
}

func (r *UserRealmRepo) Update(ctx context.Context, tx Tx, dto *models.UserRealmDTO) error {
	query := fmt.Sprintf(`UPDATE %s SET role_id = $1, is_active = $2 WHERE user_id = $3 AND realm_id = $4`, Tables.UserRealms)

	_, err := r.getExec(tx).Exec(ctx, query, dto.RoleID, dto.IsActive, dto.UserID, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to update user realm: %w", err))
	}
	return nil
}

func (r *UserRealmRepo) UpdateSeveral(ctx context.Context, tx Tx, dto []*models.UserRealmDTO) error {
	if len(dto) == 0 {
		return nil
	}

	ids := make([]string, len(dto))
	userIds := make([]string, len(dto))
	realmIds := make([]string, len(dto))
	roleIds := make([]string, len(dto))
	isActives := make([]bool, len(dto))

	for i, v := range dto {
		ids[i] = v.ID
		userIds[i] = v.UserID
		realmIds[i] = v.RealmID
		roleIds[i] = v.RoleID
		isActives[i] = v.IsActive
	}

	query := fmt.Sprintf(`
		UPDATE %s AS t
		SET role_id = s.role_id::uuid, is_active = s.is_active
		FROM (
			SELECT * FROM UNNEST(
				$1::text[],
				$2::text[],
				$3::text[],
				$4::text[],
				$5::bool[]
			) AS s(id, user_id, realm_id, role_id, is_active)
		) AS s
		WHERE t.user_id = s.user_id::uuid AND t.realm_id = s.realm_id::uuid`,
		Tables.UserRealms,
	)

	_, err := r.getExec(tx).Exec(ctx, query, ids, userIds, realmIds, roleIds, isActives)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *UserRealmRepo) Delete(ctx context.Context, tx Tx, id string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, Tables.UserRealms)

	if _, err := r.getExec(tx).Exec(ctx, query, id); err != nil {
		return MapError(fmt.Errorf("failed to delete user realm: %w", err))
	}
	return nil
}

func (r *UserRealmRepo) DeleteByUserAndRealm(ctx context.Context, tx Tx, userId, realmId string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE user_id = $1 AND realm_id = $2`, Tables.UserRealms)

	if _, err := r.getExec(tx).Exec(ctx, query, userId, realmId); err != nil {
		return MapError(fmt.Errorf("failed to delete user realm by user and realm: %w", err))
	}
	return nil
}

func (r *UserRealmRepo) DeleteSeveral(ctx context.Context, tx Tx, dto []*models.UserRealmDTO) error {
	if len(dto) == 0 {
		return nil
	}

	ids := make([]string, len(dto))
	for i, v := range dto {
		ids[i] = v.ID
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE id = ANY($1)`, Tables.UserRealms)

	if _, err := r.getExec(tx).Exec(ctx, query, ids); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
