package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Дубль исключается только для периодов, которые закончились после установки
// связи: исторический отчёт о заявке, помеченной дублем сегодня, не меняется.
func TestStatisticsDuplicateExcluded(t *testing.T) {
	to := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	prevTo := to.Add(-24 * time.Hour)
	w := &whereBuilder{}

	current := w.statisticsDuplicateExcluded(to)
	previous := w.statisticsDuplicateExcluded(prevTo)

	assert.Contains(t, current, "t.duplicate_of IS NULL")
	assert.Contains(t, current, "t.duplicate_of_set_at IS NOT NULL AND t.duplicate_of_set_at >= $1")
	assert.Contains(t, previous, "t.duplicate_of_set_at >= $2")
	assert.Equal(t, []any{to, prevTo}, w.args)
}
