package main

import "github.com/ItakawaM/greenlight/internal/data"

// CreateMovieRequest represents a request object for movie creation.
type CreateMovieRequest struct {
	Title   string   `json:"title"`
	Year    int32    `json:"year"`
	Runtime int32    `json:"runtime"`
	Genres  []string `json:"genres"`
}

// CreateMovieRequest represents a request object for partial movie update.
type UpdateMovieRequest struct {
	Title   *string  `json:"title"`
	Year    *int32   `json:"year"`
	Runtime *int32   `json:"runtime"`
	Genres  []string `json:"genres"`
}

// MovieResponse represents a wrapped response containing a single movie object.
type MovieResponse struct {
	Movie *data.Movie `json:"movie"`
}

// MovieListResponse represents a wrapped response containing multiple movies and pagination metadata.
type ListMovieResponse struct {
	Movies   []*data.Movie `json:"movies"`
	Metadata data.Metadata `json:"metadata"`
}
