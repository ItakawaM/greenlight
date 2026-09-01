package metrics

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// APIMetrics represent a collection of metrics collected by the API.
type APIMetrics struct {
	Registry             *prometheus.Registry
	HttpRequestsTotal    *prometheus.CounterVec
	HttpRequestsDuration *prometheus.HistogramVec
	APIUp                prometheus.Gauge
}

// NewAPIMetrics creates and registers API metrics.
func NewAPIMetrics(postgres *pgxpool.Pool) *APIMetrics {
	reg := prometheus.NewRegistry()

	m := &APIMetrics{
		Registry:             reg,
		HttpRequestsTotal:    newHttpRequestsTotal(),
		HttpRequestsDuration: newHttpRequestsDuration(),
		APIUp:                newApiUp(),
	}

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		newPgxPoolCollector(postgres),
		m.HttpRequestsTotal,
		m.HttpRequestsDuration,
		m.APIUp,
	)

	return m
}
