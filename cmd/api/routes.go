package main

import (
	"net/http"

	"github.com/ItakawaM/greenlight/internal/data"
)

// routes returns a serveMux with defined routes and middleware.
func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		app.notFoundResponse(w, r)
	})

	mux.HandleFunc("GET /v1/healthcheck", app.healthcheckHandler)

	// Movies API
	mux.HandleFunc("POST /v1/movies", app.requirePermissions(app.createMovieHandler, data.MoviesWritePermission))
	mux.HandleFunc("GET /v1/movies/{id}", app.requirePermissions(app.showMovieHandler, data.MoviesReadPermission))
	mux.HandleFunc("GET /v1/movies", app.requirePermissions(app.listMoviesHandler, data.MoviesReadPermission))
	mux.HandleFunc("PATCH /v1/movies/{id}", app.requirePermissions(app.updateMovieHandler, data.MoviesWritePermission))
	mux.HandleFunc("DELETE /v1/movies/{id}", app.requirePermissions(app.deleteMovieHandler, data.MoviesWritePermission))

	// Users API
	mux.HandleFunc("POST /v1/users", app.registerUserHandler)
	mux.HandleFunc("PUT /v1/users/activate", app.activateUserHandler)

	// Tokens API
	mux.HandleFunc("POST /v1/tokens/authentication", app.createAuthenticationTokenHandler)

	return app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(mux))))
}
