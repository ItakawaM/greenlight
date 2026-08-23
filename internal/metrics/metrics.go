package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type Metrics struct {
	Registry             *prometheus.Registry
	HttpRequestsTotal    *prometheus.CounterVec
	HttpRequestsDuration *prometheus.HistogramVec
}

func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()

	m := &Metrics{
		Registry:             reg,
		HttpRequestsTotal:    newHttpRequestsTotal(),
		HttpRequestsDuration: newHttpRequestsDuration(),
	}

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.HttpRequestsTotal,
		m.HttpRequestsDuration,
	)

	return m
}
