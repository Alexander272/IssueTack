package enforcer

import (
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/casbin/casbin/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxadapter "github.com/noho-digital/casbin-pgx-adapter"
)

// type EnfDB struct {
// 	db    *pgxpool.Pool
// 	redis *redis.Client
// }

// type EnforcerDeps struct {
// 	conf config.CasbinConfig
// 	DB
// }

func NewEnforcer(conf config.CasbinConfig, pool *pgxpool.Pool) (*casbin.Enforcer, error) {
	// Адаптер для хранения политик в Postgres
	adapter, err := pgxadapter.NewAdapter(pool.Config().ConnString(), pgxadapter.WithTableName("casbin_rule"))
	if err != nil {
		return nil, err
	}

	// Загружаем модель из конфига
	enforcer, err := casbin.NewEnforcer(conf.ModelPath, adapter)
	if err != nil {
		return nil, err
	}

	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load policy from DB: %w", err)
	}

	// if config.EnableWatcher && config.RedisAddr != "" {
	// 	redisClient := redis.NewClient(&redis.Options{
	// 		Addr: config.RedisAddr,
	// 	})

	// 	// Проверка подключения к Redis
	// 	if err := redisClient.Ping(ctx).Err(); err != nil {
	// 		return nil, fmt.Errorf("failed to connect to Redis for watcher: %w", err)
	// 	}

	// 	watcher, err := rediswatcher.NewWatcher(config.RedisAddr, rediswatcher.WatcherOptions{
	// 		Options: *redisClient.Options(),
	// 	})
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to create redis watcher: %w", err)
	// 	}

	// 	// Устанавливаем callback для обновления политик при изменении в других инстансах
	// 	enforcer.SetWatcher(watcher)

	// 	// Регистрируем функцию, которая будет вызываться при получении обновления от других инстансов
	// 	if err := watcher.SetUpdateCallback(func(string) {
	// 		// Загружаем обновленные политики из БД
	// 		if loadErr := enforcer.LoadPolicy(); loadErr != nil {
	// 			fmt.Printf("Failed to reload policy from watcher: %v\n", loadErr)
	// 		}
	// 	}); err != nil {
	// 		return nil, fmt.Errorf("failed to set watcher callback: %w", err)
	// 	}
	// }

	// Включаем кэш ролей для производительности
	enforcer.EnableAutoSave(true)

	return enforcer, nil
}
