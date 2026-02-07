package observability

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Histogram bucket definitions.
var (
	httpDurationBuckets    = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	backendDurationBuckets = []float64{0.005, 0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}
	bodySizeBuckets        = []float64{100, 1024, 10240, 102400, 1048576}
)

// Metrics holds all Prometheus metric instruments for the BFF.
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal        *prometheus.CounterVec
	HTTPRequestDuration      *prometheus.HistogramVec
	HTTPRequestSizeBytes     *prometheus.HistogramVec
	HTTPResponseSizeBytes    *prometheus.HistogramVec

	// Command metrics
	CommandExecutionsTotal       *prometheus.CounterVec
	CommandDuration              *prometheus.HistogramVec
	CommandValidationFailures    *prometheus.CounterVec

	// Workflow metrics
	WorkflowStartsTotal       *prometheus.CounterVec
	WorkflowAdvancesTotal      *prometheus.CounterVec
	WorkflowCompletionsTotal   *prometheus.CounterVec
	WorkflowActiveInstances    *prometheus.GaugeVec
	WorkflowStepDuration       *prometheus.HistogramVec
	WorkflowTimeoutsTotal      *prometheus.CounterVec

	// Backend invocation metrics
	BackendRequestsTotal        *prometheus.CounterVec
	BackendRequestDuration      *prometheus.HistogramVec
	BackendCircuitBreakerState  *prometheus.GaugeVec
	BackendRetriesTotal         *prometheus.CounterVec

	// Cache metrics
	CapabilityCacheHitsTotal   prometheus.Counter
	CapabilityCacheMissesTotal prometheus.Counter
	LookupCacheHitsTotal       *prometheus.CounterVec
	LookupCacheMissesTotal     *prometheus.CounterVec

	// System metrics
	DefinitionReloadTotal      *prometheus.CounterVec
	DefinitionsLoaded          prometheus.Gauge
	OpenAPIOperationsIndexed   *prometheus.GaugeVec
	SearchDuration             prometheus.Histogram
	SearchProvidersResponded   prometheus.Histogram
}

// InitMetrics creates and registers all Prometheus metric instruments.
func InitMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		// HTTP
		HTTPRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "thesa_http_requests_total",
			Help: "Total number of HTTP requests.",
		}, []string{"method", "path_pattern", "status"}),
		HTTPRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "thesa_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: httpDurationBuckets,
		}, []string{"method", "path_pattern"}),
		HTTPRequestSizeBytes: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "thesa_http_request_size_bytes",
			Help:    "HTTP request body size in bytes.",
			Buckets: bodySizeBuckets,
		}, []string{"method", "path_pattern"}),
		HTTPResponseSizeBytes: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "thesa_http_response_size_bytes",
			Help:    "HTTP response body size in bytes.",
			Buckets: bodySizeBuckets,
		}, []string{"method", "path_pattern"}),

		// Commands
		CommandExecutionsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "thesa_command_executions_total",
			Help: "Total number of command executions.",
		}, []string{"command_id", "status"}),
		CommandDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "thesa_command_duration_seconds",
			Help:    "Command execution duration in seconds.",
			Buckets: backendDurationBuckets,
		}, []string{"command_id"}),
		CommandValidationFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "thesa_command_validation_failures_total",
			Help: "Total number of command validation failures.",
		}, []string{"command_id"}),

		// Workflows
		WorkflowStartsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "thesa_workflow_starts_total",
			Help: "Total number of workflow starts.",
		}, []string{"workflow_id"}),
		WorkflowAdvancesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "thesa_workflow_advances_total",
			Help: "Total number of workflow advances.",
		}, []string{"workflow_id", "step_id", "event"}),
		WorkflowCompletionsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "thesa_workflow_completions_total",
			Help: "Total number of workflow completions.",
		}, []string{"workflow_id", "final_status"}),
		WorkflowActiveInstances: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "thesa_workflow_active_instances",
			Help: "Number of active workflow instances.",
		}, []string{"workflow_id"}),
		WorkflowStepDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "thesa_workflow_step_duration_seconds",
			Help:    "Workflow step duration in seconds.",
			Buckets: backendDurationBuckets,
		}, []string{"workflow_id", "step_id"}),
		WorkflowTimeoutsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "thesa_workflow_timeouts_total",
			Help: "Total number of workflow timeouts.",
		}, []string{"workflow_id"}),

		// Backend
		BackendRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "thesa_backend_requests_total",
			Help: "Total number of backend service requests.",
		}, []string{"service_id", "operation_id", "status"}),
		BackendRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "thesa_backend_request_duration_seconds",
			Help:    "Backend request duration in seconds.",
			Buckets: backendDurationBuckets,
		}, []string{"service_id"}),
		BackendCircuitBreakerState: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "thesa_backend_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=half-open, 2=open).",
		}, []string{"service_id"}),
		BackendRetriesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "thesa_backend_retries_total",
			Help: "Total number of backend request retries.",
		}, []string{"service_id"}),

		// Cache
		CapabilityCacheHitsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "thesa_capability_cache_hits_total",
			Help: "Total capability cache hits.",
		}),
		CapabilityCacheMissesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "thesa_capability_cache_misses_total",
			Help: "Total capability cache misses.",
		}),
		LookupCacheHitsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "thesa_lookup_cache_hits_total",
			Help: "Total lookup cache hits.",
		}, []string{"lookup_id"}),
		LookupCacheMissesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "thesa_lookup_cache_misses_total",
			Help: "Total lookup cache misses.",
		}, []string{"lookup_id"}),

		// System
		DefinitionReloadTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "thesa_definition_reload_total",
			Help: "Total definition reloads.",
		}, []string{"status"}),
		DefinitionsLoaded: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "thesa_definitions_loaded",
			Help: "Number of loaded definitions.",
		}),
		OpenAPIOperationsIndexed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "thesa_openapi_operations_indexed",
			Help: "Number of indexed OpenAPI operations.",
		}, []string{"service_id"}),
		SearchDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "thesa_search_duration_seconds",
			Help:    "Search execution duration in seconds.",
			Buckets: backendDurationBuckets,
		}),
		SearchProvidersResponded: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "thesa_search_providers_responded",
			Help:    "Number of search providers that responded.",
			Buckets: []float64{1, 2, 3, 5, 10},
		}),
	}

	reg.MustRegister(
		// HTTP
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPRequestSizeBytes,
		m.HTTPResponseSizeBytes,
		// Commands
		m.CommandExecutionsTotal,
		m.CommandDuration,
		m.CommandValidationFailures,
		// Workflows
		m.WorkflowStartsTotal,
		m.WorkflowAdvancesTotal,
		m.WorkflowCompletionsTotal,
		m.WorkflowActiveInstances,
		m.WorkflowStepDuration,
		m.WorkflowTimeoutsTotal,
		// Backend
		m.BackendRequestsTotal,
		m.BackendRequestDuration,
		m.BackendCircuitBreakerState,
		m.BackendRetriesTotal,
		// Cache
		m.CapabilityCacheHitsTotal,
		m.CapabilityCacheMissesTotal,
		m.LookupCacheHitsTotal,
		m.LookupCacheMissesTotal,
		// System
		m.DefinitionReloadTotal,
		m.DefinitionsLoaded,
		m.OpenAPIOperationsIndexed,
		m.SearchDuration,
		m.SearchProvidersResponded,
	)

	return m
}

// --- Recording helpers ---

// RecordHTTPRequest records HTTP request metrics.
func (m *Metrics) RecordHTTPRequest(method, pathPattern string, status int, duration time.Duration, reqSize, respSize int) {
	statusStr := strconv.Itoa(status)
	m.HTTPRequestsTotal.WithLabelValues(method, pathPattern, statusStr).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, pathPattern).Observe(duration.Seconds())
	m.HTTPRequestSizeBytes.WithLabelValues(method, pathPattern).Observe(float64(reqSize))
	m.HTTPResponseSizeBytes.WithLabelValues(method, pathPattern).Observe(float64(respSize))
}

// RecordCommandExecution records command execution metrics.
func (m *Metrics) RecordCommandExecution(commandID, status string, duration time.Duration) {
	m.CommandExecutionsTotal.WithLabelValues(commandID, status).Inc()
	m.CommandDuration.WithLabelValues(commandID).Observe(duration.Seconds())
}

// RecordCommandValidationFailure records a command validation failure.
func (m *Metrics) RecordCommandValidationFailure(commandID string) {
	m.CommandValidationFailures.WithLabelValues(commandID).Inc()
}

// RecordWorkflowStart records a workflow start.
func (m *Metrics) RecordWorkflowStart(workflowID string) {
	m.WorkflowStartsTotal.WithLabelValues(workflowID).Inc()
	m.WorkflowActiveInstances.WithLabelValues(workflowID).Inc()
}

// RecordWorkflowAdvance records a workflow advance.
func (m *Metrics) RecordWorkflowAdvance(workflowID, stepID, event string) {
	m.WorkflowAdvancesTotal.WithLabelValues(workflowID, stepID, event).Inc()
}

// RecordWorkflowCompletion records a workflow completion.
func (m *Metrics) RecordWorkflowCompletion(workflowID, finalStatus string) {
	m.WorkflowCompletionsTotal.WithLabelValues(workflowID, finalStatus).Inc()
	m.WorkflowActiveInstances.WithLabelValues(workflowID).Dec()
}

// RecordWorkflowStepDuration records the duration of a workflow step.
func (m *Metrics) RecordWorkflowStepDuration(workflowID, stepID string, duration time.Duration) {
	m.WorkflowStepDuration.WithLabelValues(workflowID, stepID).Observe(duration.Seconds())
}

// RecordWorkflowTimeout records a workflow timeout.
func (m *Metrics) RecordWorkflowTimeout(workflowID string) {
	m.WorkflowTimeoutsTotal.WithLabelValues(workflowID).Inc()
}

// RecordBackendRequest records a backend service request.
func (m *Metrics) RecordBackendRequest(serviceID, operationID string, status int, duration time.Duration) {
	m.BackendRequestsTotal.WithLabelValues(serviceID, operationID, strconv.Itoa(status)).Inc()
	m.BackendRequestDuration.WithLabelValues(serviceID).Observe(duration.Seconds())
}

// SetBackendCircuitBreakerState sets the circuit breaker state for a service.
// State: 0=closed, 1=half-open, 2=open.
func (m *Metrics) SetBackendCircuitBreakerState(serviceID string, state float64) {
	m.BackendCircuitBreakerState.WithLabelValues(serviceID).Set(state)
}

// RecordBackendRetry records a backend request retry.
func (m *Metrics) RecordBackendRetry(serviceID string) {
	m.BackendRetriesTotal.WithLabelValues(serviceID).Inc()
}

// RecordCapabilityCacheHit records a capability cache hit.
func (m *Metrics) RecordCapabilityCacheHit() {
	m.CapabilityCacheHitsTotal.Inc()
}

// RecordCapabilityCacheMiss records a capability cache miss.
func (m *Metrics) RecordCapabilityCacheMiss() {
	m.CapabilityCacheMissesTotal.Inc()
}

// RecordLookupCacheHit records a lookup cache hit.
func (m *Metrics) RecordLookupCacheHit(lookupID string) {
	m.LookupCacheHitsTotal.WithLabelValues(lookupID).Inc()
}

// RecordLookupCacheMiss records a lookup cache miss.
func (m *Metrics) RecordLookupCacheMiss(lookupID string) {
	m.LookupCacheMissesTotal.WithLabelValues(lookupID).Inc()
}

// RecordDefinitionReload records a definition reload.
func (m *Metrics) RecordDefinitionReload(status string) {
	m.DefinitionReloadTotal.WithLabelValues(status).Inc()
}

// SetDefinitionsLoaded sets the number of loaded definitions.
func (m *Metrics) SetDefinitionsLoaded(count float64) {
	m.DefinitionsLoaded.Set(count)
}

// SetOpenAPIOperationsIndexed sets the number of indexed OpenAPI operations.
func (m *Metrics) SetOpenAPIOperationsIndexed(serviceID string, count float64) {
	m.OpenAPIOperationsIndexed.WithLabelValues(serviceID).Set(count)
}

// RecordSearch records a search execution.
func (m *Metrics) RecordSearch(duration time.Duration, providersResponded int) {
	m.SearchDuration.Observe(duration.Seconds())
	m.SearchProvidersResponded.Observe(float64(providersResponded))
}

// --- HTTP Middleware ---

// MetricsMiddleware returns HTTP middleware that records request metrics using
// chi's route pattern (not the actual URL path) to avoid label cardinality
// explosion.
func (m *Metrics) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &metricsResponseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		duration := time.Since(start)
		pathPattern := routePattern(r)
		reqSize := 0
		if r.ContentLength > 0 {
			reqSize = int(r.ContentLength)
		}

		m.RecordHTTPRequest(r.Method, pathPattern, sw.status, duration, reqSize, sw.bytes)
	})
}

// Handler returns the Prometheus HTTP handler for the /metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}

// routePattern extracts chi's route pattern from the request context.
// Falls back to the raw URL path if no pattern is found.
func routePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return r.URL.Path
	}
	pattern := strings.Join(rctx.RoutePatterns, "")
	// chi route patterns have trailing /*, remove it.
	pattern = strings.TrimSuffix(pattern, "/*")
	if pattern == "" {
		return r.URL.Path
	}
	return pattern
}

// metricsResponseWriter wraps http.ResponseWriter to capture status and bytes.
type metricsResponseWriter struct {
	http.ResponseWriter
	status  int
	bytes   int
	written bool
}

func (w *metricsResponseWriter) WriteHeader(code int) {
	if !w.written {
		w.status = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.written = true
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}
