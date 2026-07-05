package data

import (
	"context"
	"time"

	"github.com/ItakawaM/greenlight/internal/validator"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Movie represents a single movie record in the application.
type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	Runtime   int32     `json:"runtime,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"`
}

type MovieModelInterface interface {
	Insert(ctx context.Context, movie *Movie) error
	Get(ctx context.Context, id int64) (*Movie, error)
	Update(ctx context.Context, movie *Movie) error
	Delete(ctx context.Context, id int64) error
}

type MovieModel struct {
	db *pgxpool.Pool
}

func (m *MovieModel) Insert(ctx context.Context, movie *Movie) error {
	statement :=
		`INSERT INTO movies (title, year, runtime, genres)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version;`

	return m.db.QueryRow(ctx, statement, movie.Title, movie.Year, movie.Runtime, movie.Genres).
		Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}

func (m *MovieModel) Get(ctx context.Context, id int64) (*Movie, error) {
	return nil, nil
}

func (m *MovieModel) Update(ctx context.Context, movie *Movie) error {
	return nil
}

func (m *MovieModel) Delete(ctx context.Context, id int64) error {
	return nil
}

// ValidateMovie executes validation checks against a Movie instance, populating
// the provided Validator with any formatting or business-logic errors.
func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(validator.NotBlank(movie.Title), "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")

	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be at least 1888")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")

	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")
	v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}
