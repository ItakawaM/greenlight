package main

import (
	"context"
	"net/http"

	"github.com/ItakawaM/greenlight/internal/data"
)

// contextKey represents a unique key for context values.
type contextKey string

const (
	// userContextKey is a unique identifier for user context value.
	userContextKey contextKey = "user"
)

// contextSetUser sets the provided user into the context.
func (app *application) contextSetUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

// contextGetUser returns the user stored in the context.
// It panics when no user exists in the context (example: missing middleware).
func (app *application) contextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		panic("missing user value in request context")
	}

	return user
}
