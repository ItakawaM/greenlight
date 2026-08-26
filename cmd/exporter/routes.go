package main

import (
	"net/http"

	"github.com/ItakawaM/greenlight/internal/logging"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// routes configures routes for the exporter.
func (e *exporter) routes() http.Handler {
	mux := http.NewServeMux()

	// healthcheck
	mux.HandleFunc("GET /healthcheck", e.healthcheckHandler)

	// metrics
	mux.Handle("/metrics", promhttp.HandlerFor(
		e.metrics.Registry,
		promhttp.HandlerOpts{
			ErrorLog:      logging.NewSlogProm(e.logger),
			ErrorHandling: promhttp.HTTPErrorOnError,
		},
	))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					w.Header().Set("Connection", "close")
					w.Header().Set("Content-Type", "text/plain; charset=utf-8")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(err.(error).Error()))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}(mux)
}
