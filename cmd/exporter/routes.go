package main

import (
	"net/http"

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
		promhttp.HandlerOpts{},
	))

	return mux
}
