package postgres

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Интеграционные тесты на уведомления с реальной PostgreSQL.
// Гейт — TEST_DATABASE_URL: без переменной окружения тесты пропускаются.
// Миграции накатываются автоматически (goose.Up).

func TestNotificationRepository_CanSendReminder_Concurrent_IT(t *testing.T) {
	pool := itPool(t)
	ctx := context.Background()
	f := newItFixture()
	t.Cleanup(func() { f.cleanup(t, pool) })

	realmID := f.realm(ctx, pool)
	groupID := f.group(ctx, pool, realmID, "Группа "+realmID.String())
	userID := f.user(ctx, pool, "testuser-"+realmID.String()[:8])
	f.joinRealm(ctx, pool, userID, realmID, f.role(ctx, pool, f.slug(realmID, "user"), realmID))
	ticketID := f.ticket(ctx, pool, realmID, groupID, userID, userID)

	repo := NewNotificationRepo(pool, NewTransactionRepo(pool))
	event := "reminder_high"
	minInterval := 3 * time.Hour

	// Сценарий: 10 горутин одновременно пытаются "занять" напоминание.
	// Ожидание: только одна вернёт true, остальные — false.
	const concurrency = 10
	results := make(chan bool, concurrency)
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := repo.CanSendReminder(ctx, ticketID, event, minInterval)
			require.NoError(t, err)
			results <- ok
		}()
	}

	wg.Wait()
	close(results)

	trueCount := 0
	falseCount := 0
	for result := range results {
		if result {
			trueCount++
		} else {
			falseCount++
		}
	}

	assert.Equal(t, 1, trueCount, "только одна горутина должна получить true")
	assert.Equal(t, concurrency-1, falseCount, "остальные должны получить false")

	// Проверка: в БД должна быть ровно одна строка
	var count int64
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM reminder_last_sent WHERE ticket_id = $1 AND event = $2`, ticketID, event).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "в БД должна быть одна строка напоминания")
}

func TestNotificationRepository_CanSendReminder_Sequential_IT(t *testing.T) {
	pool := itPool(t)
	ctx := context.Background()
	f := newItFixture()
	t.Cleanup(func() { f.cleanup(t, pool) })

	realmID := f.realm(ctx, pool)
	groupID := f.group(ctx, pool, realmID, "Группа "+realmID.String())
	userID := f.user(ctx, pool, "testuser2-"+realmID.String()[:8])
	f.joinRealm(ctx, pool, userID, realmID, f.role(ctx, pool, f.slug(realmID, "user"), realmID))
	ticketID := f.ticket(ctx, pool, realmID, groupID, userID, userID)

	repo := NewNotificationRepo(pool, NewTransactionRepo(pool))
	event := "reminder_high"
	minInterval := 3 * time.Hour

	// Первый вызов — должен вернуть true (строки ещё нет)
	ok1, err := repo.CanSendReminder(ctx, ticketID, event, minInterval)
	require.NoError(t, err)
	assert.True(t, ok1, "первый вызов должен вернуть true")

	// Второй вызов сразу — должен вернуть false (строка только что создана)
	ok2, err := repo.CanSendReminder(ctx, ticketID, event, minInterval)
	require.NoError(t, err)
	assert.False(t, ok2, "второй вызов сразу должен вернуть false")

	// Третий вызов через 3 часа (симулируем старую строку)
	_, err = pool.Exec(ctx, `UPDATE reminder_last_sent SET sent_at = NOW() - INTERVAL '4 hours' WHERE ticket_id = $1 AND event = $2`, ticketID, event)
	require.NoError(t, err)

	ok3, err := repo.CanSendReminder(ctx, ticketID, event, minInterval)
	require.NoError(t, err)
	assert.True(t, ok3, "вызов после истечения интервала должен вернуть true")
}

// itTicket создаёт тестовую заявку для уведомлений.
func (f *itFixture) ticket(ctx context.Context, pool *pgxpool.Pool, realmID, groupID, creatorID, ownerID uuid.UUID) uuid.UUID {
	id := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO tickets (id, realm_id, group_id, creator_id, owner_id, title, description, status, priority)
		VALUES ($1, $2, $3, $4, $5, 'Тестовая заявка', 'Описание', 'open', 'high')`,
		id, realmID, groupID, creatorID, ownerID)
	if err != nil {
		panic(err)
	}
	return id
}