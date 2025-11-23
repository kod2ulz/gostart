package api

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/kod2ulz/gostart/contracts"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// HTTP metrics
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	RequestSize      *prometheus.HistogramVec
	ResponseSize     *prometheus.HistogramVec
	RequestsInFlight prometheus.Gauge

	// Error metrics
	ErrorsTotal *prometheus.CounterVec

	// Cache metrics
	CacheHits   *prometheus.CounterVec
	CacheMisses *prometheus.CounterVec

	// Rate limit metrics
	RateLimitExceeded *prometheus.CounterVec

	// Database metrics
	DBQueriesTotal    *prometheus.CounterVec
	DBQueryDuration   *prometheus.HistogramVec
	DBConnections     prometheus.Gauge
	DBConnectionsIdle prometheus.Gauge

	// Queue metrics
	QueueMessagesPublished *prometheus.CounterVec
	QueueMessagesConsumed  *prometheus.CounterVec
	QueueMessagesFailed    *prometheus.CounterVec
	QueueProcessingTime    *prometheus.HistogramVec

	// Custom business metrics
	CustomCounters   map[string]*prometheus.CounterVec
	CustomGauges     map[string]*prometheus.GaugeVec
	CustomHistograms map[string]*prometheus.HistogramVec
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics(namespace string) *Metrics {
	if namespace == "" {
		namespace = "gostart"
	}

	m := &Metrics{
		CustomCounters:   make(map[string]*prometheus.CounterVec),
		CustomGauges:     make(map[string]*prometheus.GaugeVec),
		CustomHistograms: make(map[string]*prometheus.HistogramVec),
	}

	// HTTP metrics
	m.RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	m.RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	m.RequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_size_bytes",
			Help:      "HTTP request size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	m.ResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	m.RequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "http_requests_in_flight",
			Help:      "Current number of HTTP requests being processed",
		},
	)

	// Error metrics
	m.ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "errors_total",
			Help:      "Total number of errors",
		},
		[]string{"type", "code"},
	)

	// Cache metrics
	m.CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cache_hits_total",
			Help:      "Total number of cache hits",
		},
		[]string{"cache"},
	)

	m.CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cache_misses_total",
			Help:      "Total number of cache misses",
		},
		[]string{"cache"},
	)

	// Rate limit metrics
	m.RateLimitExceeded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "rate_limit_exceeded_total",
			Help:      "Total number of rate limit violations",
		},
		[]string{"endpoint"},
	)

	// Database metrics
	m.DBQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "db_queries_total",
			Help:      "Total number of database queries",
		},
		[]string{"operation", "table"},
	)

	m.DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "db_query_duration_seconds",
			Help:      "Database query duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	m.DBConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "db_connections",
			Help:      "Current number of database connections",
		},
	)

	m.DBConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "db_connections_idle",
			Help:      "Current number of idle database connections",
		},
	)

	// Queue metrics
	m.QueueMessagesPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "queue_messages_published_total",
			Help:      "Total number of messages published to queue",
		},
		[]string{"queue", "topic"},
	)

	m.QueueMessagesConsumed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "queue_messages_consumed_total",
			Help:      "Total number of messages consumed from queue",
		},
		[]string{"queue", "topic"},
	)

	m.QueueMessagesFailed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "queue_messages_failed_total",
			Help:      "Total number of failed message processing",
		},
		[]string{"queue", "topic", "error_type"},
	)

	m.QueueProcessingTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "queue_processing_duration_seconds",
			Help:      "Message processing duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"queue", "topic"},
	)

	return m
}

// RegisterCustomCounter registers a custom counter metric
func (m *Metrics) RegisterCustomCounter(name, help string, labels []string) *prometheus.CounterVec {
	counter := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: help,
		},
		labels,
	)
	m.CustomCounters[name] = counter
	return counter
}

// RegisterCustomGauge registers a custom gauge metric
func (m *Metrics) RegisterCustomGauge(name, help string, labels []string) *prometheus.GaugeVec {
	gauge := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: help,
		},
		labels,
	)
	m.CustomGauges[name] = gauge
	return gauge
}

// RegisterCustomHistogram registers a custom histogram metric
func (m *Metrics) RegisterCustomHistogram(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	if buckets == nil {
		buckets = prometheus.DefBuckets
	}

	histogram := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    name,
			Help:    help,
			Buckets: buckets,
		},
		labels,
	)
	m.CustomHistograms[name] = histogram
	return histogram
}

// PrometheusMiddleware returns middleware that collects HTTP metrics
func PrometheusMiddleware(metrics *Metrics) func(contracts.RequestContext) {
	return func(ctx contracts.RequestContext) {
		start := time.Now()

		// Increment in-flight requests
		metrics.RequestsInFlight.Inc()
		defer metrics.RequestsInFlight.Dec()

		// Process request
		ctx.Next()

		// Record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(ctx.Writer().Status())
		path := ctx.Path()
		method := ctx.Method()

		metrics.RequestsTotal.WithLabelValues(method, path, status).Inc()
		metrics.RequestDuration.WithLabelValues(method, path).Observe(duration)

		// Record request/response sizes if available
		if ctx.Request().ContentLength > 0 {
			metrics.RequestSize.WithLabelValues(method, path).Observe(float64(ctx.Request().ContentLength))
		}

		// Record errors
		if ctx.Writer().Status() >= 400 {
			errorType := "client_error"
			if ctx.Writer().Status() >= 500 {
				errorType = "server_error"
			}
			metrics.ErrorsTotal.WithLabelValues(errorType, status).Inc()
		}
	}
}

// MetricsHandler returns a Prometheus metrics handler
func MetricsHandler() contracts.HandlerFunc {
	handler := promhttp.Handler()
	return func(ctx contracts.RequestContext) {
		handler.ServeHTTP(ctx.Writer(), ctx.Request())
	}
}

// Timer helps measure operation duration
type Timer struct {
	start time.Time
	histogram *prometheus.HistogramVec
	labels    []string
}

// NewTimer creates a new timer
func NewTimer(histogram *prometheus.HistogramVec, labels ...string) *Timer {
	return &Timer{
		start:     time.Now(),
		histogram: histogram,
		labels:    labels,
	}
}

// ObserveDuration records the duration since timer creation
func (t *Timer) ObserveDuration() {
	duration := time.Since(t.start).Seconds()
	t.histogram.WithLabelValues(t.labels...).Observe(duration)
}

// Example usage showing how to use metrics in handlers
func ExampleMetricsUsage(metrics *Metrics) {
	// In HTTP handler
	timer := NewTimer(metrics.RequestDuration, "GET", "/users")
	defer timer.ObserveDuration()

	// Track cache hits/misses
	metrics.CacheHits.WithLabelValues("user_cache").Inc()

	// Track database queries
	dbTimer := NewTimer(metrics.DBQueryDuration, "SELECT", "users")
	// ... execute query ...
	dbTimer.ObserveDuration()
	metrics.DBQueriesTotal.WithLabelValues("SELECT", "users").Inc()

	// Track queue messages
	metrics.QueueMessagesPublished.WithLabelValues("rabbitmq", "user_events").Inc()

	// Custom metrics
	if counter, exists := metrics.CustomCounters["user_logins_total"]; exists {
		counter.WithLabelValues("success").Inc()
	}
}
