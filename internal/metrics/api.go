package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

// newApiUp is a metric that represents whether the application is up.
func newApiUp() prometheus.Gauge {
	return prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "api",
			Name:      "up",
			Help:      "Whether or not the api is up",
		})
}

// newHttpRequestsTotal is a metric that counts the total number of HTTP requests
// by their method, route and status code.
func newHttpRequestsTotal() *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests at greenlight API",
		},
		[]string{"method", "route", "status_code"})
}

// newHttpRequestsDuration is a metric that counts the duration of requests
// by their method, route and status code.
func newHttpRequestsDuration() *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "api",
			Subsystem: "http",
			Name:      "requests_duration_seconds",
			Help:      "HTTP request duration in seconds at greenlight API",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "route", "status_code"})
}

// StatusResponseWriter is a wrapper-struct around http.ResponseWriter
// that allows for recording sent HTTP status codes.
type StatusResponseWriter struct {
	wrapped       http.ResponseWriter
	statusCode    int
	headerWritten bool
}

// NewStatusResponseWriter initializes a StatusResponseWriter object.
func NewStatusResponseWriter(w http.ResponseWriter) *StatusResponseWriter {
	return &StatusResponseWriter{
		wrapped: w,
	}
}

// Header implements http.ResponseWriter interface.
func (mw *StatusResponseWriter) Header() http.Header {
	return mw.wrapped.Header()
}

// WriteHeader implements http.ResponseWriter interface.
func (mw *StatusResponseWriter) WriteHeader(statusCode int) {
	mw.wrapped.WriteHeader(statusCode)

	if !mw.headerWritten {
		mw.statusCode = statusCode
		mw.headerWritten = true
	}
}

// Write implements http.ResponseWriter interface.
func (mw *StatusResponseWriter) Write(b []byte) (int, error) {
	if !mw.headerWritten {
		mw.statusCode = http.StatusOK
		mw.headerWritten = true
	}

	return mw.wrapped.Write(b)
}

// Unwrap returns the underlying http.ResponseWriter.
func (mw *StatusResponseWriter) Unwrap() http.ResponseWriter {
	return mw.wrapped
}

// Status returns the recorded status code.
func (mw *StatusResponseWriter) Status() int {
	if !mw.headerWritten {
		return http.StatusOK
	}

	return mw.statusCode
}
