package data

import "github.com/jackc/pgx/v5/pgxpool"

type Models struct {
	Movies MovieModelInterface
}

func NewModels(db *pgxpool.Pool) *Models {
	return &Models{
		Movies: &MovieModel{db: db},
	}
}
