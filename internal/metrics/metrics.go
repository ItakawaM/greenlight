package metrics

import (
	"expvar"
	"net/http"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// pgxPoolStats represent pgxpool connection pool stats.
type pgxPoolStats struct {
	AcquireCount            int64 `json:"acquire_count"`
	AcquireDuration         int64 `json:"acquire_duration_ns"`
	AcquiredConns           int32 `json:"acquired_conns"`
	CanceledAcquireCount    int64 `json:"canceled_acquire_count"`
	ConstructingConns       int32 `json:"constructing_conns"`
	EmptyAcquireCount       int64 `json:"empty_acquire_count"`
	IdleConns               int32 `json:"idle_conns"`
	MaxConns                int32 `json:"max_conns"`
	MaxIdleDestroyCount     int64 `json:"max_idle_destroy_count"`
	MaxLifetimeDestroyCount int64 `json:"max_lifetime_destroy_count"`
	NewConnsCount           int64 `json:"new_conns_count"`
	TotalConns              int32 `json:"total_conns"`
}

// MetricsResponseWriter is a wrapper-struct around http.ResponseWriter
// that allows for recording sent HTTP status codes.
type MetricsResponseWriter struct {
	wrapped       http.ResponseWriter
	statusCode    int
	headerWritten bool
}

// NewMetricsResponseWriter initializes a MetricsResponseWriter object.
func NewMetricsResponseWriter(w http.ResponseWriter) *MetricsResponseWriter {
	return &MetricsResponseWriter{
		wrapped: w,
	}
}

// Header implements http.ResponseWriter interface.
func (mw *MetricsResponseWriter) Header() http.Header {
	return mw.wrapped.Header()
}

// WriteHeader implements http.ResponseWriter interface.
func (mw *MetricsResponseWriter) WriteHeader(statusCode int) {
	mw.wrapped.WriteHeader(statusCode)

	if !mw.headerWritten {
		mw.statusCode = statusCode
		mw.headerWritten = true
	}
}

// Write implements http.ResponseWriter interface.
func (mw *MetricsResponseWriter) Write(b []byte) (int, error) {
	if !mw.headerWritten {
		mw.statusCode = http.StatusOK
		mw.headerWritten = true
	}

	return mw.wrapped.Write(b)
}

// Unwrap returns the underlying http.ResponseWriter.
func (mw *MetricsResponseWriter) Unwrap() http.ResponseWriter {
	return mw.wrapped
}

func (mw *MetricsResponseWriter) Status() int {
	return mw.statusCode
}

// PublishGlobalMetrics sets global metrics for expvar metrics endpoint.
func PublishGlobalMetrics(postgres *pgxpool.Pool, redis *redis.Client, version string) {
	expvar.NewString("version").Set(version)

	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))

	expvar.Publish("postgres", expvar.Func(func() any {
		s := postgres.Stat()
		return pgxPoolStats{
			AcquireCount:            s.AcquireCount(),
			AcquireDuration:         int64(s.AcquireDuration()),
			AcquiredConns:           s.AcquiredConns(),
			CanceledAcquireCount:    s.CanceledAcquireCount(),
			ConstructingConns:       s.ConstructingConns(),
			EmptyAcquireCount:       s.EmptyAcquireCount(),
			IdleConns:               s.IdleConns(),
			MaxConns:                s.MaxConns(),
			MaxIdleDestroyCount:     s.MaxIdleDestroyCount(),
			MaxLifetimeDestroyCount: s.MaxLifetimeDestroyCount(),
			NewConnsCount:           s.NewConnsCount(),
			TotalConns:              s.TotalConns(),
		}
	}))

	expvar.Publish("redis", expvar.Func(func() any {
		return redis.PoolStats()
	}))

	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))
}
