package metrics

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// pgxPoolCollector implements prometheus.Collector and collects pgxpool.Pool stats.
type pgxPoolCollector struct {
	pool *pgxpool.Pool

	// acquireCount is a counter that represents the total count of successful acquires from the pool.
	acquireCount *prometheus.Desc
	// acquireDuration is a counter that represents the total duration of all successful acquires from the pool.
	acquireDuration *prometheus.Desc
	// acquiredConns is a gauge that represents the number of currently acquired connections in the pool.
	acquiredConns *prometheus.Desc
	// canceledAcquireCount is a counter that represents the total count of acquires from the pool that were canceled by a context.
	canceledAcquireCount *prometheus.Desc
	// constructingConns is a gauge that represents the number of conns with construction in progress in the pool.
	constructingConns *prometheus.Desc
	// emptyAcquireCount is a counter that represents the total count of successful acquires from the pool
	// that waited for a resource to be released or constructed because the pool was empty.
	emptyAcquireCount *prometheus.Desc
	// emptyAcquireWaitTime is a counter that represents the total time waited for successful acquires from the pool
	// for a resource to be released or constructed because the pool was empty.
	emptyAcquireWaitTime *prometheus.Desc
	// idleConns is a gauge that represents the number of currently idle conns in the pool.
	idleConns *prometheus.Desc
	// maxConns is a gauge that represents the maximum size of the pool.
	maxConns *prometheus.Desc
	// maxIdleDestroyCount is a counter that represents the total count of connections destroyed
	// because they exceeded MaxConnIdleTime.
	maxIdleDestroyCount *prometheus.Desc
	// maxLifetimeDestroyCountis a counter that represents the total count of connections destroyed
	// because they exceeded MaxConnLifetime.
	maxLifetimeDestroyCount *prometheus.Desc
	// newConnsCount is a counter that represents the total count of new connections opened.
	newConnsCount *prometheus.Desc
	// totalConns is a gauge that represents the number of resources currently in the pool.
	totalConns *prometheus.Desc
}

// newPgxPoolCollector initializes a new pgxpool.Pool metric collector.
func newPgxPoolCollector(pool *pgxpool.Pool) *pgxPoolCollector {
	return &pgxPoolCollector{
		pool: pool,

		acquireCount: prometheus.NewDesc(
			"api_pgxpool_acquire_count_total",
			"Total count of successful acquires from the pool.",
			nil, nil,
		),
		acquireDuration: prometheus.NewDesc(
			"api_pgxpool_acquire_duration_seconds_total",
			"Total duration of all successful acquires from the pool.",
			nil, nil,
		),
		acquiredConns: prometheus.NewDesc(
			"api_pgxpool_acquired_conns",
			"Number of currently acquired connections in the pool.",
			nil, nil,
		),
		canceledAcquireCount: prometheus.NewDesc(
			"api_pgxpool_canceled_acquire_count_total",
			"Total count of acquires from the pool that were canceled by a context.",
			nil, nil,
		),
		constructingConns: prometheus.NewDesc(
			"api_pgxpool_constructing_conns",
			"Number of conns with construction in progress in the pool.",
			nil, nil,
		),
		emptyAcquireCount: prometheus.NewDesc(
			"api_pgxpool_empty_acquire_count_total",
			"Total count of successful acquires from the pool that waited for a resource to be released or constructed because the pool was empty.",
			nil, nil,
		),
		emptyAcquireWaitTime: prometheus.NewDesc(
			"api_pgxpool_empty_acquire_wait_time_seconds_total",
			"Total time waited for successful acquires from the pool for a resource to be released or constructed because the pool was empty.",
			nil, nil,
		),
		idleConns: prometheus.NewDesc(
			"api_pgxpool_idle_conns",
			"Number of currently idle conns in the pool.",
			nil, nil,
		),
		maxConns: prometheus.NewDesc(
			"api_pgxpool_max_conns",
			"Maximum size of the pool.",
			nil, nil,
		),
		maxIdleDestroyCount: prometheus.NewDesc(
			"api_pgxpool_max_idle_destroy_count_total",
			"Total count of connections destroyed because they exceeded MaxConnIdleTime.",
			nil, nil,
		),
		maxLifetimeDestroyCount: prometheus.NewDesc(
			"api_pgxpool_max_lifetime_destroy_count_total",
			"Total count of connections destroyed because they exceeded MaxConnLifetime.",
			nil, nil,
		),
		newConnsCount: prometheus.NewDesc(
			"api_pgxpool_new_conns_count_total",
			"Total count of new connections opened.",
			nil, nil,
		),
		totalConns: prometheus.NewDesc(
			"api_pgxpool_total_conns",
			"Total number of resources currently in the pool.",
			nil, nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (pc *pgxPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- pc.acquireCount
	ch <- pc.acquireDuration
	ch <- pc.acquiredConns
	ch <- pc.canceledAcquireCount
	ch <- pc.constructingConns
	ch <- pc.emptyAcquireCount
	ch <- pc.emptyAcquireWaitTime
	ch <- pc.idleConns
	ch <- pc.maxConns
	ch <- pc.maxIdleDestroyCount
	ch <- pc.maxLifetimeDestroyCount
	ch <- pc.newConnsCount
	ch <- pc.totalConns
}

// Collect implements prometheus.Collector.
func (pc *pgxPoolCollector) Collect(ch chan<- prometheus.Metric) {
	stats := pc.pool.Stat()

	ch <- prometheus.MustNewConstMetric(pc.acquireCount, prometheus.CounterValue, float64(stats.AcquireCount()))
	ch <- prometheus.MustNewConstMetric(pc.acquireDuration, prometheus.CounterValue, stats.AcquireDuration().Seconds())
	ch <- prometheus.MustNewConstMetric(pc.acquiredConns, prometheus.GaugeValue, float64(stats.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(pc.canceledAcquireCount, prometheus.CounterValue, float64(stats.CanceledAcquireCount()))
	ch <- prometheus.MustNewConstMetric(pc.constructingConns, prometheus.GaugeValue, float64(stats.ConstructingConns()))
	ch <- prometheus.MustNewConstMetric(pc.emptyAcquireCount, prometheus.CounterValue, float64(stats.EmptyAcquireCount()))
	ch <- prometheus.MustNewConstMetric(pc.emptyAcquireWaitTime, prometheus.CounterValue, stats.EmptyAcquireWaitTime().Seconds())
	ch <- prometheus.MustNewConstMetric(pc.idleConns, prometheus.GaugeValue, float64(stats.IdleConns()))
	ch <- prometheus.MustNewConstMetric(pc.maxConns, prometheus.GaugeValue, float64(stats.MaxConns()))
	ch <- prometheus.MustNewConstMetric(pc.maxIdleDestroyCount, prometheus.CounterValue, float64(stats.MaxIdleDestroyCount()))
	ch <- prometheus.MustNewConstMetric(pc.maxLifetimeDestroyCount, prometheus.CounterValue, float64(stats.MaxLifetimeDestroyCount()))
	ch <- prometheus.MustNewConstMetric(pc.newConnsCount, prometheus.CounterValue, float64(stats.NewConnsCount()))
	ch <- prometheus.MustNewConstMetric(pc.totalConns, prometheus.GaugeValue, float64(stats.TotalConns()))
}
