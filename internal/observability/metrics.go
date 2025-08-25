package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	RequestCount      *prometheus.CounterVec
	RequestDuration   *prometheus.HistogramVec
	RequestSize       *prometheus.HistogramVec
	ResponseSize      *prometheus.HistogramVec
	ActiveConnections prometheus.Gauge
	HealthStatus      prometheus.Gauge

	registry *prometheus.Registry
	handler  http.Handler
}

func NewMetrics() *Metrics {
	return &Metrics{
		RequestCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint", "status_code"},
		),
		RequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),
		ResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint", "status_code"},
		),
		ActiveConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_active_connections",
				Help: "Number of active HTTP connections",
			},
		),
		HealthStatus: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "app_health_status",
				Help: "Application health status (1 = healthy, 0 = unhealthy)",
			},
		),
	}
}

func (m *Metrics) RecordRequest(method, endpoint string, statusCode int, duration time.Duration, requestSize, responseSize int64) {
	status := strconv.Itoa(statusCode)

	m.RequestCount.WithLabelValues(method, endpoint, status).Inc()
	m.RequestDuration.WithLabelValues(method, endpoint, status).Observe(duration.Seconds())
	m.RequestSize.WithLabelValues(method, endpoint).Observe(float64(requestSize))
	m.ResponseSize.WithLabelValues(method, endpoint, status).Observe(float64(responseSize))
}

func (m *Metrics) SetHealthStatus(healthy bool) {
	if healthy {
		m.HealthStatus.Set(1)
	} else {
		m.HealthStatus.Set(0)
	}
}

func (m *Metrics) Handler() http.Handler {
	if m.handler != nil {
		return m.handler
	}
	return promhttp.Handler()
}

func (m *Metrics) Register() error {
	m.registry = prometheus.NewRegistry()

	// Register all metrics with the custom registry
	if err := m.registry.Register(m.RequestCount); err != nil {
		return err
	}
	if err := m.registry.Register(m.RequestDuration); err != nil {
		return err
	}
	if err := m.registry.Register(m.RequestSize); err != nil {
		return err
	}
	if err := m.registry.Register(m.ResponseSize); err != nil {
		return err
	}
	if err := m.registry.Register(m.ActiveConnections); err != nil {
		return err
	}
	if err := m.registry.Register(m.HealthStatus); err != nil {
		return err
	}

	// Create the handler using our custom registry
	m.handler = promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})

	return nil
}
