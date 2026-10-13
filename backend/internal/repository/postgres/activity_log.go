package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type activityRepository struct {
	db *pgxpool.Pool
	Transaction
}

func NewActivityRepo(db *pgxpool.Pool, tr Transaction) *activityRepository {
	return &activityRepository{
		db:          db,
		Transaction: tr,
	}
}

type ActivityLog interface {
	Get(ctx context.Context, req *models.GetLogsDTO) ([]*models.ActivityLog, error)
	Create(ctx context.Context, tx Tx, dto []*models.ActivityLogDTO) error
	// GetRealmByEntity возвращает реалм сущности журнала по entity_type/entity_id:
	// заявки — из tickets, подзадачи — через subtasks.ticket_id. ok=false — тип не
	// резолвится или сущности нет. Запрос идёт через ту же транзакцию: строки
	// незакоммиченной заявки видит только tx-запрос, а не пул.
	GetRealmByEntity(ctx context.Context, tx Tx, entityType string, entityID uuid.UUID) (uuid.UUID, string, bool, error)
}

type nullableActivityActor struct {
	ActorID             *uuid.UUID
	ActorUsername       *string
	ActorFirstName      *string
	ActorLastName       *string
	ActorInternalNumber *string
}

func (a *nullableActivityActor) assign(log *models.ActivityLog) {
	if a.ActorID != nil {
		log.Actor = &models.UserShort{
			ID:             *a.ActorID,
			Username:       *a.ActorUsername,
			FirstName:      *a.ActorFirstName,
			LastName:       *a.ActorLastName,
			InternalNumber: *a.ActorInternalNumber,
		}
	}
}

func (r *activityRepository) Get(ctx context.Context, req *models.GetLogsDTO) ([]*models.ActivityLog, error) {
	where := ""
	args := make([]any, 0)

	if req.ParentID != nil {
		where = "WHERE al.entity_id = $1 OR al.parent_id = $1"
		args = append(args, *req.ParentID)
	} else if req.EntityID != nil {
		where = "WHERE al.entity_id = $1"
		args = append(args, *req.EntityID)
		if req.EntityType != nil {
			where += fmt.Sprintf(" AND al.entity_type = $%d", len(args)+1)
			args = append(args, *req.EntityType)
		}
	}

	if req.RealmID != nil {
		if where == "" {
			where = "WHERE al.realm_id = $1"
			args = append(args, *req.RealmID)
		} else {
			where += fmt.Sprintf(" AND al.realm_id = $%d", len(args)+1)
			args = append(args, *req.RealmID)
		}
	}

	query := fmt.Sprintf(`SELECT al.id, al.action, al.changed_by, al.changed_by_name,
		al.entity_type, al.entity_id, al.entity, al.parent_id,
		al.realm_id, al.realm_name, al.old_value, al.new_value, al.created_at,
		u.id, u.username, u.first_name, u.last_name, u.internal_number
		FROM %s al
		LEFT JOIN %s u ON u.id = al.changed_by
		%s ORDER BY al.created_at DESC`,
		Tables.ActivityLog, Tables.Users, where,
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.ActivityLog{}
	for rows.Next() {
		item := &models.ActivityLog{}
		actor := nullableActivityActor{}
		if err := rows.Scan(
			&item.ID, &item.Action, &item.ChangedBy, &item.ChangedByName,
			&item.EntityType, &item.EntityID, &item.Entity, &item.ParentID,
			&item.RealmID, &item.RealmName,
			&item.OldValues, &item.NewValues, &item.CreatedAt,
			&actor.ActorID, &actor.ActorUsername, &actor.ActorFirstName,
			&actor.ActorLastName, &actor.ActorInternalNumber,
		); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		actor.assign(item)
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

// GetRealmByEntity резолвит реалм сущности журнала для записей, у которых
// RealmID не заполнен (старые/обходные пути). Без него выборки GET /activity-log
// с фильтром по авторизованному реалму никогда не вернут запись — «История»
// пустая. Запрос через getExec(tx): внутри транзакции создания заявки строка
// tickets ещё не закоммичена и видна только tx-запросу.
func (r *activityRepository) GetRealmByEntity(ctx context.Context, tx Tx, entityType string, entityID uuid.UUID) (uuid.UUID, string, bool, error) {
	var query string
	switch entityType {
	case models.EntityTypeTicket:
		query = fmt.Sprintf(`SELECT t.realm_id, COALESCE(r.name, '')
			FROM %s t
			LEFT JOIN %s r ON r.id = t.realm_id
			WHERE t.id = $1`, Tables.Tickets, Tables.Realms)
	case models.EntityTypeSubtask:
		query = fmt.Sprintf(`SELECT t.realm_id, COALESCE(r.name, '')
			FROM %s s
			JOIN %s t ON t.id = s.ticket_id
			LEFT JOIN %s r ON r.id = t.realm_id
			WHERE s.id = $1`, Tables.Subtasks, Tables.Tickets, Tables.Realms)
	default:
		return uuid.Nil, "", false, nil
	}

	var realmID *uuid.UUID
	var realmName string
	err := r.getExec(tx).QueryRow(ctx, query, entityID).Scan(&realmID, &realmName)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", false, nil
	}
	if err != nil {
		return uuid.Nil, "", false, MapError(fmt.Errorf("get realm by entity: %w", err))
	}
	if realmID == nil {
		return uuid.Nil, "", false, nil
	}
	return *realmID, realmName, true, nil
}

func (r *activityRepository) Create(ctx context.Context, tx Tx, dto []*models.ActivityLogDTO) error {
	if len(dto) == 0 {
		return nil
	}

	rows := make([][]any, len(dto))

	for i, v := range dto {
		if v.ID == uuid.Nil {
			v.ID = uuid.New()
		}

		rows[i] = []any{
			v.ID,
			v.Action,
			v.ChangedBy,
			v.ChangedByName,
			v.EntityType,
			v.EntityID,
			v.Entity,
			v.ParentID,
			v.RealmID,
			v.RealmName,
			v.OldValues,
			v.NewValues,
		}
	}

	columns := []string{"id", "action", "changed_by", "changed_by_name", "entity_type", "entity_id", "entity", "parent_id", "realm_id", "realm_name", "old_value", "new_value"}
	_, err := r.getExec(tx).CopyFrom(
		ctx,
		pgx.Identifier{Tables.ActivityLog},
		columns,
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		return MapError(fmt.Errorf("failed to execute query. error: %w", err))
	}
	return nil
}
