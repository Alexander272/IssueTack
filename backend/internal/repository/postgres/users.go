package postgres

import "github.com/jackc/pgx/v5/pgxpool"

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{
		db: db,
	}
}

type Users interface{}
