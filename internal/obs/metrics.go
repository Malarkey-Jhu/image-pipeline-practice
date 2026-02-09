package obs

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	APIRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_requests_total",
			Help: "Total API requests.",
		},
		[]string{"method", "path", "status"},
	)
	APILatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_request_duration_seconds",
			Help:    "API request latency.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	TasksCreated = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tasks_created_total",
			Help: "Total tasks created.",
		},
	)
	TasksPublished = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tasks_published_total",
			Help: "Total tasks published to MQ.",
		},
	)
	TasksProcessed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tasks_processed_total",
			Help: "Total tasks processed by worker.",
		},
	)
	TasksSkipped = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tasks_skipped_total",
			Help: "Total tasks skipped due to existing output.",
		},
	)
	TasksRetried = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tasks_retried_total",
			Help: "Total tasks retried.",
		},
	)
	TasksFailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tasks_failed_total",
			Help: "Total tasks failed.",
		},
	)
)

func RegisterAll() {
	prometheus.MustRegister(
		APIRequests,
		APILatency,
		TasksCreated,
		TasksPublished,
		TasksProcessed,
		TasksSkipped,
		TasksRetried,
		TasksFailed,
	)
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
