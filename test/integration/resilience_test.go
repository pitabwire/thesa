package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/pitabwire/thesa/internal/config"
)

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
