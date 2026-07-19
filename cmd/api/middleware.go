package main

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
)

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

			allowed, remaining, err := app.limiter.Allow(r.Context(), fmt.Sprintf("ip:%s", ip))
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}

			// TODO: Adjust rate limiting headers and LUA script to compute time needed for Retry-After
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(app.config.limiter.burst))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatFloat(remaining, 'f', 0, 64))

			if !allowed {
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
