package metrics

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	goredis "github.com/redis/go-redis/v9"
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

// ExporterMetrics represent a collection of metrics collected by the exporter
// from Redis and PostgreSQL.
type ExporterMetrics struct {
	Registry *prometheus.Registry
}

// NewExporterMetrics creates and registers PostgreSQL and Redis Collectors.
func NewExporterMetrics(redis *goredis.Client, postgres *pgxpool.Pool) *ExporterMetrics {
	reg := prometheus.NewRegistry()

	m := &ExporterMetrics{
		Registry: reg,
	}

	reg.MustRegister(
		newRedisCollector(redis),
	)

	return m
}
