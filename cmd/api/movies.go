package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ItakawaM/greenlight/internal/data"
	"github.com/ItakawaM/greenlight/internal/validator"
)

// @Summary      Create a movie
// @Description  Creates a new movie record with the given title, year, runtime, and genres.
// @Tags         movies
// @Accept       json
// @Produce      json
// @Param        input  body  CreateMovieRequest  true  "Movie details"
// @Success      201  {object} MovieResponse  "Movie created successfully"
// @Failure      400  {object} ErrorResponse  "Malformed request body"
// @Failure      422  {object} ErrorResponse  "Validation failed"
// @Failure      500  {object} ErrorResponse  "Server encountered a problem"
// @Failure      504  {object} ErrorResponse  "Gateway timeout"
// @Router       /movies [post]
func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	req := CreateMovieRequest{}
	if err := app.readJSON(w, r, &req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	movie := &data.Movie{
		Title:   req.Title,
		Year:    req.Year,
		Runtime: req.Runtime,
		Genres:  req.Genres,
	}

	v := validator.New()
	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := app.models.Movies.Insert(ctx, movie); err != nil {
		app.handleModelError(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/movies/%d", movie.ID))

	if err := app.writeJSON(w, http.StatusCreated, MovieResponse{Movie: movie}, headers); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// @Summary      Get a movie
// @Description  Returns the details of a single movie by ID.
// @Tags         movies
// @Produce      json
// @Param        id   path  int  true  "Movie ID"
// @Success      200  {object} MovieResponse  "Movie found"
// @Failure      404  {object} ErrorResponse  "Movie not found"
// @Failure      500  {object} ErrorResponse  "Server encountered a problem"
// @Failure      504  {object} ErrorResponse  "Gateway timeout"
// @Router       /movies/{id} [get]
func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	movie, err := app.models.Movies.Get(ctx, id)
	if err != nil {
		app.handleModelError(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusOK, MovieResponse{Movie: movie}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// @Summary      List movies
// @Description  Returns a paginated, filterable, sortable list of movies.
// @Tags         movies
// @Produce      json
// @Param        title      query  string  false  "Filter by title (partial match)"
// @Param        genres     query  string  false  "Filter by genres (comma-separated)"
// @Param        page       query  int     false  "Page number"       default(1)
// @Param        page_size  query  int     false  "Items per page"    default(20)
// @Param        sort       query  string  false  "Sort field, prefix with - for descending. One of: id, title, year, runtime, -id, -title, -year, -runtime"  default(id)
// @Success      200  {object} ListMovieResponse  "List of movies with pagination metadata"
// @Failure      422  {object} ErrorResponse  "Validation failed"
// @Failure      500  {object} ErrorResponse  "Server encountered a problem"
// @Failure      504  {object} ErrorResponse  "Gateway timeout"
// @Router       /movies [get]
func (app *application) listMoviesHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title  string
		Genres []string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Title = app.readString(qs, "title", "")
	input.Genres = app.readCSV(qs, "genres", []string{})

	input.Page = app.readInt(qs, "page", 1, v)
	input.PageSize = app.readInt(qs, "page_size", 20, v)

	input.Sort = app.readString(qs, "sort", "id")
	input.SortSafeList = []string{"id", "title", "year", "runtime", "-id", "-title", "-year", "-runtime"}

	if data.ValidateFilters(v, &input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	movies, metadata, err := app.models.Movies.GetAll(ctx, input.Title, input.Genres, &input.Filters)
	if err != nil {
		app.handleModelError(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusOK, ListMovieResponse{Movies: movies, Metadata: metadata}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// @Summary      Update a movie
// @Description  Partially updates an existing movie. Only fields present in the request body are changed.
// @Tags         movies
// @Accept       json
// @Produce      json
// @Param        id     path  int                                                               true  "Movie ID"
// @Param        input  body  UpdateMovieRequest     true  "Fields to update"
// @Success      200  {object} MovieResponse  "Movie updated successfully"
// @Failure      400  {object} ErrorResponse  "Malformed request body"
// @Failure      404  {object} ErrorResponse  "Movie not found"
// @Failure      409  {object} ErrorResponse  "Concurrent update conflict"
// @Failure      422  {object} ErrorResponse  "Validation failed"
// @Failure      500  {object} ErrorResponse  "Server encountered a problem"
// @Failure      504  {object} ErrorResponse  "Gateway timeout"
// @Router       /movies/{id} [patch]
func (app *application) updateMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	movie, err := app.models.Movies.Get(ctx, id)
	if err != nil {
		app.handleModelError(w, r, err)
		return
	}

	req := UpdateMovieRequest{}
	if err := app.readJSON(w, r, &req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if req.Title != nil {
		movie.Title = *req.Title
	}

	if req.Year != nil {
		movie.Year = *req.Year
	}

	if req.Runtime != nil {
		movie.Runtime = *req.Runtime
	}

	if req.Genres != nil {
		movie.Genres = req.Genres
	}

	v := validator.New()
	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	if err := app.models.Movies.Update(ctx, movie); err != nil {
		app.handleModelError(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusOK, MovieResponse{Movie: movie}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// @Summary      Delete a movie
// @Description  Permanently deletes a movie by ID.
// @Tags         movies
// @Produce      json
// @Param        id   path  int  true  "Movie ID"
// @Success      200  {object} MessageResponse  "Movie deleted successfully"
// @Failure      404  {object} ErrorResponse  "Movie not found"
// @Failure      500  {object} ErrorResponse  "Server encountered a problem"
// @Failure      504  {object} ErrorResponse  "Gateway timeout"
// @Router       /movies/{id} [delete]
func (app *application) deleteMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := app.models.Movies.Delete(ctx, id); err != nil {
		app.handleModelError(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusOK, MessageResponse{Message: "movie successfully deleted"}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
