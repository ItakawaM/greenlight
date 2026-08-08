package main

import "github.com/ItakawaM/greenlight/internal/data"

// CreateMovieRequest represents a request object for movie creation.
type CreateMovieRequest struct {
	Title   string   `json:"title" example:"Blade Runner"`
	Year    int32    `json:"year" example:"1982"`
	Runtime int32    `json:"runtime" example:"117"`
	Genres  []string `json:"genres" example:"sci-fi,action"`
}

// UpdateMovieRequest represents a request object for partial movie update.
type UpdateMovieRequest struct {
	Title   *string  `json:"title" example:"Blade Runner"`
	Year    *int32   `json:"year" example:"1982"`
	Runtime *int32   `json:"runtime" example:"117"`
	Genres  []string `json:"genres" example:"sci-fi,action"`
}

// ListMovieRequest represents url values for paginated movies listing.
type ListMovieRequest struct {
	Title  string
	Genres []string
	data.Filters
}

// MovieResponse represents a wrapped response containing a single movie object.
type MovieResponse struct {
	Movie *data.Movie `json:"movie"`
}

// ListMovieResponse represents a wrapped response containing multiple movies and pagination metadata.
type ListMovieResponse struct {
	Movies   []*data.Movie `json:"movies"`
	Metadata data.Metadata `json:"metadata"`
}
