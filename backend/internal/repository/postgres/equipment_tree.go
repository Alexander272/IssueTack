package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

func (r *EquipmentRepo) GetChildren(ctx context.Context, realmID, parentID uuid.UUID) ([]*models.Equipment, error) {
	query := equipmentSelectBase() + ` WHERE e.parent_id = $1 AND e.realm_id = $2 ORDER BY e.name`

	rows, err := r.db.Query(ctx, query, parentID, realmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.Equipment{}
	for rows.Next() {
		item, _, err := scanEquipmentRow(rows)
		if err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

// GetTree возвращает один уровень дерева оборудования реалма. Дерево грузится
// лениво: при ParentID == nil это корни (нет физического родителя в реалме и нет
// открытого членства в комплекте), иначе — непосредственные дети узла: физический
// состав (parent_id) плюс текущие члены логического комплекта (kit_compositions).
// У члена комплекта с физическим родителем приоритет у родителя — как в прежней
// сборке дерева. Узлы несут HasChildren/ChildCount, а не только своё имя.
func (r *EquipmentRepo) GetTree(ctx context.Context, dto *models.GetEquipmentTreeDTO) ([]*models.EquipmentTreeNode, int, error) {
	w := &whereBuilder{}
	eq(w, "e.realm_id", &dto.RealmID)

	if dto.ParentID == nil {
		// $1 — realmID (первый аргумент eq). Статус и IsKit сужают только корни:
		// дети раскрытого узла показываются целиком (как было на клиенте).
		eq(w, "e.status", dto.Status)
		eq(w, "e.is_kit", dto.IsKit)
		w.add(`NOT EXISTS (SELECT 1 FROM ` + Tables.Equipment + ` p WHERE p.id = e.parent_id AND p.realm_id = $1)`)
		w.add(`NOT EXISTS (SELECT 1 FROM ` + Tables.KitCompositions + ` kc WHERE kc.component_id = e.id AND kc.ended_at IS NULL)`)
	} else {
		w.idx++
		parent := w.idx
		w.args = append(w.args, *dto.ParentID)
		w.add(fmt.Sprintf(`(e.parent_id = $%d OR (EXISTS (SELECT 1 FROM %s kc WHERE kc.component_id = e.id AND kc.kit_id = $%d AND kc.ended_at IS NULL)
			AND NOT EXISTS (SELECT 1 FROM %s p WHERE p.id = e.parent_id AND p.realm_id = $1)))`,
			parent, Tables.KitCompositions, parent, Tables.Equipment))
	}

	query := equipmentSelectBase() + " WHERE " + strings.Join(w.clauses, " AND ")
	query += " ORDER BY e.name"

	if dto.Limit > 0 {
		w.idx++
		query += fmt.Sprintf(" LIMIT $%d", w.idx)
		w.args = append(w.args, dto.Limit)
	}
	if dto.Offset > 0 {
		w.idx++
		query += fmt.Sprintf(" OFFSET $%d", w.idx)
		w.args = append(w.args, dto.Offset)
	}

	rows, err := r.db.Query(ctx, query, w.args...)
	if err != nil {
		return nil, 0, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.Equipment{}
	total := 0
	for rows.Next() {
		item, rowTotal, err := scanEquipmentRow(rows)
		if err != nil {
			return nil, 0, MapError(fmt.Errorf("scan row error: %w", err))
		}
		total = rowTotal
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	if len(data) == 0 {
		return []*models.EquipmentTreeNode{}, 0, nil
	}

	ids := make([]uuid.UUID, len(data))
	for i, item := range data {
		ids[i] = item.ID
	}
	childCounts, err := r.getChildCounts(ctx, dto.RealmID, ids)
	if err != nil {
		return nil, 0, err
	}

	nodes := make([]*models.EquipmentTreeNode, 0, len(data))
	for _, item := range data {
		count := childCounts[item.ID]
		nodes = append(nodes, &models.EquipmentTreeNode{
			Equipment:   item,
			Children:    []*models.EquipmentTreeNode{},
			HasChildren: count > 0,
			ChildCount:  count,
		})
	}
	return nodes, total, nil
}

// getChildCounts считает детей узлов одного уровня: физический состав (parent_id)
// и текущих членов комплекта (kit_compositions). Член комплекта, у которого есть
// физический родитель в реалме, не считается — он показывается внутри родителя.
func (r *EquipmentRepo) getChildCounts(ctx context.Context, realmID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]int, error) {
	counts := make(map[uuid.UUID]int, len(ids))
	if len(ids) == 0 {
		return counts, nil
	}

	physical := fmt.Sprintf(`SELECT c.parent_id, COUNT(*)::int FROM %s c
		WHERE c.realm_id = $2 AND c.parent_id = ANY($1) GROUP BY c.parent_id`, Tables.Equipment)
	members := fmt.Sprintf(`SELECT kc.kit_id, COUNT(*)::int FROM %s kc
		JOIN %s c ON c.id = kc.component_id AND c.realm_id = $2
		WHERE kc.kit_id = ANY($1) AND kc.ended_at IS NULL
			AND NOT EXISTS (SELECT 1 FROM %s p WHERE p.id = c.parent_id AND p.realm_id = $2)
		GROUP BY kc.kit_id`, Tables.KitCompositions, Tables.Equipment, Tables.Equipment)

	for _, query := range []string{physical, members} {
		rows, err := r.db.Query(ctx, query, ids, realmID)
		if err != nil {
			return nil, MapError(fmt.Errorf("failed to count equipment children: %w", err))
		}
		for rows.Next() {
			var id uuid.UUID
			var count int
			if err := rows.Scan(&id, &count); err != nil {
				rows.Close()
				return nil, MapError(fmt.Errorf("scan row error: %w", err))
			}
			counts[id] += count
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
		}
		rows.Close()
	}
	return counts, nil
}

// GetAncestors возвращает цепочку предков узла (родитель, родитель родителя, …).
func (r *EquipmentRepo) GetAncestors(ctx context.Context, realmID, id uuid.UUID) ([]uuid.UUID, error) {
	query := fmt.Sprintf(`
		WITH RECURSIVE chain AS (
			SELECT parent_id FROM %s WHERE id = $2 AND realm_id = $1
			UNION ALL
			SELECT e.parent_id FROM %s e JOIN chain c ON e.id = c.parent_id WHERE e.realm_id = $1
		)
		SELECT parent_id FROM chain WHERE parent_id IS NOT NULL`, Tables.Equipment, Tables.Equipment)

	rows, err := r.db.Query(ctx, query, realmID, id)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []uuid.UUID{}
	for rows.Next() {
		var pid uuid.UUID
		if err := rows.Scan(&pid); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, pid)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}
