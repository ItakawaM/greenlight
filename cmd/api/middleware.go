package main

import (
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"time"
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

			allowed, remaining, retryAfter, resetAfter, err := app.limiter.Allow(r.Context(), fmt.Sprintf("ip:%s", ip))
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(app.config.limiter.burst))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatFloat(remaining, 'f', 0, 64))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(
				time.Now().Add(time.Duration(resetAfter*float64(time.Second))).Unix(), 10))

			if !allowed {
				w.Header().Set("Retry-After", strconv.FormatInt(int64(math.Ceil(retryAfter)), 10))
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
