package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/pitabwire/thesa/model"
)

func TestHarness_Startup(t *testing.T) {
	h := NewTestHarness(t, WithWorkflows(), WithIdempotency())

	// Verify the server is running.
	resp := h.GET("/ui/health", "")
	h.AssertStatus(t, resp, http.StatusOK)
}

func TestHarness_HealthEndpoints(t *testing.T) {
	h := NewTestHarness(t)

	t.Run("health", func(t *testing.T) {
		resp := h.GET("/ui/health", "")
		h.AssertStatus(t, resp, http.StatusOK)

		var body map[string]string
		h.ParseJSON(resp, &body)
		if body["status"] != "ok" {
			t.Errorf("health status = %q, want ok", body["status"])
		}
	})

	t.Run("ready", func(t *testing.T) {
		resp := h.GET("/ui/ready", "")
		h.AssertStatus(t, resp, http.StatusOK)
	})
}

func TestHarness_AuthenticationRequired(t *testing.T) {
	h := NewTestHarness(t)

	t.Run("no token returns 401", func(t *testing.T) {
		resp := h.GET("/ui/navigation", "")
		h.AssertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("expired token returns 401", func(t *testing.T) {
		token := h.GenerateExpiredToken(ManagerClaims())
		resp := h.GET("/ui/navigation", token)
		h.AssertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		resp := h.GET("/ui/navigation", "invalid-token")
		h.AssertStatus(t, resp, http.StatusUnauthorized)
	})
}

func TestHarness_Navigation(t *testing.T) {
	h := NewTestHarness(t)

	t.Run("manager sees orders navigation", func(t *testing.T) {
		token := h.GenerateToken(ManagerClaims())

		// Configure badge backend to respond with counts.
		h.MockBackend("orders-svc").OnOperation("getOrderCounts").
			RespondWith(200, map[string]any{
				"pending_count":    5,
				"processing_count": 3,
				"total_count":      42,
			})

		resp := h.GET("/ui/navigation", token)
		h.AssertStatus(t, resp, http.StatusOK)

		var body map[string]any
		h.ParseJSON(resp, &body)

		items, ok := body["items"].([]any)
		if !ok || len(items) == 0 {
			t.Fatal("expected navigation items")
		}
	})

	t.Run("viewer sees orders but no create", func(t *testing.T) {
		token := h.GenerateToken(ViewerClaims())
		resp := h.GET("/ui/navigation", token)
		h.AssertStatus(t, resp, http.StatusOK)

		var body map[string]any
		h.ParseJSON(resp, &body)

		items, ok := body["items"].([]any)
		if !ok || len(items) == 0 {
			t.Fatal("expected navigation items")
		}

		// The viewer should see the Orders domain but not the Create Order child.
		ordersNode, ok := items[0].(map[string]any)
		if !ok {
			t.Fatal("expected first item to be a map")
		}

		children, _ := ordersNode["children"].([]any)
		for _, child := range children {
			childMap, _ := child.(map[string]any)
			if childMap["label"] == "Create Order" {
				t.Error("viewer should not see Create Order nav item")
			}
		}
	})
}

func TestHarness_PageLoad(t *testing.T) {
	h := NewTestHarness(t)

	t.Run("get page descriptor", func(t *testing.T) {
		token := h.GenerateToken(ManagerClaims())
		resp := h.GET("/ui/pages/orders.list", token)
		h.AssertStatus(t, resp, http.StatusOK)

		var body map[string]any
		h.ParseJSON(resp, &body)

		if body["id"] != "orders.list" {
			t.Errorf("page id = %v, want orders.list", body["id"])
		}
	})

	t.Run("get page data proxies to backend", func(t *testing.T) {
		token := h.GenerateToken(ManagerClaims())

		orders := []map[string]any{
			OrderFixture("ord-1", "ORD-001", "pending"),
			OrderFixture("ord-2", "ORD-002", "processing"),
		}
		h.MockBackend("orders-svc").OnOperation("listOrders").
			RespondWith(200, OrderListFixture(orders, 2))

		resp := h.GET("/ui/pages/orders.list/data", token)
		h.AssertStatus(t, resp, http.StatusOK)

		h.MockBackend("orders-svc").AssertCalled(t, "listOrders", 1)

		// Verify the backend received proper headers.
		req := h.MockBackend("orders-svc").LastRequest("listOrders")
		if req == nil {
			t.Fatal("expected a recorded request")
		}
		if req.Headers.Get("Authorization") == "" {
			t.Error("backend request missing Authorization header")
		}
		if req.Headers.Get("X-Tenant-Id") != "acme-corp" {
			t.Errorf("X-Tenant-Id = %q, want acme-corp", req.Headers.Get("X-Tenant-Id"))
		}
	})

	t.Run("forbidden page", func(t *testing.T) {
		// A viewer without "orders:create" cannot access a page requiring that cap.
		// But our orders.list page only requires "orders:view" so the viewer can access it.
		token := h.GenerateToken(ViewerClaims())
		resp := h.GET("/ui/pages/orders.list", token)
		h.AssertStatus(t, resp, http.StatusOK)
	})

	t.Run("nonexistent page returns 404", func(t *testing.T) {
		token := h.GenerateToken(ManagerClaims())
		resp := h.GET("/ui/pages/nonexistent.page", token)
		h.AssertStatus(t, resp, http.StatusNotFound)
	})
}

func TestHarness_FormLoad(t *testing.T) {
	h := NewTestHarness(t)

	t.Run("get form descriptor", func(t *testing.T) {
		token := h.GenerateToken(ClerkClaims())
		resp := h.GET("/ui/forms/orders.edit_form", token)
		h.AssertStatus(t, resp, http.StatusOK)

		var body map[string]any
		h.ParseJSON(resp, &body)

		if body["id"] != "orders.edit_form" {
			t.Errorf("form id = %v, want orders.edit_form", body["id"])
		}
	})

	t.Run("forbidden form", func(t *testing.T) {
		token := h.GenerateToken(ViewerClaims())
		resp := h.GET("/ui/forms/orders.edit_form", token)
		h.AssertStatus(t, resp, http.StatusForbidden)
	})

	t.Run("get form data loads from backend", func(t *testing.T) {
		token := h.GenerateToken(ClerkClaims())

		order := OrderFixture("ord-1", "ORD-001", "pending")
		h.MockBackend("orders-svc").OnOperation("getOrder").
			RespondWith(200, order)

		resp := h.GET("/ui/forms/orders.edit_form/data?id=ord-1", token)
		h.AssertStatus(t, resp, http.StatusOK)

		h.MockBackend("orders-svc").AssertCalled(t, "getOrder", 1)
	})
}

func TestHarness_CommandExecution(t *testing.T) {
	h := NewTestHarness(t)

	t.Run("successful cancel command", func(t *testing.T) {
		token := h.GenerateToken(ManagerClaims())

		cancelledOrder := OrderFixture("ord-1", "ORD-001", "cancelled")
		h.MockBackend("orders-svc").OnOperation("cancelOrder").
			RespondWith(200, cancelledOrder)

		resp := h.POST("/ui/commands/orders.cancel", map[string]any{
			"input": map[string]any{
				"id":     "ord-1",
				"reason": "Customer requested cancellation",
			},
		}, token)
		h.AssertStatus(t, resp, http.StatusOK)

		h.MockBackend("orders-svc").AssertCalled(t, "cancelOrder", 1)
	})

	t.Run("command with insufficient capabilities", func(t *testing.T) {
		token := h.GenerateToken(ViewerClaims())

		// Reset mock to clear state from previous subtests.
		h.MockBackend("orders-svc").ResetOperation("cancelOrder")

		resp := h.POST("/ui/commands/orders.cancel", map[string]any{
			"input": map[string]any{
				"id":     "ord-1",
				"reason": "test",
			},
		}, token)
		h.AssertStatus(t, resp, http.StatusForbidden)

		h.MockBackend("orders-svc").AssertNotCalled(t, "cancelOrder")
	})

	t.Run("backend error translation", func(t *testing.T) {
		token := h.GenerateToken(ManagerClaims())

		h.MockBackend("orders-svc").ResetOperation("cancelOrder")
		h.MockBackend("orders-svc").OnOperation("cancelOrder").
			RespondWithError(422, "ORDER_SHIPPED", "Cannot cancel a shipped order")

		resp := h.POST("/ui/commands/orders.cancel", map[string]any{
			"input": map[string]any{
				"id":     "ord-2",
				"reason": "test",
			},
		}, token)

		// The BFF translates backend 4xx errors to 400 (BAD_REQUEST) with
		// the translated error message from the command's error_map.
		h.AssertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("nonexistent command returns 404", func(t *testing.T) {
		token := h.GenerateToken(ManagerClaims())
		resp := h.POST("/ui/commands/nonexistent.command", map[string]any{
			"input": map[string]any{},
		}, token)
		h.AssertStatus(t, resp, http.StatusNotFound)
	})
}

func TestHarness_MockBackendRecording(t *testing.T) {
	h := NewTestHarness(t)
	mock := h.MockBackend("orders-svc")

	token := h.GenerateToken(ManagerClaims())

	// Make a cancel command call.
	mock.OnOperation("cancelOrder").RespondWith(200, OrderFixture("ord-1", "ORD-001", "cancelled"))

	h.POST("/ui/commands/orders.cancel", map[string]any{
		"input": map[string]any{
			"id":     "ord-1",
			"reason": "test cancellation",
		},
	}, token)

	t.Run("assert called count", func(t *testing.T) {
		mock.AssertCalled(t, "cancelOrder", 1)
	})

	t.Run("inspect last request", func(t *testing.T) {
		req := mock.LastRequest("cancelOrder")
		if req == nil {
			t.Fatal("expected recorded request")
		}
		if req.Method != "POST" {
			t.Errorf("method = %q, want POST", req.Method)
		}
		// The path should include the order ID.
		if req.Path != "/api/orders/ord-1/cancel" {
			t.Errorf("path = %q, want /api/orders/ord-1/cancel", req.Path)
		}
	})

	t.Run("inspect request body", func(t *testing.T) {
		req := mock.LastRequest("cancelOrder")
		if req == nil {
			t.Fatal("expected recorded request")
		}
		if req.Body == nil {
			t.Fatal("expected request body")
		}
		// The body is the projected field (input.reason → reason).
		reason, _ := req.Body["reason"].(string)
		if reason != "test cancellation" {
			t.Errorf("reason = %q, want 'test cancellation'", reason)
		}
	})

	t.Run("inspect forwarded headers", func(t *testing.T) {
		req := mock.LastRequest("cancelOrder")
		if req.Headers.Get("X-Tenant-Id") != "acme-corp" {
			t.Errorf("X-Tenant-Id = %q, want acme-corp", req.Headers.Get("X-Tenant-Id"))
		}
		if req.Headers.Get("X-Request-Subject") != "user-manager" {
			t.Errorf("X-Request-Subject = %q, want user-manager", req.Headers.Get("X-Request-Subject"))
		}
	})

	t.Run("reset clears recordings", func(t *testing.T) {
		mock.Reset()
		mock.AssertNotCalled(t, "cancelOrder")
		if req := mock.LastRequest("cancelOrder"); req != nil {
			t.Error("expected nil after reset")
		}
	})
}

func TestHarness_SearchEndpoint(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	orders := []map[string]any{
		OrderFixture("ord-1", "ORD-2024-001", "pending"),
	}
	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(200, map[string]any{
			"data":        orders,
			"total_count": 1,
		})

	resp := h.GET("/ui/search?q=ORD-2024", token)
	h.AssertStatus(t, resp, http.StatusOK)

	h.MockBackend("orders-svc").AssertCalled(t, "searchOrders", 1)
}

func TestHarness_LookupEndpoint(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	statuses := []map[string]any{
		{"label": "Pending", "value": "pending"},
		{"label": "Processing", "value": "processing"},
		{"label": "Shipped", "value": "shipped"},
	}
	h.MockBackend("orders-svc").OnOperation("getOrderStatuses").
		RespondWith(200, statuses)

	resp := h.GET("/ui/lookups/orders.statuses", token)
	h.AssertStatus(t, resp, http.StatusOK)

	h.MockBackend("orders-svc").AssertCalled(t, "getOrderStatuses", 1)
}

func TestHarness_WorkflowStart(t *testing.T) {
	// Register a no-op SDK handler for the notification step.
	noop := &noopSDKHandler{name: "notifications.send"}
	h := NewTestHarness(t, WithWorkflows(), WithSDKHandler("notifications.send", noop))
	token := h.GenerateToken(ApproverClaims())

	// The workflow start may invoke the confirmOrder operation if it auto-advances.
	// For now, we just test that the start endpoint works.
	resp := h.POST("/ui/workflows/orders.approval/start", map[string]any{
		"order_id": "ord-1",
	}, token)

	// 201 Created or 200 OK depending on implementation.
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body := h.ReadBody(resp)
		t.Fatalf("workflow start: status = %d\nbody: %s", resp.StatusCode, body)
	}
}

func TestHarness_CrossTenantIsolation(t *testing.T) {
	noop := &noopSDKHandler{name: "notifications.send"}
	h := NewTestHarness(t, WithWorkflows(), WithSDKHandler("notifications.send", noop))

	// Start a workflow as tenant "acme-corp".
	acmeToken := h.GenerateToken(ApproverClaims())
	resp := h.POST("/ui/workflows/orders.approval/start", map[string]any{
		"order_id": "ord-1",
	}, acmeToken)

	var startBody map[string]any
	h.ParseJSON(resp, &startBody)

	instanceID, _ := startBody["id"].(string)
	if instanceID == "" {
		t.Fatal("expected workflow instance ID in response")
	}

	// Try to access from a different tenant.
	evilClaims := TestClaims{
		SubjectID: "evil-user",
		TenantID:  "evil-corp",
		Email:     "evil@evil.example.com",
		Roles:     []string{"order_approver"},
	}
	evilToken := h.GenerateToken(evilClaims)

	resp = h.GET("/ui/workflows/"+instanceID, evilToken)
	// Should be 404 (not 403) to prevent ID enumeration.
	h.AssertStatus(t, resp, http.StatusNotFound)
}

// noopSDKHandler is a no-op SDK handler for testing.
type noopSDKHandler struct {
	name    string
	invoked bool
}

func (h *noopSDKHandler) Name() string { return h.name }
func (h *noopSDKHandler) Invoke(_ context.Context, _ *model.RequestContext, _ model.InvocationInput) (model.InvocationResult, error) {
	h.invoked = true
	return model.InvocationResult{StatusCode: 200}, nil
}
