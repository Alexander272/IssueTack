package postgres

import (
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/stretchr/testify/assert"
)

// TestStatusMode фиксирует раскладку статусов по вкладкам. withResolved (страница
// «Мои задачи», mode=assigned) переносит resolved в архив: на активной вкладке
// его нет, на архивной появляется. Пакетные срезы при этом не мутируются.
func TestStatusMode(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }

	cases := []struct {
		name         string
		archived     *bool
		withResolved bool
		wantClause   string
		wantArgs     []models.TicketStatus
	}{
		{
			name:       "active without resolved flag",
			wantArgs:   []models.TicketStatus{"open", "in_progress", "pending", "on_hold", "resolved"},
			wantClause: "t.status IN ($1,$2,$3,$4,$5)",
		},
		{
			name:       "archive without resolved flag",
			archived:   boolPtr(true),
			wantArgs:   []models.TicketStatus{"closed", "cancelled"},
			wantClause: "t.status IN ($1,$2)",
		},
		{
			name:         "active resolved in archive",
			withResolved: true,
			wantArgs:     []models.TicketStatus{"open", "in_progress", "pending", "on_hold"},
			wantClause:   "t.status IN ($1,$2,$3,$4)",
		},
		{
			name:         "archive resolved in archive",
			archived:     boolPtr(true),
			withResolved: true,
			wantArgs:     []models.TicketStatus{"resolved", "closed", "cancelled"},
			wantClause:   "t.status IN ($1,$2,$3)",
		},
		{
			name:       "nil archive means active",
			archived:   nil,
			wantArgs:   []models.TicketStatus{"open", "in_progress", "pending", "on_hold", "resolved"},
			wantClause: "t.status IN ($1,$2,$3,$4,$5)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := &whereBuilder{}
			w.statusMode(tc.archived, tc.withResolved)

			assert.Len(t, w.clauses, 1)
			assert.Equal(t, tc.wantClause, w.clauses[0])
			assert.Equal(t, toAny(tc.wantArgs), w.args)
		})
	}

	// Глобальные наборы не должны меняться вызовами выше.
	assert.Equal(t, []models.TicketStatus{"open", "in_progress", "pending", "on_hold", "resolved"}, activeStatuses)
	assert.Equal(t, []models.TicketStatus{"closed", "cancelled"}, archiveStatuses)
}
