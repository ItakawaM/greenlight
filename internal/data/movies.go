package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ItakawaM/greenlight/internal/validator"
	"github.com/jackc/pgx/v5"
)

// Movie represents a single movie record in the application.
type Movie struct {
	ID        int64     `json:"id" db:"id"`
	CreatedAt time.Time `json:"-" db:"created_at"`
	Title     string    `json:"title" db:"title"`
	Year      int32     `json:"year,omitempty" db:"year"`
	Runtime   int32     `json:"runtime,omitempty" db:"runtime"`
	Genres    []string  `json:"genres,omitempty" db:"genres"`
	Version   int32     `json:"version" db:"version"`
}

// MovieModelInterface defines the storage operations available for movies.
type MovieModelInterface interface {
	// Insert adds a new movie record to the database; populates its
	// ID, CreatedAt and Version fields on success.
	Insert(ctx context.Context, movie *Movie) error

	// Get retrieves a movie record by its ID.
	// Returns ErrRecordNotFound if no movie with that ID exists.
	Get(ctx context.Context, id int64) (*Movie, error)

	// GetAll returns a paginated, filtered and sorted list of movie records
	// together with pagination Metadata.
	// Title performs a full-text search match and is ignored if empty;
	// genres filters for movies containing all specified genres and is
	// ignored if empty.
	GetAll(ctx context.Context, title string, genres []string, filters *Filters) ([]*Movie, Metadata, error)

	// Update persists changes to an existing movie record, using the
	// movie's Version field for optimistic concurrency control.
	// Returns ErrEditConflict if the record was modified concurrently
	// since it was last read.
	Update(ctx context.Context, movie *Movie) error

	// Delete removes the movie record with the given ID.
	// Returns ErrRecordNotFound if no movie with that ID exists.
	Delete(ctx context.Context, id int64) error
}

// MovieModel implements MovieModelInterface.
type MovieModel struct {
	db DBTX
}

// Insert implements MovieModelInterface.
func (m *MovieModel) Insert(ctx context.Context, movie *Movie) error {
	statement :=
		`INSERT INTO movies (title, year, runtime, genres)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version;`

	err := m.db.QueryRow(ctx, statement, movie.Title, movie.Year, movie.Runtime, movie.Genres).
		Scan(&movie.ID, &movie.CreatedAt, &movie.Version)

	return handleContextErrors(err)
}

// Get implements MovieModelInterface.
func (m *MovieModel) Get(ctx context.Context, id int64) (*Movie, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	statement :=
		`SELECT id, created_at, title, year, runtime, genres, version
		FROM movies
		WHERE id = $1;`

	var movie Movie
	if err := m.db.QueryRow(ctx, statement, id).
		Scan(&movie.ID, &movie.CreatedAt, &movie.Title, &movie.Year, &movie.Runtime, &movie.Genres, &movie.Version); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, handleContextErrors(err)
		}
	}

	return &movie, nil
}

// GetAll implements MovieModelInterface.
func (m *MovieModel) GetAll(ctx context.Context, title string, genres []string, filters *Filters) ([]*Movie, Metadata, error) {
	statement :=
		fmt.Sprintf(
			`SELECT count(*) OVER(), id, created_at, title, year, runtime, genres, version
		FROM movies
		WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
		AND (genres @> $2 OR $2 = '{}')
		ORDER BY %s %s, id ASC
		LIMIT $3 OFFSET $4;`, filters.sortColumn(), filters.sortDirection())

	rows, err := m.db.Query(ctx, statement, title, genres, filters.limit(), filters.offset())
	if err != nil {
		return nil, Metadata{}, handleContextErrors(err)
	}
	defer rows.Close()

	totalRecords := 0
	movies := []*Movie{}

	for rows.Next() {
		var movie Movie

		if err := rows.Scan(&totalRecords, &movie.ID, &movie.CreatedAt, &movie.Title, &movie.Year, &movie.Runtime,
			&movie.Genres, &movie.Version); err != nil {
			return nil, Metadata{}, handleContextErrors(err)
		}

		movies = append(movies, &movie)
	}

	if err := rows.Err(); err != nil {
		return nil, Metadata{}, handleContextErrors(err)
	}

	return movies, calculateMetadata(totalRecords, filters.Page, filters.PageSize), nil
}

// Update implements MovieModelInterface.
func (m *MovieModel) Update(ctx context.Context, movie *Movie) error {
	statement :=
		`UPDATE movies
		SET title = $1, year = $2, runtime = $3, genres = $4, version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version;`

	if err := m.db.QueryRow(ctx, statement, movie.Title, movie.Year, movie.Runtime, movie.Genres, movie.ID, movie.Version).
		Scan(&movie.Version); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return ErrEditConflict
		default:
			return handleContextErrors(err)
		}
	}

	return nil
}

// Delete implements MovieModelInterface.
func (m *MovieModel) Delete(ctx context.Context, id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	statement :=
		`DELETE FROM movies
		WHERE id = $1;`

	result, err := m.db.Exec(ctx, statement, id)
	if err != nil {
		return handleContextErrors(err)
	}

	if rowsAffected := result.RowsAffected(); rowsAffected == 0 {
		return ErrRecordNotFound
	}

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
