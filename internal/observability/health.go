package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Build-time variables injected via ldflags.
var (
	Version = "dev"
	Commit  = "unknown"
)

// HealthResponse is the JSON response for the liveness endpoint.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// ReadinessResponse is the JSON response for the readiness endpoint.
type ReadinessResponse struct {
	Status string                   `json:"status"`
	Checks map[string]CheckResult   `json:"checks"`
}

// CheckResult is the result of a single readiness check.
type CheckResult struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// HealthChecker can verify its own health.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// ReadinessChecks holds the dependency checkers for the readiness endpoint.
type ReadinessChecks struct {
	// Required checks — always run.
	DefinitionsLoaded func() bool
	OpenAPILoaded     func() bool

	// Optional checks — only run if non-nil.
	WorkflowStore    HealthChecker
	PolicyEngine     HealthChecker
	IdempotencyStore HealthChecker
}

const checkTimeout = 2 * time.Second

// HandleHealth returns an HTTP handler for the liveness endpoint.
func HandleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HealthResponse{
			Status:  "ok",
			Version: Version,
			Commit:  Commit,
		})
	}
}

// HandleReady returns an HTTP handler for the readiness endpoint.
func HandleReady(checks ReadinessChecks) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results := make(map[string]CheckResult)
		var mu sync.Mutex
		var wg sync.WaitGroup

		record := func(name string, result CheckResult) {
			mu.Lock()
			results[name] = result
			mu.Unlock()
		}

		// Required: definitions loaded.
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			if checks.DefinitionsLoaded != nil && checks.DefinitionsLoaded() {
				record("definitions", CheckResult{
					Status:    "ok",
					LatencyMs: time.Since(start).Milliseconds(),
				})
			} else {
				record("definitions", CheckResult{
					Status:    "error",
					LatencyMs: time.Since(start).Milliseconds(),
					Error:     "no definitions loaded",
				})
			}
		}()

		// Required: OpenAPI index loaded.
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			if checks.OpenAPILoaded != nil && checks.OpenAPILoaded() {
				record("openapi_index", CheckResult{
					Status:    "ok",
					LatencyMs: time.Since(start).Milliseconds(),
				})
			} else {
				record("openapi_index", CheckResult{
					Status:    "error",
					LatencyMs: time.Since(start).Milliseconds(),
					Error:     "no OpenAPI specs loaded",
				})
			}
		}()

		// Optional: workflow store.
		if checks.WorkflowStore != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				record("workflow_store", runCheck(r.Context(), checks.WorkflowStore))
			}()
		}

		// Optional: policy engine.
		if checks.PolicyEngine != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				record("policy_engine", runCheck(r.Context(), checks.PolicyEngine))
			}()
		}

		// Optional: idempotency store.
		if checks.IdempotencyStore != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				record("idempotency_store", runCheck(r.Context(), checks.IdempotencyStore))
			}()
		}

		wg.Wait()

		// Determine overall status.
		status := "ready"
		httpStatus := http.StatusOK
		for _, result := range results {
			if result.Status != "ok" {
				status = "not_ready"
				httpStatus = http.StatusServiceUnavailable
				break
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(httpStatus)
		json.NewEncoder(w).Encode(ReadinessResponse{
			Status: status,
			Checks: results,
		})
	}
}

// runCheck executes a health check with a per-check timeout.
func runCheck(parent context.Context, checker HealthChecker) CheckResult {
	ctx, cancel := context.WithTimeout(parent, checkTimeout)
	defer cancel()

	start := time.Now()
	err := checker.HealthCheck(ctx)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return CheckResult{
			Status:    "error",
			LatencyMs: latency,
			Error:     err.Error(),
		}
	}
	return CheckResult{
		Status:    "ok",
		LatencyMs: latency,
	}
}
