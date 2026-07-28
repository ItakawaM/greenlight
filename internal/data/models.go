package data

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// Models represents a wrapped struct around all DB models.
type Models struct {
	Movies MovieModelInterface
	Users  UserModelInterface
	Tokens TokenModelInterface
}

// NewModels creates a new Models instance with all the implemented DB models.
func NewModels(db *pgxpool.Pool) *Models {
	return &Models{
		Movies: &MovieModel{db: db},
		Users:  &UserModel{db: db},
		Tokens: &TokenModel{db: db},
	}
}
