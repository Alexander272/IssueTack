package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/migrate"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/server"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport"
	"github.com/Alexander272/IssueTrack/backend/pkg/auth"
	"github.com/Alexander272/IssueTrack/backend/pkg/database/postgres"
	"github.com/Alexander272/IssueTrack/backend/pkg/database/redis"
	"github.com/Alexander272/IssueTrack/backend/pkg/limiter"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/Alexander272/IssueTrack/backend/pkg/ws_hub"
	"github.com/subosito/gotenv"
)

func main() {
	//* Init config
	if err := gotenv.Load(".env"); err != nil {
		log.Printf("warning: error loading .env file: %s", err.Error())
	}

	conf, err := config.Init("configs/config.yaml")
	if err != nil {
		log.Fatalf("error initializing configs: %s", err.Error())
	}
	logger.NewLogger(logger.WithLevel(conf.LogLevel), logger.WithAddSource(conf.LogSource))

	//* Dependencies
	db, err := postgres.NewPostgresDB(context.Background(), &postgres.Config{
		Host:     conf.Postgres.Host,
		Port:     conf.Postgres.Port,
		Username: conf.Postgres.Username,
		Password: conf.Postgres.Password,
		DBName:   conf.Postgres.DbName,
		SSLMode:  conf.Postgres.SSLMode,
	})
	if err != nil {
		log.Fatalf("failed to initialize db: %s", err.Error())
	}
	if err := migrate.Migrate(db); err != nil {
		log.Fatalf("failed to migrate: %s", err.Error())
	}

	memDb, err := redis.NewRedisClient(&redis.Config{
		Host:     conf.Redis.Host,
		Port:     conf.Redis.Port,
		Password: conf.Redis.Password,
		DB:       conf.Redis.DB,
	})
	if err != nil {
		log.Fatalf("failed to initialize redis: %s", err.Error())
	}

	keycloak := auth.NewKeycloakClient(&auth.Deps{
		Url:       conf.Keycloak.Url,
		ClientId:  conf.Keycloak.ClientId,
		Realm:     conf.Keycloak.Realm,
		GroupName: conf.Keycloak.GroupName,
		AdminName: conf.Keycloak.Root,
		AdminPass: conf.Keycloak.RootPass,
	})

	// Контекст для управления всеми фоновыми процессами
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := ws_hub.NewWebsocketHub()
	go hub.Run(ctx)

	//* Services, Repos & API Handlers
	repo := repository.NewRepository(db, memDb, conf.Auth)
	service := services.NewServices(&services.Deps{
		Ctx:      ctx,
		Conf:     conf,
		Repo:     repo,
		Hub:      hub,
		Keycloak: keycloak,
	})

	handlers := transport.NewHandler(keycloak, service, hub)

	//* HTTP Server
	// if err := services.Scheduler.Start(&conf.Scheduler); err != nil {
	// 	log.Fatalf("failed to start scheduler. error: %s\n", err.Error())
	// }

	// Запускаем все Runner'ы
	// for _, runner := range service.GetRunners() {
	// 	go func(r services.Runner) {
	// 		if err := r.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
	// 			log.Printf("Runner error: %v", err)
	// 			// Тут можно реализовать логику паники или уведомления в телеграм
	// 		}
	// 	}(runner)
	// }

	srv := server.NewServer(conf, handlers.Init(conf))
	go func() {
		if err := srv.Run(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("error occurred while running http server: %s\n", err.Error())
		}
	}()
	logger.Info("Application started on port: " + conf.Http.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	<-quit

	logger.Info("Shutting down server...")

	const timeout = 5 * time.Second

	shutdownCtx, shutdown := context.WithTimeout(context.Background(), timeout)
	defer shutdown()

	// if err := services.Scheduler.Stop(); err != nil {
	// 	logger.Error("failed to stop scheduler.", logger.ErrAttr(err))
	// }

	if err := srv.Stop(shutdownCtx); err != nil {
		logger.Error("failed to stop server:", logger.ErrAttr(err))
	}

	// Остановка всех rate limiter'ов
	limiter.StopAll()

	hub.Stop()

	cancel()

	db.Close()
	logger.Info("Database connection closed")
}
