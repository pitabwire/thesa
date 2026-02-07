package observability

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHealth_returnsOK(t *testing.T) {
	// Set build-time variables for test.
	origVersion, origCommit := Version, Commit
	Version = "1.2.3"
	Commit = "abc1234"
	t.Cleanup(func() {
		Version = origVersion
		Commit = origCommit
	})

	handler := HandleHealth()
	req := httptest.NewRequest(http.MethodGet, "/ui/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
	if resp.Version != "1.2.3" {
		t.Errorf("version = %q, want 1.2.3", resp.Version)
	}
	if resp.Commit != "abc1234" {
		t.Errorf("commit = %q, want abc1234", resp.Commit)
	}
}

func TestHandleHealth_defaultValues(t *testing.T) {
	handler := HandleHealth()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/health", nil))

	var resp HealthResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Version == "" {
		t.Error("version should have a default value")
	}
}

func TestHandleReady_allHealthy(t *testing.T) {
	checks := ReadinessChecks{
		DefinitionsLoaded: func() bool { return true },
		OpenAPILoaded:     func() bool { return true },
	}

	handler := HandleReady(checks)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/ready", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp ReadinessResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "ready" {
		t.Errorf("status = %q, want ready", resp.Status)
	}
	if resp.Checks["definitions"].Status != "ok" {
		t.Errorf("definitions = %q, want ok", resp.Checks["definitions"].Status)
	}
	if resp.Checks["openapi_index"].Status != "ok" {
		t.Errorf("openapi_index = %q, want ok", resp.Checks["openapi_index"].Status)
	}
}

func TestHandleReady_definitionsNotLoaded(t *testing.T) {
	checks := ReadinessChecks{
		DefinitionsLoaded: func() bool { return false },
		OpenAPILoaded:     func() bool { return true },
	}

	handler := HandleReady(checks)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/ready", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}

	var resp ReadinessResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "not_ready" {
		t.Errorf("status = %q, want not_ready", resp.Status)
	}
	if resp.Checks["definitions"].Status != "error" {
		t.Errorf("definitions = %q, want error", resp.Checks["definitions"].Status)
	}
	if resp.Checks["definitions"].Error == "" {
		t.Error("definitions error should have a message")
	}
}

func TestHandleReady_openAPINotLoaded(t *testing.T) {
	checks := ReadinessChecks{
		DefinitionsLoaded: func() bool { return true },
		OpenAPILoaded:     func() bool { return false },
	}

	handler := HandleReady(checks)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/ready", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}

	var resp ReadinessResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Checks["openapi_index"].Status != "error" {
		t.Errorf("openapi_index = %q, want error", resp.Checks["openapi_index"].Status)
	}
}

type mockHealthChecker struct {
	err error
}

func (m *mockHealthChecker) HealthCheck(_ context.Context) error {
	return m.err
}

func TestHandleReady_withOptionalChecks_allHealthy(t *testing.T) {
	checks := ReadinessChecks{
		DefinitionsLoaded: func() bool { return true },
		OpenAPILoaded:     func() bool { return true },
		WorkflowStore:     &mockHealthChecker{},
		PolicyEngine:      &mockHealthChecker{},
		IdempotencyStore:  &mockHealthChecker{},
	}

	handler := HandleReady(checks)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/ready", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp ReadinessResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "ready" {
		t.Errorf("status = %q, want ready", resp.Status)
	}
	// Should have 5 checks total.
	if len(resp.Checks) != 5 {
		t.Errorf("checks count = %d, want 5", len(resp.Checks))
	}
	for name, check := range resp.Checks {
		if check.Status != "ok" {
			t.Errorf("%s = %q, want ok", name, check.Status)
		}
	}
}

func TestHandleReady_workflowStoreDown(t *testing.T) {
	checks := ReadinessChecks{
		DefinitionsLoaded: func() bool { return true },
		OpenAPILoaded:     func() bool { return true },
		WorkflowStore:     &mockHealthChecker{err: errors.New("connection refused")},
	}

	handler := HandleReady(checks)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/ready", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}

	var resp ReadinessResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Checks["workflow_store"].Status != "error" {
		t.Errorf("workflow_store = %q, want error", resp.Checks["workflow_store"].Status)
	}
	if resp.Checks["workflow_store"].Error != "connection refused" {
		t.Errorf("workflow_store error = %q, want 'connection refused'", resp.Checks["workflow_store"].Error)
	}
}

func TestHandleReady_policyEngineDown(t *testing.T) {
	checks := ReadinessChecks{
		DefinitionsLoaded: func() bool { return true },
		OpenAPILoaded:     func() bool { return true },
		PolicyEngine:      &mockHealthChecker{err: errors.New("OPA unreachable")},
	}

	handler := HandleReady(checks)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/ready", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}

	var resp ReadinessResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Checks["policy_engine"].Status != "error" {
		t.Errorf("policy_engine = %q, want error", resp.Checks["policy_engine"].Status)
	}
}

func TestHandleReady_idempotencyStoreDown(t *testing.T) {
	checks := ReadinessChecks{
		DefinitionsLoaded: func() bool { return true },
		OpenAPILoaded:     func() bool { return true },
		IdempotencyStore:  &mockHealthChecker{err: errors.New("redis timeout")},
	}

	handler := HandleReady(checks)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/ready", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}

	var resp ReadinessResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Checks["idempotency_store"].Status != "error" {
		t.Errorf("idempotency_store = %q, want error", resp.Checks["idempotency_store"].Status)
	}
}

func TestHandleReady_nilCheckerFunctions(t *testing.T) {
	// When checker functions are nil, definitions and openapi should fail.
	checks := ReadinessChecks{}

	handler := HandleReady(checks)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/ready", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}

	var resp ReadinessResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Checks["definitions"].Status != "error" {
		t.Errorf("definitions = %q, want error", resp.Checks["definitions"].Status)
	}
	if resp.Checks["openapi_index"].Status != "error" {
		t.Errorf("openapi_index = %q, want error", resp.Checks["openapi_index"].Status)
	}
}

func TestHandleReady_checksHaveLatency(t *testing.T) {
	checks := ReadinessChecks{
		DefinitionsLoaded: func() bool { return true },
		OpenAPILoaded:     func() bool { return true },
	}

	handler := HandleReady(checks)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/ready", nil))

	var resp ReadinessResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	// Latency should be non-negative (likely 0 for fast checks).
	for name, check := range resp.Checks {
		if check.LatencyMs < 0 {
			t.Errorf("%s latency = %d, should be >= 0", name, check.LatencyMs)
		}
	}
}

func TestHandleReady_withoutOptionalChecks(t *testing.T) {
	// When optional checkers are nil, only required checks should appear.
	checks := ReadinessChecks{
		DefinitionsLoaded: func() bool { return true },
		OpenAPILoaded:     func() bool { return true },
	}

	handler := HandleReady(checks)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/ready", nil))

	var resp ReadinessResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Checks) != 2 {
		t.Errorf("checks count = %d, want 2 (only required checks)", len(resp.Checks))
	}
	if _, ok := resp.Checks["workflow_store"]; ok {
		t.Error("workflow_store should not be in checks when nil")
	}
	if _, ok := resp.Checks["policy_engine"]; ok {
		t.Error("policy_engine should not be in checks when nil")
	}
	if _, ok := resp.Checks["idempotency_store"]; ok {
		t.Error("idempotency_store should not be in checks when nil")
	}
}

func TestHandleReady_multipleFailures(t *testing.T) {
	checks := ReadinessChecks{
		DefinitionsLoaded: func() bool { return false },
		OpenAPILoaded:     func() bool { return false },
		WorkflowStore:     &mockHealthChecker{err: errors.New("pg down")},
	}

	handler := HandleReady(checks)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/ready", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}

	var resp ReadinessResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	failCount := 0
	for _, check := range resp.Checks {
		if check.Status == "error" {
			failCount++
		}
	}
	if failCount != 3 {
		t.Errorf("failed checks = %d, want 3", failCount)
	}
}
