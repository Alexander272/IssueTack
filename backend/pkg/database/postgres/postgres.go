package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	DBName   string
	SSLMode  string
}

func NewPostgresDB(ctx context.Context, conf *Config) (*pgxpool.Pool, error) {
	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		conf.Username, conf.Password, conf.Host, conf.Port, conf.DBName, conf.SSLMode,
	)

	db, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(ctx); err != nil {
		return nil, err
	}
	return db, nil
}
