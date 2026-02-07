package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func newTestMetrics(t *testing.T) (*Metrics, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	m := InitMetrics(reg)
	return m, reg
}

func TestInitMetrics_registersAllMetrics(t *testing.T) {
	m, reg := newTestMetrics(t)
	if m == nil {
		t.Fatal("InitMetrics returned nil")
	}

	// Gather all registered metric families.
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}

	// Verify expected metric names are registered.
	names := make(map[string]bool)
	for _, f := range families {
		names[f.GetName()] = true
	}

	expected := []string{
		"thesa_http_requests_total",
		"thesa_http_request_duration_seconds",
		"thesa_http_request_size_bytes",
		"thesa_http_response_size_bytes",
		"thesa_command_executions_total",
		"thesa_command_duration_seconds",
		"thesa_command_validation_failures_total",
		"thesa_workflow_starts_total",
		"thesa_workflow_advances_total",
		"thesa_workflow_completions_total",
		"thesa_workflow_active_instances",
		"thesa_workflow_step_duration_seconds",
		"thesa_workflow_timeouts_total",
		"thesa_backend_requests_total",
		"thesa_backend_request_duration_seconds",
		"thesa_backend_circuit_breaker_state",
		"thesa_backend_retries_total",
		"thesa_capability_cache_hits_total",
		"thesa_capability_cache_misses_total",
		"thesa_lookup_cache_hits_total",
		"thesa_lookup_cache_misses_total",
		"thesa_definition_reload_total",
		"thesa_definitions_loaded",
		"thesa_openapi_operations_indexed",
		"thesa_search_duration_seconds",
		"thesa_search_providers_responded",
	}

	// Record a value for each metric so they appear in Gather.
	m.RecordHTTPRequest("GET", "/test", 200, time.Millisecond, 0, 100)
	m.RecordCommandExecution("cmd-1", "success", time.Millisecond)
	m.RecordCommandValidationFailure("cmd-1")
	m.RecordWorkflowStart("wf-1")
	m.RecordWorkflowAdvance("wf-1", "step-1", "submitted")
	m.RecordWorkflowCompletion("wf-1", "completed")
	m.RecordWorkflowStepDuration("wf-1", "step-1", time.Millisecond)
	m.RecordWorkflowTimeout("wf-1")
	m.RecordBackendRequest("svc-1", "op-1", 200, time.Millisecond)
	m.SetBackendCircuitBreakerState("svc-1", 0)
	m.RecordBackendRetry("svc-1")
	m.RecordCapabilityCacheHit()
	m.RecordCapabilityCacheMiss()
	m.RecordLookupCacheHit("lu-1")
	m.RecordLookupCacheMiss("lu-1")
	m.RecordDefinitionReload("success")
	m.SetDefinitionsLoaded(5)
	m.SetOpenAPIOperationsIndexed("svc-1", 10)
	m.RecordSearch(time.Millisecond, 3)

	families, err = reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	names = make(map[string]bool)
	for _, f := range families {
		names[f.GetName()] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("metric %q not registered", name)
		}
	}
}

func TestRecordHTTPRequest(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.RecordHTTPRequest("GET", "/ui/pages/{pageId}", 200, 50*time.Millisecond, 0, 1024)
	m.RecordHTTPRequest("GET", "/ui/pages/{pageId}", 200, 100*time.Millisecond, 0, 2048)
	m.RecordHTTPRequest("POST", "/ui/commands/{commandId}", 500, 200*time.Millisecond, 512, 256)

	// Verify counter values.
	val := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("GET", "/ui/pages/{pageId}", "200"))
	if val != 2 {
		t.Errorf("GET requests = %v, want 2", val)
	}
	val = testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("POST", "/ui/commands/{commandId}", "500"))
	if val != 1 {
		t.Errorf("POST requests = %v, want 1", val)
	}
}

func TestRecordCommandExecution(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.RecordCommandExecution("create-order", "success", 150*time.Millisecond)
	m.RecordCommandExecution("create-order", "failure", 50*time.Millisecond)

	success := testutil.ToFloat64(m.CommandExecutionsTotal.WithLabelValues("create-order", "success"))
	if success != 1 {
		t.Errorf("success count = %v, want 1", success)
	}
	failure := testutil.ToFloat64(m.CommandExecutionsTotal.WithLabelValues("create-order", "failure"))
	if failure != 1 {
		t.Errorf("failure count = %v, want 1", failure)
	}
}

func TestRecordCommandValidationFailure(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.RecordCommandValidationFailure("create-order")
	m.RecordCommandValidationFailure("create-order")

	val := testutil.ToFloat64(m.CommandValidationFailures.WithLabelValues("create-order"))
	if val != 2 {
		t.Errorf("validation failures = %v, want 2", val)
	}
}

func TestRecordWorkflowLifecycle(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.RecordWorkflowStart("onboarding")
	active := testutil.ToFloat64(m.WorkflowActiveInstances.WithLabelValues("onboarding"))
	if active != 1 {
		t.Errorf("active instances = %v, want 1", active)
	}

	m.RecordWorkflowAdvance("onboarding", "step-1", "submitted")
	advances := testutil.ToFloat64(m.WorkflowAdvancesTotal.WithLabelValues("onboarding", "step-1", "submitted"))
	if advances != 1 {
		t.Errorf("advances = %v, want 1", advances)
	}

	m.RecordWorkflowCompletion("onboarding", "completed")
	active = testutil.ToFloat64(m.WorkflowActiveInstances.WithLabelValues("onboarding"))
	if active != 0 {
		t.Errorf("active instances after completion = %v, want 0", active)
	}

	completions := testutil.ToFloat64(m.WorkflowCompletionsTotal.WithLabelValues("onboarding", "completed"))
	if completions != 1 {
		t.Errorf("completions = %v, want 1", completions)
	}
}

func TestRecordWorkflowStepDuration(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.RecordWorkflowStepDuration("onboarding", "step-1", 500*time.Millisecond)

	count := testutil.CollectAndCount(m.WorkflowStepDuration)
	if count == 0 {
		t.Error("expected workflow step duration histogram to have observations")
	}
}

func TestRecordWorkflowTimeout(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.RecordWorkflowTimeout("onboarding")
	val := testutil.ToFloat64(m.WorkflowTimeoutsTotal.WithLabelValues("onboarding"))
	if val != 1 {
		t.Errorf("timeouts = %v, want 1", val)
	}
}

func TestRecordBackendRequest(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.RecordBackendRequest("order-service", "createOrder", 201, 100*time.Millisecond)

	val := testutil.ToFloat64(m.BackendRequestsTotal.WithLabelValues("order-service", "createOrder", "201"))
	if val != 1 {
		t.Errorf("backend requests = %v, want 1", val)
	}
}

func TestSetBackendCircuitBreakerState(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.SetBackendCircuitBreakerState("order-service", 0)
	val := testutil.ToFloat64(m.BackendCircuitBreakerState.WithLabelValues("order-service"))
	if val != 0 {
		t.Errorf("circuit breaker state = %v, want 0 (closed)", val)
	}

	m.SetBackendCircuitBreakerState("order-service", 2)
	val = testutil.ToFloat64(m.BackendCircuitBreakerState.WithLabelValues("order-service"))
	if val != 2 {
		t.Errorf("circuit breaker state = %v, want 2 (open)", val)
	}
}

func TestRecordBackendRetry(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.RecordBackendRetry("order-service")
	m.RecordBackendRetry("order-service")
	val := testutil.ToFloat64(m.BackendRetriesTotal.WithLabelValues("order-service"))
	if val != 2 {
		t.Errorf("retries = %v, want 2", val)
	}
}

func TestRecordCapabilityCache(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.RecordCapabilityCacheHit()
	m.RecordCapabilityCacheHit()
	m.RecordCapabilityCacheMiss()

	hits := testutil.ToFloat64(m.CapabilityCacheHitsTotal)
	if hits != 2 {
		t.Errorf("cache hits = %v, want 2", hits)
	}
	misses := testutil.ToFloat64(m.CapabilityCacheMissesTotal)
	if misses != 1 {
		t.Errorf("cache misses = %v, want 1", misses)
	}
}

func TestRecordLookupCache(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.RecordLookupCacheHit("countries")
	m.RecordLookupCacheMiss("countries")

	hits := testutil.ToFloat64(m.LookupCacheHitsTotal.WithLabelValues("countries"))
	if hits != 1 {
		t.Errorf("lookup hits = %v, want 1", hits)
	}
	misses := testutil.ToFloat64(m.LookupCacheMissesTotal.WithLabelValues("countries"))
	if misses != 1 {
		t.Errorf("lookup misses = %v, want 1", misses)
	}
}

func TestRecordDefinitionReload(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.RecordDefinitionReload("success")
	m.RecordDefinitionReload("failure")

	success := testutil.ToFloat64(m.DefinitionReloadTotal.WithLabelValues("success"))
	if success != 1 {
		t.Errorf("reload success = %v, want 1", success)
	}
	failure := testutil.ToFloat64(m.DefinitionReloadTotal.WithLabelValues("failure"))
	if failure != 1 {
		t.Errorf("reload failure = %v, want 1", failure)
	}
}

func TestSetDefinitionsLoaded(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.SetDefinitionsLoaded(5)
	val := testutil.ToFloat64(m.DefinitionsLoaded)
	if val != 5 {
		t.Errorf("definitions loaded = %v, want 5", val)
	}

	m.SetDefinitionsLoaded(10)
	val = testutil.ToFloat64(m.DefinitionsLoaded)
	if val != 10 {
		t.Errorf("definitions loaded = %v, want 10", val)
	}
}

func TestSetOpenAPIOperationsIndexed(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.SetOpenAPIOperationsIndexed("order-service", 25)
	val := testutil.ToFloat64(m.OpenAPIOperationsIndexed.WithLabelValues("order-service"))
	if val != 25 {
		t.Errorf("operations indexed = %v, want 25", val)
	}
}

func TestRecordSearch(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.RecordSearch(200*time.Millisecond, 3)

	count := testutil.CollectAndCount(m.SearchDuration)
	if count == 0 {
		t.Error("expected search duration histogram to have observations")
	}
	count = testutil.CollectAndCount(m.SearchProvidersResponded)
	if count == 0 {
		t.Error("expected search providers histogram to have observations")
	}
}

func TestMetricsMiddleware_recordsRequestMetrics(t *testing.T) {
	m, _ := newTestMetrics(t)

	// Build a chi router so route patterns are captured.
	r := chi.NewRouter()
	r.Use(m.MetricsMiddleware)
	r.Get("/ui/pages/{pageId}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/ui/pages/dashboard", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	// Verify metrics were recorded with the route pattern, not the actual path.
	val := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("GET", "/ui/pages/{pageId}", "200"))
	if val != 1 {
		t.Errorf("requests total = %v, want 1", val)
	}
}

func TestMetricsMiddleware_capturesResponseSize(t *testing.T) {
	m, _ := newTestMetrics(t)

	r := chi.NewRouter()
	r.Use(m.MetricsMiddleware)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("healthy"))
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// Response size should have been recorded.
	count := testutil.CollectAndCount(m.HTTPResponseSizeBytes)
	if count == 0 {
		t.Error("expected response size histogram to have observations")
	}
}

func TestMetricsMiddleware_capturesStatusCode(t *testing.T) {
	m, _ := newTestMetrics(t)

	r := chi.NewRouter()
	r.Use(m.MetricsMiddleware)
	r.Post("/ui/commands/{commandId}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	req := httptest.NewRequest(http.MethodPost, "/ui/commands/do-stuff", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	val := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("POST", "/ui/commands/{commandId}", "400"))
	if val != 1 {
		t.Errorf("400 requests = %v, want 1", val)
	}
}

func TestMetricsMiddleware_fallsBackToPath(t *testing.T) {
	m, _ := newTestMetrics(t)

	// Use middleware directly without chi router.
	handler := m.MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/raw/path", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Without chi, should fall back to raw path.
	val := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("GET", "/raw/path", "200"))
	if val != 1 {
		t.Errorf("raw path requests = %v, want 1", val)
	}
}

func TestHandler_servesMetrics(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	// Prometheus handler should return at least go runtime metrics.
	if !strings.Contains(body, "go_") {
		t.Error("metrics response should contain go runtime metrics")
	}
}

func TestHistogramBuckets(t *testing.T) {
	// Verify bucket configurations are correct.
	if len(httpDurationBuckets) != 11 {
		t.Errorf("httpDurationBuckets length = %d, want 11", len(httpDurationBuckets))
	}
	if len(backendDurationBuckets) != 9 {
		t.Errorf("backendDurationBuckets length = %d, want 9", len(backendDurationBuckets))
	}
	if len(bodySizeBuckets) != 5 {
		t.Errorf("bodySizeBuckets length = %d, want 5", len(bodySizeBuckets))
	}

	// Verify buckets are sorted ascending.
	for i := 1; i < len(httpDurationBuckets); i++ {
		if httpDurationBuckets[i] <= httpDurationBuckets[i-1] {
			t.Errorf("httpDurationBuckets not sorted at index %d", i)
		}
	}
}
