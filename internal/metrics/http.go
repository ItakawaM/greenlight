package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

func newHttpRequestsTotal() *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "greenlight",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests at greenlight API",
		},
		[]string{"method", "route", "status_code"})
}

func newHttpRequestsDuration() *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "greenlight",
			Subsystem: "http",
			Name:      "requests_duration_seconds",
			Help:      "HTTP request duration in seconds at greenlight API",
			Buckets:   prometheus.DefBuckets,
			Unit:      "seconds",
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

func (mw *StatusResponseWriter) Status() int {
	return mw.statusCode
}
