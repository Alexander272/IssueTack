package migrations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upBackfillActivityLogValues, downBackfillActivityLogValues)
}

// activityLogUpdate — одна изменённая строка журнала для пакетного апдейта.
type activityLogUpdate struct {
	id     string
	oldVal []byte
	newVal []byte
}

// Ключи снапшота журнала активности, значения которых нужно привести к тому же виду,
// что пишет рантайм (services/activity_values.go, models.formatFieldValue).
//
// Время: closed / due_date_changed.
var activityTimeKeys = map[string]struct{}{
	"closed":           {},
	"due_date_changed": {},
}

// Пользовательские ключи (id → ФИО).
var activityUserKeys = map[string]struct{}{
	"assigned":        {},
	"assign_changed":  {},
	"manager_changed": {},
	"owner_changed":   {},
	"assignee":        {},
}

// Справочные ключи (id → имя): группа, площадка, категория.
var (
	activityGroupKeys = map[string]struct{}{
		"group_changed":  {},
		"group_assigned": {},
	}
	activitySiteKeys = map[string]struct{}{
		"site_changed": {},
	}
	activityCategoryKeys = map[string]struct{}{
		"category_changed": {},
	}
)

// upBackfillActivityLogValues приводит исторические значения журнала активности к
// контракту вкладки «История» (frontend ActivityTimeline):
//
//   - время (`closed`, `due_date_changed`): сырой Go time.Time.String()
//     ("2026-10-12 14:45:55.709614141 +0500 +05 m=+9883.256736244") → RFC3339 UTC,
//     а пустое/<nil>/nil → "none";
//   - id (`assigned`/`assign_changed`/`manager_changed`/`owner_changed`/`assignee`,
//     `group_changed`/`group_assigned`, `site_changed`, `category_changed`): сырой
//     uuid → человекочитаемая подпись (ФИО / имя справочника).
//
// Идемпотентно: значения, уже приведённые к контракту (RFC3339, имена, коды, "none"),
// не трогаются. Резолв id — best-effort: удалённая сущность или пустое имя оставляют
// сырой id, как и в рантайме.
func upBackfillActivityLogValues(ctx context.Context, tx *sql.Tx) error {
	h := &activityHumanizer{}

	var err error
	if h.users, err = loadActivityUserLabels(ctx, tx); err != nil {
		return fmt.Errorf("failed to load users: %w", err)
	}
	if h.groups, err = loadActivityEntityNames(ctx, tx, "groups"); err != nil {
		return fmt.Errorf("failed to load groups: %w", err)
	}
	if h.sites, err = loadActivityEntityNames(ctx, tx, "sites"); err != nil {
		return fmt.Errorf("failed to load sites: %w", err)
	}
	if h.categories, err = loadActivityEntityNames(ctx, tx, "categories"); err != nil {
		return fmt.Errorf("failed to load categories: %w", err)
	}

	rows, err := tx.QueryContext(ctx, `SELECT id, old_value, new_value FROM public.activity_logs`)
	if err != nil {
		return fmt.Errorf("failed to select activity logs: %w", err)
	}

	var updates []activityLogUpdate

	for rows.Next() {
		var (
			id     string
			oldRaw []byte
			newRaw []byte
		)
		if err := rows.Scan(&id, &oldRaw, &newRaw); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan activity log: %w", err)
		}

		oldVal, oldChanged, err := humanizeActivityValues(oldRaw, h)
		if err != nil {
			rows.Close()
			return fmt.Errorf("failed to normalize old_value of %s: %w", id, err)
		}
		newVal, newChanged, err := humanizeActivityValues(newRaw, h)
		if err != nil {
			rows.Close()
			return fmt.Errorf("failed to normalize new_value of %s: %w", id, err)
		}
		if !oldChanged && !newChanged {
			continue
		}

		updates = append(updates, activityLogUpdate{id: id, oldVal: oldVal, newVal: newVal})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("failed to iterate activity logs: %w", err)
	}
	rows.Close()

	if len(updates) == 0 {
		return nil
	}

	return applyActivityLogUpdates(ctx, tx, updates)
}

// downBackfillActivityLogValues — бэкфилл необратим: возвращать в историю сырые время
// и uuid смысла нет.
func downBackfillActivityLogValues(ctx context.Context, tx *sql.Tx) error {
	return nil
}

// activityHumanizer резолвит id из журнала в человекочитаемые подписи по справочникам,
// загруженным в память (id → подпись).
type activityHumanizer struct {
	users      map[string]string
	groups     map[string]string
	sites      map[string]string
	categories map[string]string
}

// normalizeValue приводит одно значение журнала по его ключу. Второй результат —
// известен ли ключ (знакомые ключи нормализуются всегда, даже если значение не
// изменилось, чтобы отличать «нет ключа» от «ключ есть, но не строковый»).
func (h *activityHumanizer) normalizeValue(key, val string) (string, bool) {
	if _, ok := activityTimeKeys[key]; ok {
		return normalizeActivityTime(val), true
	}
	if _, ok := activityUserKeys[key]; ok {
		return resolveActivityLabel(h.users, val), true
	}
	if _, ok := activityGroupKeys[key]; ok {
		return resolveActivityLabel(h.groups, val), true
	}
	if _, ok := activitySiteKeys[key]; ok {
		return resolveActivityLabel(h.sites, val), true
	}
	if _, ok := activityCategoryKeys[key]; ok {
		return resolveActivityLabel(h.categories, val), true
	}
	return val, false
}

// humanizeActivityValues нормализует JSONB-снапшот журнала. Возвращает исходные байты
// и false, если менять нечего (NULL, не объект, нет знакомых ключей или значения уже
// контрактные).
func humanizeActivityValues(raw []byte, h *activityHumanizer) ([]byte, bool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return raw, false, nil
	}

	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, false, err
	}

	changed := false
	for key, rawVal := range values {
		var val string
		if err := json.Unmarshal(rawVal, &val); err != nil {
			continue // не строка — не трогаем
		}
		normalized, known := h.normalizeValue(key, val)
		if !known || normalized == val {
			continue
		}
		encoded, err := json.Marshal(normalized)
		if err != nil {
			return nil, false, err
		}
		values[key] = encoded
		changed = true
	}
	if !changed {
		return raw, false, nil
	}

	out, err := json.Marshal(values)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

// normalizeActivityTime приводит значение времени к RFC3339 (UTC). Пустое, "none" и
// легаси-артефакты типизированного nil ("<nil>"/"nil") становятся "none"; уже
// контрактный RFC3339 не трогается; неизвестный формат остаётся как есть.
func normalizeActivityTime(raw string) string {
	val := strings.TrimSpace(raw)
	if val == "" || val == "none" || val == "<nil>" || val == "nil" {
		return "none"
	}
	if _, err := time.Parse(time.RFC3339, val); err == nil {
		return raw // уже контракт
	}
	if t, ok := parseLegacyGoTime(val); ok {
		return t.UTC().Format(time.RFC3339)
	}
	return raw
}

// parseLegacyGoTime разбирает сырой Go-формат time.Time.String():
// "2006-01-02 15:04:05[.frac] -0700 MST" с необязательным суффиксом монотонных часов
// (" m=+123.456"). Оба варианта — с дробной частью и без — разбираются time.Parse.
func parseLegacyGoTime(val string) (time.Time, bool) {
	if i := strings.Index(val, " m="); i >= 0 {
		val = val[:i]
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
	} {
		if t, err := time.Parse(layout, val); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// resolveActivityLabel заменяет uuid на подпись из справочника; не-идентификатор,
// отсутствие записи или пустая подпись — исходное значение (как в рантайме).
func resolveActivityLabel(labels map[string]string, raw string) string {
	id, err := uuid.Parse(raw)
	if err != nil {
		return raw
	}
	if label, ok := labels[id.String()]; ok && label != "" {
		return label
	}
	return raw
}

// loadActivityUserLabels собирает id → «Фамилия Имя (username)» (см. services.logPersonName).
func loadActivityUserLabels(ctx context.Context, tx *sql.Tx) (map[string]string, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT id, coalesce(username, ''), coalesce(first_name, ''), coalesce(last_name, '') FROM public.users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	labels := make(map[string]string)
	for rows.Next() {
		var id, username, firstName, lastName string
		if err := rows.Scan(&id, &username, &firstName, &lastName); err != nil {
			return nil, err
		}
		labels[id] = activityPersonLabel(firstName, lastName, username, id)
	}
	return labels, rows.Err()
}

// loadActivityEntityNames собирает id → имя справочной таблицы. Имя таблицы приходит
// только из литералов ниже, не из пользовательских данных.
func loadActivityEntityNames(ctx context.Context, tx *sql.Tx, table string) (map[string]string, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`SELECT id, name FROM public.%s`, table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	names := make(map[string]string)
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		if name != "" {
			names[id] = name
		}
	}
	return names, rows.Err()
}

// activityPersonLabel повторяет формат services.logPersonName без пустых скобок; при
// полностью пустых имени и username — сырой id.
func activityPersonLabel(firstName, lastName, username, id string) string {
	name := strings.TrimSpace(lastName + " " + firstName)
	switch {
	case name == "" && username == "":
		return id
	case name == "":
		return username
	case username == "" || strings.EqualFold(username, name):
		return name
	default:
		return fmt.Sprintf("%s (%s)", name, username)
	}
}

// applyActivityLogUpdates пишет изменения пакетами через VALUES. NULL-параметр
// сохраняет исходную колонку (COALESCE), чтобы не превращать SQL NULL в JSON null.
func applyActivityLogUpdates(ctx context.Context, tx *sql.Tx, updates []activityLogUpdate) error {
	const chunkSize = 500

	for start := 0; start < len(updates); start += chunkSize {
		end := min(start+chunkSize, len(updates))
		chunk := updates[start:end]

		placeholders := make([]string, 0, len(chunk))
		args := make([]any, 0, len(chunk)*3)
		for i, u := range chunk {
			placeholders = append(placeholders, fmt.Sprintf("($%d::uuid, $%d::jsonb, $%d::jsonb)", i*3+1, i*3+2, i*3+3))

			var oldArg, newArg any
			if u.oldVal != nil {
				oldArg = string(u.oldVal)
			}
			if u.newVal != nil {
				newArg = string(u.newVal)
			}
			args = append(args, u.id, oldArg, newArg)
		}

		query := fmt.Sprintf(`
			UPDATE public.activity_logs AS a
			SET old_value = COALESCE(v.old_value, a.old_value),
			    new_value = COALESCE(v.new_value, a.new_value)
			FROM (VALUES %s) AS v(id, old_value, new_value)
			WHERE a.id = v.id`, strings.Join(placeholders, ", "))

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to update activity logs: %w", err)
		}
	}

	return nil
}
