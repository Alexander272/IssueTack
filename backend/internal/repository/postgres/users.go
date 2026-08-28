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
	LoadPolicy(ctx context.Context) ([]*models.UserRole, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.UserData, error)
	GetByLogin(ctx context.Context, login string) (*models.UserData, error)
	GetByMattermostID(ctx context.Context, mattermostID string) (*models.UserData, error)
	GetAll(ctx context.Context, realmID *uuid.UUID) ([]*models.UserData, error)
	CreateSeveral(ctx context.Context, tx Tx, dto []*models.UserDataDTO) error
	Update(ctx context.Context, tx Tx, dto *models.UserDataDTO) error
	UpdateSeveral(ctx context.Context, tx Tx, dto []*models.UserDataDTO) error
	UpdateAccount(ctx context.Context, tx Tx, dto *models.UpdateAccountDTO) error
	UpdateMMAndSite(ctx context.Context, tx Tx, dto *models.UserDataDTO) error
	DeleteSeveral(ctx context.Context, tx Tx, ids []uuid.UUID) error
}

func (r *userRepo) LoadPolicy(ctx context.Context) ([]*models.UserRole, error) {
	query := fmt.Sprintf(`SELECT u.id, r.slug, rl.id
		FROM %s u
		JOIN %s ur ON u.id = ur.user_id
		JOIN %s r ON ur.role_id = r.id
		JOIN %s rl ON ur.realm_id = rl.id
		WHERE u.is_active = true AND rl.is_active = true`,
		Tables.Users, Tables.UserRealms, Tables.Roles, Tables.Realms,
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

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.UserData, error) {
	query := fmt.Sprintf(`SELECT u.id, u.username, u.email, u.first_name, u.last_name, u.created_at,
			u.is_active AS user_is_active,
			u.is_system AS user_is_system,
			u.internal_number AS internal_number,
			ur.id AS ur_id, ur.is_active,
			r.id AS role_id, r.name AS role_name, r.description AS role_description, r.level AS role_level,
			r.is_active AS role_is_active, r.is_editable AS role_is_editable, r.slug AS role_slug,
			rl.id AS realm_id, rl.name AS realm_name, rl.description AS realm_description,
			rl.is_active AS realm_is_active,
			ur.created_at AS realm_created_at
		FROM %s u
		LEFT JOIN %s ur ON u.id = ur.user_id
		LEFT JOIN %s r ON ur.role_id = r.id
		LEFT JOIN %s rl ON ur.realm_id = rl.id
		WHERE u.id = $1`,
		Tables.Users, Tables.UserRealms, Tables.Roles, Tables.Realms,
	)

	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to get user by id: %w", err))
	}
	defer rows.Close()

	userRows, err := scanUserRows(rows)
	if err != nil {
		return nil, err
	}
	if len(userRows) == 0 {
		return nil, models.ErrNotFound
	}

	data, err := mapUsersData(userRows)
	if err != nil {
		return nil, err
	}
	return data[0], nil
}

func (r *userRepo) GetByLogin(ctx context.Context, login string) (*models.UserData, error) {
	query := fmt.Sprintf(`SELECT u.id, u.username, u.email, u.first_name, u.last_name, u.created_at,
			u.is_active AS user_is_active,
			u.is_system AS user_is_system,
			u.internal_number AS internal_number,
			ur.id AS ur_id, ur.is_active,
			r.id AS role_id, r.name AS role_name, r.description AS role_description, r.level AS role_level,
			r.is_active AS role_is_active, r.is_editable AS role_is_editable, r.slug AS role_slug,
			rl.id AS realm_id, rl.name AS realm_name, rl.description AS realm_description,
			rl.is_active AS realm_is_active,
			ur.created_at AS realm_created_at
		FROM %s u
		LEFT JOIN %s ur ON u.id = ur.user_id
		LEFT JOIN %s r ON ur.role_id = r.id
		LEFT JOIN %s rl ON ur.realm_id = rl.id
		WHERE u.username = $1 OR u.email = $1`,
		Tables.Users, Tables.UserRealms, Tables.Roles, Tables.Realms,
	)

	rows, err := r.db.Query(ctx, query, login)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to get user by login: %w", err))
	}
	defer rows.Close()

	userRows, err := scanUserRows(rows)
	if err != nil {
		return nil, err
	}
	if len(userRows) == 0 {
		return nil, models.ErrNotFound
	}

	data, err := mapUsersData(userRows)
	if err != nil {
		return nil, err
	}
	return data[0], nil
}

func (r *userRepo) GetByMattermostID(ctx context.Context, mattermostID string) (*models.UserData, error) {
	query := fmt.Sprintf(`SELECT u.id, u.username, u.email, u.first_name, u.last_name, u.created_at,
			u.is_active AS user_is_active,
			u.is_system AS user_is_system,
			u.internal_number AS internal_number,
			ur.id AS ur_id, ur.is_active,
			r.id AS role_id, r.name AS role_name, r.description AS role_description, r.level AS role_level,
			r.is_active AS role_is_active, r.is_editable AS role_is_editable, r.slug AS role_slug,
			rl.id AS realm_id, rl.name AS realm_name, rl.description AS realm_description,
			rl.is_active AS realm_is_active,
			ur.created_at AS realm_created_at
		FROM %s u
		LEFT JOIN %s ur ON u.id = ur.user_id
		LEFT JOIN %s r ON ur.role_id = r.id
		LEFT JOIN %s rl ON ur.realm_id = rl.id
		WHERE u.mattermost_id = $1 AND u.mattermost_id <> ''`,
		Tables.Users, Tables.UserRealms, Tables.Roles, Tables.Realms,
	)

	rows, err := r.db.Query(ctx, query, mattermostID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to get user by mattermost id: %w", err))
	}
	defer rows.Close()

	userRows, err := scanUserRows(rows)
	if err != nil {
		return nil, err
	}
	if len(userRows) == 0 {
		return nil, models.ErrNotFound
	}

	data, err := mapUsersData(userRows)
	if err != nil {
		return nil, err
	}
	return data[0], nil
}

func (r *userRepo) GetAll(ctx context.Context, realmID *uuid.UUID) ([]*models.UserData, error) {
	baseQuery := fmt.Sprintf(`SELECT u.id, u.username, u.email, u.first_name, u.last_name, u.created_at,
			u.is_active AS user_is_active,
			u.is_system AS user_is_system,
			u.internal_number AS internal_number,
			ur.id AS ur_id, ur.is_active,
			r.id AS role_id, r.name AS role_name, r.description AS role_description, r.level AS role_level,
			r.is_active AS role_is_active, r.is_editable AS role_is_editable, r.slug AS role_slug,
			rl.id AS realm_id, rl.name AS realm_name, rl.description AS realm_description,
			rl.is_active AS realm_is_active,
			ur.created_at AS realm_created_at
		FROM %s u
		LEFT JOIN %s ur ON u.id = ur.user_id
		LEFT JOIN %s r ON ur.role_id = r.id
		LEFT JOIN %s rl ON ur.realm_id = rl.id`,
		Tables.Users, Tables.UserRealms, Tables.Roles, Tables.Realms,
	)

	var query string
	var args []interface{}

	if realmID != nil {
		query = baseQuery + ` WHERE ur.realm_id = $1 ORDER BY u.first_name, u.last_name, u.username, rl.name`
		args = []interface{}{*realmID}
	} else {
		query = baseQuery + ` ORDER BY u.first_name, u.last_name, u.username, rl.name`
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to get all users: %w", err))
	}
	defer rows.Close()

	userRows, err := scanUserRows(rows)
	if err != nil {
		return nil, err
	}

	data, err := mapUsersData(userRows)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func scanUserRows(rows pgx.Rows) ([]*pq_models.User, error) {
	var result []*pq_models.User
	for rows.Next() {
		item := &pq_models.User{}
		if err := rows.Scan(
			&item.Id, &item.Username, &item.Email, &item.FirstName, &item.LastName, &item.CreatedAt,
			&item.UserIsActive, &item.UserIsSystem,
			&item.InternalNumber,
			&item.UserRealmId, &item.IsActive,
			&item.RoleId, &item.RoleName, &item.RoleDescription, &item.RoleLevel,
			&item.RoleIsActive, &item.RoleIsEditable, &item.RoleSlug,
			&item.RealmId, &item.RealmName, &item.RealmDescription, &item.RealmIsActive,
			&item.RealmCreatedAt,
		); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return result, nil
}

func mapUsersData(rows []*pq_models.User) ([]*models.UserData, error) {
	result := make([]*models.UserData, 0, 10)
	userIndex := make(map[string]int)

	for _, u := range rows {
		if !u.UserRealmId.Valid {
			if _, ok := userIndex[u.Id]; !ok {
				id, err := uuid.Parse(u.Id)
				if err != nil {
					return nil, fmt.Errorf("invalid user id %q: %w", u.Id, err)
				}
				result = append(result, &models.UserData{
					ID:             id,
					Username:       u.Username,
					Email:          u.Email,
					FirstName:      u.FirstName,
					LastName:       u.LastName,
					IsActive:       u.UserIsActive.Bool,
					IsSystem:       u.UserIsSystem.Bool,
					InternalNumber: u.InternalNumber.String,
					CreatedAt:      u.CreatedAt,
					Realms:         []*models.UserRealm{},
				})
				userIndex[u.Id] = len(result) - 1
			}
			continue
		}

		var role *models.Role
		if u.RoleId.Valid {
			roleID, err := uuid.Parse(u.RoleId.String)
			if err != nil {
				return nil, fmt.Errorf("invalid role id %q: %w", u.RoleId.String, err)
			}
			role = &models.Role{
				ID:          roleID,
				Slug:        u.RoleSlug.String,
				Name:        u.RoleName.String,
				Description: u.RoleDescription.String,
				Level:       int(u.RoleLevel.Int64),
				IsActive:    u.RoleIsActive.Bool,
				IsEditable:  u.RoleIsEditable.Bool,
			}
		}

		var realm *models.Realm
		var realmID uuid.UUID
		if u.RealmId.Valid {
			var err error
			realmID, err = uuid.Parse(u.RealmId.String)
			if err != nil {
				return nil, fmt.Errorf("invalid realm id %q: %w", u.RealmId.String, err)
			}
			realm = &models.Realm{
				ID:          realmID,
				Name:        u.RealmName.String,
				Description: u.RealmDescription.String,
				IsActive:    u.RealmIsActive.Bool,
			}
		}

		userRealmID, err := uuid.Parse(u.UserRealmId.String)
		if err != nil {
			return nil, fmt.Errorf("invalid user realm id %q: %w", u.UserRealmId.String, err)
		}
		userRealm := &models.UserRealm{
			ID:        userRealmID,
			IsActive:  u.IsActive.Bool,
			CreatedAt: u.RealmCreatedAt.Time,
			Realm:     realm,
			Role:      role,
		}
		if u.RealmId.Valid {
			userRealm.RealmID = realmID
		}
		if u.RoleId.Valid {
			roleID, err := uuid.Parse(u.RoleId.String)
			if err != nil {
				return nil, fmt.Errorf("invalid role id %q: %w", u.RoleId.String, err)
			}
			userRealm.RoleID = roleID
		}

		if idx, ok := userIndex[u.Id]; ok {
			result[idx].Realms = append(result[idx].Realms, userRealm)
		} else {
			id, err := uuid.Parse(u.Id)
			if err != nil {
				return nil, fmt.Errorf("invalid user id %q: %w", u.Id, err)
			}
			result = append(result, &models.UserData{
				ID:             id,
				Username:       u.Username,
				Email:          u.Email,
				FirstName:      u.FirstName,
				LastName:       u.LastName,
				IsActive:       u.UserIsActive.Bool,
				IsSystem:       u.UserIsSystem.Bool,
				InternalNumber: u.InternalNumber.String,
				CreatedAt:      u.CreatedAt,
				Realms:         []*models.UserRealm{userRealm},
			})
			userIndex[u.Id] = len(result) - 1
		}
	}

	return result, nil
}

func (r *userRepo) CreateSeveral(ctx context.Context, tx Tx, dto []*models.UserDataDTO) error {
	if len(dto) == 0 {
		return nil
	}

	rows := make([][]interface{}, len(dto))

	for i, v := range dto {
		rows[i] = []interface{}{
			v.ID,
			v.Username,
			v.FirstName,
			v.LastName,
			v.Email,
			v.MattermostID,
			v.SiteID,
			v.IsActive,
			v.IsSystem,
			v.InternalNumber,
		}
	}

	columns := []string{"id", "username", "first_name", "last_name", "email", "mattermost_id", "site_id", "is_active", "is_system", "internal_number"}
	_, err := r.getExec(tx).CopyFrom(
		ctx,
		pgx.Identifier{Tables.Users},
		columns,
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *userRepo) Update(ctx context.Context, tx Tx, dto *models.UserDataDTO) error {
	query := fmt.Sprintf(`UPDATE %s SET username = $1, email = $2, first_name = $3, last_name = $4, is_active = $5, internal_number = $6, updated_at = now()
		WHERE id = $7`,
		Tables.Users,
	)

	_, err := r.getExec(tx).Exec(ctx, query, dto.Username, dto.Email, dto.FirstName, dto.LastName, dto.IsActive, dto.InternalNumber, dto.ID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *userRepo) UpdateSeveral(ctx context.Context, tx Tx, dto []*models.UserDataDTO) error {
	if len(dto) == 0 {
		return nil
	}

	n := len(dto)
	ids := make([]uuid.UUID, n)
	usernames := make([]string, n)
	emails := make([]string, n)
	firstNames := make([]string, n)
	lastNames := make([]string, n)
	isActives := make([]bool, n)
	internalNumbers := make([]string, n)

	for i, v := range dto {
		ids[i] = v.ID
		usernames[i] = v.Username
		emails[i] = v.Email
		firstNames[i] = v.FirstName
		lastNames[i] = v.LastName
		isActives[i] = v.IsActive
		internalNumbers[i] = v.InternalNumber
	}

	query := fmt.Sprintf(`
		UPDATE %s AS t
		SET
			username = s.username,
			email = s.email,
			first_name = s.first_name,
			last_name = s.last_name,
			is_active = s.is_active,
			internal_number = s.internal_number
		FROM (
			SELECT * FROM UNNEST(
				$1::text[],
				$2::text[],
				$3::text[],
				$4::text[],
				$5::bool[],
				$6::text[],
				$7::text[]
			) AS s(username, email, first_name, last_name, is_active, id, internal_number)
		) AS s
		WHERE t.id = s.id`,
		Tables.Users,
	)

	_, err := r.getExec(tx).Exec(ctx, query, usernames, emails, firstNames, lastNames, isActives, ids, internalNumbers)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute bulk update: %w", err))
	}
	return nil
}

func (r *userRepo) UpdateAccount(ctx context.Context, tx Tx, dto *models.UpdateAccountDTO) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET is_active = $1,
		    mattermost_id = NULLIF($2::text, ''),
		    internal_number = $3,
		    updated_at = now()
		WHERE id = $4`,
		Tables.Users,
	)

	// Передаем указатель напрямую. Драйвер сам преобразует nil в SQL NULL, а *string в text.
	_, err := r.getExec(tx).Exec(ctx, query, dto.IsActive, dto.MattermostID, dto.InternalNumber, dto.ID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

// UpdateMMAndSite обновляет mattermost_id и site_id пользователя. Переданные
// nil-указатели не трогают соответствующие колонки (COALESCE к текущему значению).
func (r *userRepo) UpdateMMAndSite(ctx context.Context, tx Tx, dto *models.UserDataDTO) error {
	query := fmt.Sprintf(`
		UPDATE %s SET
			mattermost_id = COALESCE($2, mattermost_id),
			site_id = COALESCE($3, site_id),
			updated_at = now()
		WHERE id = $1`,
		Tables.Users,
	)
	_, err := r.getExec(tx).Exec(ctx, query, dto.ID, dto.MattermostID, dto.SiteID)
	return MapError(fmt.Errorf("failed to update mattermost id/site: %w", err))
}

func (r *userRepo) DeleteSeveral(ctx context.Context, tx Tx, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = ANY($1)`, Tables.Users)

	if _, err := r.getExec(tx).Exec(ctx, query, ids); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
