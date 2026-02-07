package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/pitabwire/thesa/internal/config"
)

// ==========================================================================
// Circuit Breaker Tests
// ==========================================================================

func TestResilience_CircuitBreakerTripsOnConsecutiveFailures(t *testing.T) {
	h := NewTestHarness(t,
		WithCircuitBreaker(config.CircuitBreakerConfig{
			FailureThreshold: 3,
			SuccessThreshold: 1,
			Timeout:          30 * time.Second,
		}),
	)
	token := h.GenerateToken(ViewerClaims())

	// Configure backend to return 500 for all requests.
	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(500, map[string]any{"error": "internal error"})

	// Send enough requests to trip the circuit breaker.
	for range 3 {
		h.GET("/ui/search?q=test", token)
	}

	// Reset mock to track the next call.
	callsBefore := len(h.MockBackend("orders-svc").AllRequests("searchOrders"))

	// Next request should fail immediately without hitting backend (circuit open).
	resp := h.GET("/ui/search?q=test", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	// Check that the provider reported as "error" (circuit breaker open).
	meta := result["meta"].(map[string]any)
	providers := meta["providers"].(map[string]any)
	status := providers["orders.search"].(string)
	if status != "error" {
		t.Errorf("provider status = %q, want %q (circuit breaker should be open)", status, "error")
	}

	// Backend should not have received an additional call.
	callsAfter := len(h.MockBackend("orders-svc").AllRequests("searchOrders"))
	if callsAfter != callsBefore {
		t.Errorf("backend received %d additional calls after circuit opened, want 0", callsAfter-callsBefore)
	}
}

func TestResilience_CircuitBreakerRecoveryAfterTimeout(t *testing.T) {
	h := NewTestHarness(t,
		WithCircuitBreaker(config.CircuitBreakerConfig{
			FailureThreshold: 2,
			SuccessThreshold: 1,
			Timeout:          1 * time.Second, // Short timeout for testing.
		}),
	)
	token := h.GenerateToken(ViewerClaims())

	// Trip the circuit breaker with failures.
	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(500, map[string]any{"error": "fail"})

	for range 2 {
		h.GET("/ui/search?q=test", token)
	}

	// Wait for circuit breaker timeout to expire (transitions to half-open).
	time.Sleep(1500 * time.Millisecond)

	// Now configure backend to succeed.
	h.MockBackend("orders-svc").ResetOperation("searchOrders")
	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(200, map[string]any{
			"data": []map[string]any{
				{"id": "ord-1", "order_number": "ORD-001", "customer_name": "Test", "status": "pending"},
			},
		})

	// Probe request should succeed (half-open → closed).
	resp := h.GET("/ui/search?q=test", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	totalCount := int(data["total_count"].(float64))
	if totalCount < 1 {
		t.Error("expected results after circuit breaker recovery")
	}

	meta := result["meta"].(map[string]any)
	providers := meta["providers"].(map[string]any)
	if providers["orders.search"] != "ok" {
		t.Errorf("provider status = %q, want %q after recovery", providers["orders.search"], "ok")
	}
}

func TestResilience_CircuitBreakerFailedProbeReopens(t *testing.T) {
	h := NewTestHarness(t,
		WithCircuitBreaker(config.CircuitBreakerConfig{
			FailureThreshold: 2,
			SuccessThreshold: 1,
			Timeout:          1 * time.Second,
		}),
	)
	token := h.GenerateToken(ViewerClaims())

	// Trip the circuit breaker.
	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(500, map[string]any{"error": "fail"})

	for range 2 {
		h.GET("/ui/search?q=test", token)
	}

	// Wait for half-open transition.
	time.Sleep(1500 * time.Millisecond)

	// Probe also fails → circuit reopens.
	// Backend is still returning 500 (same configured response).
	resp := h.GET("/ui/search?q=test", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	// The probe hit the backend (half-open allows one request).
	meta := result["meta"].(map[string]any)
	providers := meta["providers"].(map[string]any)
	if providers["orders.search"] != "error" {
		t.Errorf("provider status = %q after failed probe, want %q", providers["orders.search"], "error")
	}

	// Immediately after failed probe, circuit should be open again.
	// Next request should not reach backend.
	callsBefore := len(h.MockBackend("orders-svc").AllRequests("searchOrders"))
	h.GET("/ui/search?q=test", token)
	callsAfter := len(h.MockBackend("orders-svc").AllRequests("searchOrders"))

	if callsAfter != callsBefore {
		t.Error("circuit breaker should be open after failed probe, but backend was called")
	}
}

func TestResilience_4xxDoesNotTripCircuitBreaker(t *testing.T) {
	h := NewTestHarness(t,
		WithCircuitBreaker(config.CircuitBreakerConfig{
			FailureThreshold: 2,
			SuccessThreshold: 1,
			Timeout:          30 * time.Second,
		}),
	)
	token := h.GenerateToken(ViewerClaims())

	// Backend returns 400 (client error) — should NOT count as failure.
	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(400, map[string]any{"error": "bad request"})

	// Send many requests — all should still reach backend.
	for range 5 {
		h.GET("/ui/search?q=test", token)
	}

	// All 5 requests should have hit the backend (circuit still closed).
	h.MockBackend("orders-svc").AssertCalled(t, "searchOrders", 5)
}

// ==========================================================================
// Retry Tests
// ==========================================================================

func TestResilience_GETRequestRetriedOn502(t *testing.T) {
	h := NewTestHarness(t,
		WithRetry(config.RetryConfig{
			MaxAttempts:       3,
			BackoffInitial:    10 * time.Millisecond,
			BackoffMultiplier: 1.0,
			BackoffMax:        50 * time.Millisecond,
			IdempotentOnly:    true,
		}),
	)
	token := h.GenerateToken(ViewerClaims())

	// First two calls return 502, third succeeds.
	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(502, map[string]any{"error": "bad gateway"})
	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(502, map[string]any{"error": "bad gateway"})
	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(200, map[string]any{
			"data": []map[string]any{
				{"id": "ord-1", "order_number": "ORD-001", "customer_name": "Test", "status": "pending"},
			},
		})

	resp := h.GET("/ui/search?q=test", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	totalCount := int(data["total_count"].(float64))
	if totalCount != 1 {
		t.Errorf("total_count = %d, want 1 (should have retried to success)", totalCount)
	}

	// Backend should have been called 3 times (2 retries + 1 success).
	h.MockBackend("orders-svc").AssertCalled(t, "searchOrders", 3)
}

func TestResilience_POSTNotRetriedWhenIdempotentOnly(t *testing.T) {
	h := NewTestHarness(t,
		WithRetry(config.RetryConfig{
			MaxAttempts:       3,
			BackoffInitial:    10 * time.Millisecond,
			BackoffMultiplier: 1.0,
			BackoffMax:        50 * time.Millisecond,
			IdempotentOnly:    true,
		}),
	)
	token := h.GenerateToken(ManagerClaims())

	// Backend returns 502 on cancel (POST operation).
	h.MockBackend("orders-svc").OnOperation("cancelOrder").
		RespondWith(502, map[string]any{"error": "bad gateway"})

	resp := h.POST("/ui/commands/orders.cancel", map[string]any{
		"input": map[string]any{
			"id":     "ord-1",
			"reason": "test",
		},
	}, token)

	// Should get an error (not retried since POST + idempotent_only).
	// The command executor may wrap the 502 into a different error code.
	if resp.StatusCode == http.StatusOK {
		t.Error("expected error response for 502 backend, got 200 OK")
	}

	// The critical assertion: backend should have been called only once (no retries for POST).
	h.MockBackend("orders-svc").AssertCalled(t, "cancelOrder", 1)
}

func TestResilience_RetryStopsWhenCircuitBreakerOpens(t *testing.T) {
	h := NewTestHarness(t,
		WithCircuitBreaker(config.CircuitBreakerConfig{
			FailureThreshold: 2,
			SuccessThreshold: 1,
			Timeout:          30 * time.Second,
		}),
		WithRetry(config.RetryConfig{
			MaxAttempts:       5,
			BackoffInitial:    10 * time.Millisecond,
			BackoffMultiplier: 1.0,
			BackoffMax:        50 * time.Millisecond,
			IdempotentOnly:    false,
		}),
	)
	token := h.GenerateToken(ViewerClaims())

	// Backend always fails.
	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(500, map[string]any{"error": "fail"})

	// First request: retries until circuit breaker trips (2 failures).
	h.GET("/ui/search?q=test", token)

	// Backend should have been called exactly 2 times (circuit breaker threshold),
	// not 5 times (max_attempts), because retries stop when circuit opens.
	callCount := len(h.MockBackend("orders-svc").AllRequests("searchOrders"))
	if callCount > 2 {
		t.Errorf("backend called %d times, expected <= 2 (circuit breaker should stop retries)", callCount)
	}
}

// ==========================================================================
// Timeout Tests
// ==========================================================================

func TestResilience_BackendTimeout_504(t *testing.T) {
	h := NewTestHarness(t,
		WithServiceTimeout(500*time.Millisecond),
		WithHandlerTimeout(3*time.Second),
	)
	token := h.GenerateToken(ViewerClaims())

	// Backend delays longer than service timeout.
	h.MockBackend("orders-svc").OnOperation("listOrders").
		RespondWithDelay(2*time.Second, 200, OrderListFixture(nil, 0))

	resp := h.GET("/ui/pages/orders.list/data", token)

	// Backend timeout should result in an error response (not 200).
	if resp.StatusCode == http.StatusOK {
		t.Error("expected error for backend timeout, got 200 OK")
	}

	// Acceptable statuses: 504 (BACKEND_TIMEOUT) or 500 (INTERNAL_ERROR wrapping timeout).
	if resp.StatusCode != http.StatusGatewayTimeout && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 504 or 500", resp.StatusCode)
	}
}

func TestResilience_HandlerTimeout_TerminatesSlowRequest(t *testing.T) {
	h := NewTestHarness(t,
		WithHandlerTimeout(500*time.Millisecond),
	)
	token := h.GenerateToken(ViewerClaims())

	// Backend delays longer than handler timeout.
	h.MockBackend("orders-svc").OnOperation("listOrders").
		RespondWithDelay(3*time.Second, 200, OrderListFixture(nil, 0))

	resp := h.GET("/ui/pages/orders.list/data", token)

	// Handler timeout should have terminated the request.
	// The exact status may vary (504 or context cancellation error).
	if resp.StatusCode == http.StatusOK {
		t.Error("expected timeout error, got 200 OK")
	}
}

func TestResilience_FastBackend_NoTimeout(t *testing.T) {
	h := NewTestHarness(t,
		WithServiceTimeout(5*time.Second),
		WithHandlerTimeout(10*time.Second),
	)
	token := h.GenerateToken(ViewerClaims())

	h.MockBackend("orders-svc").OnOperation("listOrders").
		RespondWith(200, OrderListFixture([]map[string]any{
			OrderFixture("ord-1", "ORD-001", "pending"),
		}, 1))

	resp := h.GET("/ui/pages/orders.list/data", token)
	h.AssertStatus(t, resp, http.StatusOK)
}
