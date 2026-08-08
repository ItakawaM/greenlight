package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ItakawaM/greenlight/internal/data"
	"github.com/ItakawaM/greenlight/internal/validator"
)

// authenticate is a middleware that checks the provided authorization token in the request
// and stores the user in the context.
// It stores AnonymousUser if no Authorization header is provided.
// It errors with 401 if the Authorization header is malformed or the token is invalid.
func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		token := headerParts[1]

		v := validator.New()
		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel() // we can safely defer here, because next.ServeHTTP(w, r) uses a different context

		user, err := app.models.Users.GetForToken(ctx, data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)

			default:
				app.handleContextErrors(w, r, err)
			}
			return
		}

		r = app.contextSetUser(r, user)
		next.ServeHTTP(w, r)
	})
}

// rateLimit is a middleware that limits the amount of requests
// a user from a single IP Address can do, implementing a token bucket algorithm.
//
// Configurable via .env and flags:
//   - Requests per Minute
//   - Burst requests
//   - State: disabled/enabled
func (app *application) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.config.limiter.enabled {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}

			metadata, err := app.limiter.Allow(r.Context(), "ip:"+ip)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(app.config.limiter.burst))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(int64(math.Floor(metadata.Remaining)), 10))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(
				time.Now().Add(time.Duration(metadata.ResetAfter*float64(time.Second))).Unix(), 10))

			if !metadata.Allowed {
				w.Header().Set("Retry-After", strconv.FormatInt(int64(math.Ceil(metadata.RetryAfter)), 10))
				app.rateLimitExceededResponse(w, r)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// recoverPanic is a middleware that gracefully recovers panics in http method handlers,
// logs them and sends a 500 response to the client.
func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%v", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
