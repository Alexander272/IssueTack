package postgres

import (
	"context"
	"sync"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTicketDuplicates_FlattenAndParallelLink_IT проверяет гоночную пару на
// живой PostgreSQL: пока цепочка сплющивается (дети промежуточной основной
// переводятся прямо на корень), параллельно в её ребёнка linking-руется новая
// заявка. Порядок id в LockTickets намеренно разный — детерминированный ORDER BY
// внутри запроса должен исключить дедлок. Гейт — TEST_DATABASE_URL.
func TestTicketDuplicates_FlattenAndParallelLink_IT(t *testing.T) {
	pool := itPool(t)
	ctx := context.Background()
	f := newItFixture()
	t.Cleanup(func() { f.cleanup(t, pool) })

	realmID := f.realm(ctx, pool)
	groupID := f.group(ctx, pool, realmID, "Группа дублей "+realmID.String()[:8])
	userID := f.user(ctx, pool, "dup-"+realmID.String()[:8])
	f.joinRealm(ctx, pool, userID, realmID, f.role(ctx, pool, f.slug(realmID, "dup-user"), realmID))

	rootID := f.ticket(ctx, pool, realmID, groupID, userID, userID)
	midID := f.ticket(ctx, pool, realmID, groupID, userID, userID)
	childID := f.ticket(ctx, pool, realmID, groupID, userID, userID)
	dupID := f.ticket(ctx, pool, realmID, groupID, userID, userID)

	txRepo := NewTransactionRepo(pool)
	repo := NewTicketRepo(pool, txRepo)

	// Исходная цепочка: child → mid → root.
	require.NoError(t, withItTx(ctx, txRepo, func(tx Tx) error {
		if err := repo.SetDuplicate(ctx, tx, midID, rootID, models.StatusOpen); err != nil {
			return err
		}
		return repo.SetDuplicate(ctx, tx, childID, midID, models.StatusOpen)
	}))

	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	// Flatten: все дубли mid (в том числе child) переезжают прямо на root.
	go func() {
		defer wg.Done()
		errs <- withItTx(ctx, txRepo, func(tx Tx) error {
			// Намеренно не sorted — LockTickets сам упорядочит id в SQL.
			if err := repo.LockTickets(ctx, tx, []uuid.UUID{childID, rootID, midID}); err != nil {
				return err
			}
			return repo.RepointDuplicates(ctx, tx, midID, rootID)
		})
	}()

	// Параллельный link ещё одной заявки в child — потомка сплющиваемой ветки.
	go func() {
		defer wg.Done()
		errs <- withItTx(ctx, txRepo, func(tx Tx) error {
			if err := repo.LockTickets(ctx, tx, []uuid.UUID{rootID, dupID, childID}); err != nil {
				return err
			}
			if err := repo.SetDuplicate(ctx, tx, dupID, childID, models.StatusOpen); err != nil {
				return err
			}
			return repo.AddDuplicateNotifier(ctx, tx, childID, userID, dupID, userID)
		})
	}()

	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	// Не появилось петли, цепочка по-прежнему заканчивается на корне.
	chain, err := repo.GetDuplicateChain(ctx, nil, dupID)
	require.NoError(t, err)
	require.NotEmpty(t, chain)
	assert.Equal(t, dupID, chain[0].ID)
	assert.Equal(t, rootID, chain[len(chain)-1].ID)

	hasRefsRoot, err := repo.HasDuplicateReferences(ctx, nil, rootID)
	require.NoError(t, err)
	assert.True(t, hasRefsRoot)
	hasRefsMid, err := repo.HasDuplicateReferences(ctx, nil, midID)
	require.NoError(t, err)
	assert.False(t, hasRefsMid, "после flatten на промежуточную основу никто не ссылается")

	// Отвязка дочернего дубля после flatten: остальные дубли root не трогаются,
	// у child восстанавливается прежний статус.
	require.NoError(t, withItTx(ctx, txRepo, func(tx Tx) error {
		if err := repo.LockTickets(ctx, tx, []uuid.UUID{childID, rootID}); err != nil {
			return err
		}
		if err := repo.ClearDuplicate(ctx, tx, childID); err != nil {
			return err
		}
		return repo.DeleteDuplicateNotifiers(ctx, tx, childID)
	}))

	chain, err = repo.GetDuplicateChain(ctx, nil, childID)
	require.NoError(t, err)
	require.Len(t, chain, 1, "child больше ни на кого не ссылается")
	var status models.TicketStatus
	require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM tickets WHERE id = $1`, childID).Scan(&status))
	assert.Equal(t, models.StatusOpen, status, "восстанавливается previous_status")
	// dup остался привязанным к child; обрыв связи child → root не «перевешивает»
	// дубли на корень молча — связь восстанавливается только явной операцией.
	chain, err = repo.GetDuplicateChain(ctx, nil, dupID)
	require.NoError(t, err)
	require.NotEmpty(t, chain)
	assert.Equal(t, childID, chain[len(chain)-1].ID)
}

// GetNotifiers/GetDuplicateNotifiers не должны смешиваться: явный список
// редактируется формой и полностью перезаписывается SyncNotifiers, неявные
// наблюдатели дублей при этом обязаны остаться на месте.
func TestTicketDuplicates_NotifierSourcesSeparate_IT(t *testing.T) {
	pool := itPool(t)
	ctx := context.Background()
	f := newItFixture()
	t.Cleanup(func() { f.cleanup(t, pool) })

	realmID := f.realm(ctx, pool)
	groupID := f.group(ctx, pool, realmID, "Группа наблюдателей "+realmID.String()[:8])
	explicitID := f.user(ctx, pool, "explicit-"+realmID.String()[:8])
	implicitID := f.user(ctx, pool, "implicit-"+realmID.String()[:8])
	roleID := f.role(ctx, pool, f.slug(realmID, "notifier"), realmID)
	f.joinRealm(ctx, pool, explicitID, realmID, roleID)
	f.joinRealm(ctx, pool, implicitID, realmID, roleID)

	primaryID := f.ticket(ctx, pool, realmID, groupID, explicitID, explicitID)
	dupID := f.ticket(ctx, pool, realmID, groupID, implicitID, implicitID)

	txRepo := NewTransactionRepo(pool)
	repo := NewTicketRepo(pool, txRepo)
	require.NoError(t, withItTx(ctx, txRepo, func(tx Tx) error {
		if err := repo.SetDuplicate(ctx, tx, dupID, primaryID, models.StatusOpen); err != nil {
			return err
		}
		if err := repo.SyncNotifiers(ctx, tx, primaryID, explicitID, []uuid.UUID{explicitID}); err != nil {
			return err
		}
		return repo.AddDuplicateNotifier(ctx, tx, primaryID, implicitID, dupID, explicitID)
	}))

	explicit, err := repo.GetNotifiers(ctx, primaryID)
	require.NoError(t, err)
	require.Len(t, explicit, 1)
	assert.Equal(t, explicitID, explicit[0].ID)

	implicit, err := repo.GetDuplicateNotifiers(ctx, primaryID)
	require.NoError(t, err)
	require.Len(t, implicit, 1)
	assert.Equal(t, implicitID, implicit[0].ID)

	// Чистый SyncNotifiers не должен задеть наблюдателей дублей.
	require.NoError(t, withItTx(ctx, txRepo, func(tx Tx) error {
		return repo.SyncNotifiers(ctx, tx, primaryID, explicitID, nil)
	}))
	explicit, err = repo.GetNotifiers(ctx, primaryID)
	require.NoError(t, err)
	assert.Empty(t, explicit)
	implicit, err = repo.GetDuplicateNotifiers(ctx, primaryID)
	require.NoError(t, err)
	require.Len(t, implicit, 1)
	assert.Equal(t, implicitID, implicit[0].ID)
}

func withItTx(ctx context.Context, tr *TransactionRepo, fn func(Tx) error) error {
	tx, err := tr.BeginTx(ctx)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}
