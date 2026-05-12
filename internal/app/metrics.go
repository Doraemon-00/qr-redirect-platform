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
	redirectDBLookupsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "redirect_db_lookups_total",
			Help: "Total redirect requests that queried PostgreSQL.",
		},
	)
	analyticsEnqueueFailuresTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "analytics_enqueue_failures_total",
			Help: "Total scan analytics enqueue failures.",
		},
	)
	analyticsEventsWrittenTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "analytics_events_written_total",
			Help: "Total scan analytics events written to ClickHouse.",
		},
	)
	analyticsBatchesWrittenTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "analytics_batches_written_total",
			Help: "Total scan analytics batches written to ClickHouse.",
		},
	)
	analyticsEventsReclaimedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "analytics_events_reclaimed_total",
			Help: "Total pending scan analytics events reclaimed by the worker.",
		},
	)
	analyticsWorkerFailuresTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "analytics_worker_failures_total",
			Help: "Total analytics worker failures.",
		},
	)
	analyticsStreamLength = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "analytics_stream_length",
			Help: "Current Redis scan analytics stream length.",
		},
	)
	analyticsEventsPending = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "analytics_events_pending",
			Help: "Current pending scan analytics events in the Redis consumer group.",
		},
	)
	ownerRateLimitedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "owner_rate_limited_total",
			Help: "Total owner API requests rejected by rate limiting.",
		},
	)
	ownerRateLimitFailuresTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "owner_rate_limit_failures_total",
			Help: "Total owner API rate limiter backend failures.",
		},
	)
	redirectLatencySeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "redirect_latency_seconds",
			Help:    "Redirect endpoint latency.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
	)
	analyticsBatchWriteDurationSeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "analytics_batch_write_duration_seconds",
			Help:    "ClickHouse analytics batch write latency.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		},
	)
)

func init() {
	prometheus.MustRegister(
		redirectRequestsTotal,
		redirectCacheHitsTotal,
		redirectCacheMissesTotal,
		redirectDBLookupsTotal,
		analyticsEnqueueFailuresTotal,
		analyticsEventsWrittenTotal,
		analyticsBatchesWrittenTotal,
		analyticsEventsReclaimedTotal,
		analyticsWorkerFailuresTotal,
		analyticsStreamLength,
		analyticsEventsPending,
		ownerRateLimitedTotal,
		ownerRateLimitFailuresTotal,
		redirectLatencySeconds,
		analyticsBatchWriteDurationSeconds,
	)
}
