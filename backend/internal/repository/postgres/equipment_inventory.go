package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

// NextInventoryNumber выдаёт следующее значение счётчика инвентарных номеров
// реалма для серии. Тот же приём, что у ticket_counters: вставка-или-инкремент
// одной командой, поэтому параллельные создания не получат одно значение, а
// откат транзакции откатит и увеличение — номер не «сгорает».
func (r *EquipmentRepo) NextInventoryNumber(ctx context.Context, tx Tx, realmID uuid.UUID, series models.EquipmentInventorySeries) (int, error) {
	query := fmt.Sprintf(`INSERT INTO %s (realm_id, series, last_value) VALUES ($1, $2, 1)
		ON CONFLICT (realm_id, series) DO UPDATE SET last_value = %s.last_value + 1
		RETURNING last_value`,
		Tables.EquipmentInventoryCounters, Tables.EquipmentInventoryCounters)

	var seq int
	if err := r.getExec(tx).QueryRow(ctx, query, realmID, series).Scan(&seq); err != nil {
		return 0, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return seq, nil
}

// InventoryNumberExists проверяет, занят ли номер в реалме. Номер уникален в
// пределах реалма (uq_equipment_realm_inventory); пустые значения индекс не
// учитывает, поэтому проверяются только непустые номера.
func (r *EquipmentRepo) InventoryNumberExists(ctx context.Context, tx Tx, realmID uuid.UUID, number string) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM %s WHERE realm_id = $1 AND inventory_number = $2)`, Tables.Equipment)

	var exists bool
	if err := r.getExec(tx).QueryRow(ctx, query, realmID, number).Scan(&exists); err != nil {
		return false, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return exists, nil
}
