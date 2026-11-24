package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

const duplicateChainLimit = 50

// GetDuplicateChain поднимается от start по duplicate_of до корня цепочки.
// Возвращает узлы в порядке start → … → root. Цикл и слишком длинная цепочка
// отсекаются здесь, чтобы сервис не зациклился и не блокировал всю базу.
func (r *TicketRepo) GetDuplicateChain(ctx context.Context, tx Tx, start uuid.UUID) ([]DuplicateNode, error) {
	exec := r.getExec(tx)
	chain := make([]DuplicateNode, 0, 4)
	seen := map[uuid.UUID]struct{}{start: {}}
	current := start

	for {
		var node DuplicateNode
		err := exec.QueryRow(ctx, fmt.Sprintf(`SELECT id, duplicate_of, status FROM %s WHERE id = $1`, Tables.Tickets), current).
			Scan(&node.ID, &node.DuplicateOf, &node.Status)
		if err != nil {
			return nil, MapError(fmt.Errorf("failed to load duplicate chain node: %w", err))
		}
		chain = append(chain, node)
		if node.DuplicateOf == nil {
			return chain, nil
		}
		if len(chain) >= duplicateChainLimit {
			return nil, models.ErrDuplicateChainTooDeep
		}
		if _, ok := seen[*node.DuplicateOf]; ok {
			return nil, models.ErrDuplicateCycle
		}
		seen[*node.DuplicateOf] = struct{}{}
		current = *node.DuplicateOf
	}
}

// GetDuplicateDescendants возвращает всех потомков указанных узлов по дублям
// (не включая сами корни). Используется перед блокировкой: при смене основной
// заявки нужно зафиксировать всю подветку.
func (r *TicketRepo) GetDuplicateDescendants(ctx context.Context, tx Tx, roots []uuid.UUID) ([]uuid.UUID, error) {
	if len(roots) == 0 {
		return []uuid.UUID{}, nil
	}
	query := fmt.Sprintf(`WITH RECURSIVE tree AS (
			SELECT id, 0 AS depth FROM %s WHERE id = ANY($1::uuid[])
			UNION ALL
			SELECT t.id, tree.depth + 1 FROM %s t
				JOIN tree ON t.duplicate_of = tree.id
			WHERE tree.depth < $2
		)
		SELECT DISTINCT id FROM tree`, Tables.Tickets, Tables.Tickets)

	rows, err := r.getExec(tx).Query(ctx, query, roots, duplicateChainLimit)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to collect duplicate descendants: %w", err))
	}
	defer rows.Close()

	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, MapError(fmt.Errorf("failed to scan duplicate descendant: %w", err))
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("duplicate descendants rows error: %w", err))
	}
	return ids, nil
}

// LockTickets блокирует заявки в детерминированном порядке — concurrent link
// двух дублей в одну основную не должен перемешать цепочки.
func (r *TicketRepo) LockTickets(ctx context.Context, tx Tx, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	query := fmt.Sprintf(`SELECT id FROM %s WHERE id = ANY($1::uuid[]) ORDER BY id FOR UPDATE`, Tables.Tickets)

	rows, err := r.getExec(tx).Query(ctx, query, ids)
	if err != nil {
		return MapError(fmt.Errorf("failed to lock tickets: %w", err))
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return MapError(fmt.Errorf("failed to scan locked ticket: %w", err))
		}
	}
	if err := rows.Err(); err != nil {
		return MapError(fmt.Errorf("locked tickets rows error: %w", err))
	}
	return nil
}

// SetDuplicate закрывает дубль и запоминает его прежний статус. Закрытие здесь
// осознанно идёт мимо ErrCloseRequiresResolved: дубль может быть открытым.
func (r *TicketRepo) SetDuplicate(ctx context.Context, tx Tx, id, primaryID uuid.UUID, previousStatus models.TicketStatus) error {
	// duplicate_of_set_at — момент 현재 связи (не дата самой заявки): статистика
	// по нему решает, исключать дубль из периода или оставить в историческом.
	query := fmt.Sprintf(`UPDATE %s SET duplicate_of = $2, previous_status = $3, status = 'closed',
		duplicate_of_set_at = NOW(), closed_at = COALESCE(closed_at, NOW()), updated_at = NOW() WHERE id = $1`, Tables.Tickets)

	if _, err := r.getExec(tx).Exec(ctx, query, id, primaryID, previousStatus); err != nil {
		return MapError(fmt.Errorf("failed to link duplicate: %w", err))
	}
	return nil
}

// ClearDuplicate возвращает дублю прежний статус; resolved_at не трогается —
// если заявку уже решили, решение остаётся в истории.
func (r *TicketRepo) ClearDuplicate(ctx context.Context, tx Tx, id uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET duplicate_of = NULL, previous_status = NULL, duplicate_of_set_at = NULL,
		status = COALESCE(previous_status, 'open'), closed_at = NULL, updated_at = NOW() WHERE id = $1`, Tables.Tickets)

	if _, err := r.getExec(tx).Exec(ctx, query, id); err != nil {
		return MapError(fmt.Errorf("failed to unlink duplicate: %w", err))
	}
	return nil
}

// RepointDuplicates переводит всех direct-потомков old на new. Статусы и
// previous_status детей не меняются: они остаются закрытыми дублями.
func (r *TicketRepo) RepointDuplicates(ctx context.Context, tx Tx, from, to uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET duplicate_of = $2, updated_at = NOW() WHERE duplicate_of = $1`, Tables.Tickets)

	if _, err := r.getExec(tx).Exec(ctx, query, from, to); err != nil {
		return MapError(fmt.Errorf("failed to repoint duplicates: %w", err))
	}
	return nil
}

// RepointOne меняет основную только у одного дубля, сохраняя его прежний статус.
func (r *TicketRepo) RepointOne(ctx context.Context, tx Tx, id, primaryID uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET duplicate_of = $2, updated_at = NOW() WHERE id = $1`, Tables.Tickets)

	if _, err := r.getExec(tx).Exec(ctx, query, id, primaryID); err != nil {
		return MapError(fmt.Errorf("failed to repoint duplicate: %w", err))
	}
	return nil
}

// MoveDuplicateNotifiers переносит наблюдателей, появившихся из дублей old,
// на новую основную — иначе они потеряли бы доступ к заявке после flatten.
func (r *TicketRepo) MoveDuplicateNotifiers(ctx context.Context, tx Tx, from, to uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET primary_ticket_id = $2 WHERE primary_ticket_id = $1`, Tables.TicketDuplicateNotifiers)

	if _, err := r.getExec(tx).Exec(ctx, query, from, to); err != nil {
		return MapError(fmt.Errorf("failed to move duplicate notifiers: %w", err))
	}
	return nil
}

// MoveDuplicateNotifier переносит наблюдателей только одного дубля — при
// перенаправлении отдельной заявки трогать строки остальных дублей нельзя.
func (r *TicketRepo) MoveDuplicateNotifier(ctx context.Context, tx Tx, from, to, duplicateID uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET primary_ticket_id = $2
		WHERE primary_ticket_id = $1 AND duplicate_ticket_id = $3`, Tables.TicketDuplicateNotifiers)

	if _, err := r.getExec(tx).Exec(ctx, query, from, to, duplicateID); err != nil {
		return MapError(fmt.Errorf("failed to move duplicate notifier: %w", err))
	}
	return nil
}

// DeleteDuplicateNotifiers убирает автоматических наблюдателей при отвязке
// дубля: после восстановления заявки её автор/заказчик снова владеют ею напрямую
// и не должны сохранять доступ к чужой основной через старую связь.
func (r *TicketRepo) DeleteDuplicateNotifiers(ctx context.Context, tx Tx, duplicateID uuid.UUID) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE duplicate_ticket_id = $1`, Tables.TicketDuplicateNotifiers)

	if _, err := r.getExec(tx).Exec(ctx, query, duplicateID); err != nil {
		return MapError(fmt.Errorf("failed to delete duplicate notifiers: %w", err))
	}
	return nil
}

// HasDuplicateReferences сообщает, есть ли заявки, ссылающиеся на данную как на
// основную. Удаление такой заявки уперлось бы в ON DELETE RESTRICT. Вызывается
// в транзакции сразу после блокировки родителя FOR UPDATE: иначе параллельный
// link успел бы добавить ссылку между проверкой и DELETE, и вместо TK015
// вылетела бы сырая FK-ошибка.
func (r *TicketRepo) HasDuplicateReferences(ctx context.Context, tx Tx, ticketID uuid.UUID) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE duplicate_of = $1)`, Tables.Tickets)

	var exists bool
	if err := r.getExec(tx).QueryRow(ctx, query, ticketID).Scan(&exists); err != nil {
		return false, MapError(fmt.Errorf("failed to check duplicate references: %w", err))
	}
	return exists, nil
}

// AddDuplicateNotifier добавляет автору/владельцу дубля read-доступ к основной.
// Повторная привязка не создаёт дубликат строки.
func (r *TicketRepo) AddDuplicateNotifier(ctx context.Context, tx Tx, primaryID, userID, duplicateID, actorID uuid.UUID) error {
	query := fmt.Sprintf(`INSERT INTO %s (primary_ticket_id, user_id, duplicate_ticket_id, added_by)
		VALUES ($1, $2, $3, $4) ON CONFLICT (primary_ticket_id, user_id, duplicate_ticket_id) DO NOTHING`,
		Tables.TicketDuplicateNotifiers)

	if _, err := r.getExec(tx).Exec(ctx, query, primaryID, userID, duplicateID, actorID); err != nil {
		return MapError(fmt.Errorf("failed to add duplicate notifier: %w", err))
	}
	return nil
}

// GetIDsWithDuplicatePrimaries возвращает из переданных пользователей тех, кто
// является автором или владельцем основной заявки, на которую ссылаются дубли.
// Keycloak-синк использует это, чтобы не удалять такого пользователя вместе с
// его заявками (duplicates ссылаются на основную через ON DELETE RESTRICT).
func (r *TicketRepo) GetIDsWithDuplicatePrimaries(ctx context.Context, userIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(userIDs) == 0 {
		return []uuid.UUID{}, nil
	}
	query := fmt.Sprintf(`SELECT DISTINCT u.id FROM %s u
		WHERE u.id = ANY($1::uuid[]) AND (
			EXISTS (SELECT 1 FROM %s p JOIN %s d ON d.duplicate_of = p.id WHERE p.creator_id = u.id)
			OR EXISTS (SELECT 1 FROM %s p JOIN %s d ON d.duplicate_of = p.id WHERE p.owner_id = u.id)
		)`, Tables.Users, Tables.Tickets, Tables.Tickets, Tables.Tickets, Tables.Tickets)

	rows, err := r.db.Query(ctx, query, userIDs)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to check duplicate primaries: %w", err))
	}
	defer rows.Close()

	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, MapError(fmt.Errorf("failed to scan duplicate primary owner: %w", err))
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("duplicate primary owners rows error: %w", err))
	}
	return ids, nil
}

// CountDuplicateActivity считает комментарии и вложения перечисленных заявок
// одним запросом. Вложения считаем вместе с файлами из комментариев: у тикета
// они лежат в той же таблице с entity_type = 'ticket' и comment_id != NULL.
func (r *TicketRepo) CountDuplicateActivity(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]DuplicateActivityCounts, error) {
	counts := make(map[uuid.UUID]DuplicateActivityCounts, len(ids))
	if len(ids) == 0 {
		return counts, nil
	}

	query := fmt.Sprintf(`SELECT t.id,
		(SELECT COUNT(*) FROM %s c WHERE c.ticket_id = t.id),
		(SELECT COUNT(*) FROM %s a WHERE a.entity_type = 'ticket' AND a.entity_id = t.id)
		FROM %s t WHERE t.id = ANY($1::uuid[])`,
		Tables.Comments, Tables.Attachments, Tables.Tickets,
	)
	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to count duplicate activity: %w", err))
	}
	defer rows.Close()

	for rows.Next() {
		var ticketID uuid.UUID
		var comments, attachments int64
		if err := rows.Scan(&ticketID, &comments, &attachments); err != nil {
			return nil, MapError(fmt.Errorf("failed to scan duplicate activity counts: %w", err))
		}
		counts[ticketID] = DuplicateActivityCounts{Comments: int(comments), Attachments: int(attachments)}
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("duplicate activity counts rows error: %w", err))
	}
	return counts, nil
}
