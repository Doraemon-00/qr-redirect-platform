package app

import "github.com/prometheus/client_golang/prometheus"

var (
	redirectRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redirect_requests_total",
			Help: "Total redirect requests by result.",
		},
		[]string{"result"},
	)
	redirectCacheHitsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "redirect_cache_hits_total",
			Help: "Total redirect cache hits.",
		},
	)
	redirectCacheMissesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "redirect_cache_misses_total",
			Help: "Total redirect cache misses.",
		},
	)
	analyticsEnqueueFailuresTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "analytics_enqueue_failures_total",
			Help: "Total scan analytics enqueue failures.",
		},
	)
	redirectLatencySeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "redirect_latency_seconds",
			Help:    "Redirect endpoint latency.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
	)
)

func init() {
	prometheus.MustRegister(
		redirectRequestsTotal,
		redirectCacheHitsTotal,
		redirectCacheMissesTotal,
		analyticsEnqueueFailuresTotal,
		redirectLatencySeconds,
	)
}
