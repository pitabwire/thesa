package integration

import (
	"net/http"
	"testing"
)

// ==========================================================================
// Successful Command Execution
// ==========================================================================

func TestCommand_SuccessfulCancel(t *testing.T) {
	h := NewTestHarness(t)
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

	// Verify success response structure.
	var body map[string]any
	h.ParseJSON(resp, &body)

	if body["success"] != true {
		t.Errorf("success = %v, want true", body["success"])
	}
	// Result should contain the response from the backend.
	result, _ := body["result"].(map[string]any)
	if result == nil {
		t.Fatal("expected result in command response")
	}
	assertEqual(t, result["status"], "cancelled", "result.status")

	h.MockBackend("orders-svc").AssertCalled(t, "cancelOrder", 1)
}

func TestCommand_PathParamsResolvedFromInput(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	h.MockBackend("orders-svc").OnOperation("cancelOrder").
		RespondWith(200, OrderFixture("ord-42", "ORD-042", "cancelled"))

	h.POST("/ui/commands/orders.cancel", map[string]any{
		"input": map[string]any{
			"id":     "ord-42",
			"reason": "test",
		},
	}, token)

	req := h.MockBackend("orders-svc").LastRequest("cancelOrder")
	if req == nil {
		t.Fatal("expected recorded request")
	}
	// The path should include the order ID from input.id.
	assertEqual(t, req.Path, "/api/orders/ord-42/cancel", "request path")
}

func TestCommand_BodyProjectionMapping(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	h.MockBackend("orders-svc").OnOperation("updateOrder").
		RespondWith(200, OrderFixture("ord-1", "ORD-001", "pending"))

	h.POST("/ui/commands/orders.update", map[string]any{
		"input": map[string]any{
			"id":               "ord-1",
			"shipping_address": "456 New St",
			"priority":         "high",
			"notes":            "Urgent delivery",
			"internal_code":    "IC-789",
		},
	}, token)

	h.MockBackend("orders-svc").AssertCalled(t, "updateOrder", 1)

	req := h.MockBackend("orders-svc").LastRequest("updateOrder")
	if req == nil {
		t.Fatal("expected recorded request")
	}

	// Verify path param resolved from input.id.
	assertEqual(t, req.Path, "/api/orders/ord-1", "request path")
	// Verify method is PATCH.
	assertEqual(t, req.Method, "PATCH", "request method")

	// Verify projected body fields.
	assertEqual(t, req.Body["shipping_address"], "456 New St", "body.shipping_address")
	assertEqual(t, req.Body["priority"], "high", "body.priority")
	assertEqual(t, req.Body["notes"], "Urgent delivery", "body.notes")
	assertEqual(t, req.Body["internal_code"], "IC-789", "body.internal_code")

	// Verify input.id is NOT in the body (it's a path param, not a body field).
	if _, exists := req.Body["id"]; exists {
		t.Error("body should not contain 'id' (it's a path param)")
	}
}

func TestCommand_CancelProjectionBody(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	h.MockBackend("orders-svc").OnOperation("cancelOrder").
		RespondWith(200, OrderFixture("ord-1", "ORD-001", "cancelled"))

	h.POST("/ui/commands/orders.cancel", map[string]any{
		"input": map[string]any{
			"id":     "ord-1",
			"reason": "Customer changed mind",
		},
	}, token)

	req := h.MockBackend("orders-svc").LastRequest("cancelOrder")
	if req == nil {
		t.Fatal("expected recorded request")
	}

	// Only the projected field (reason) should be in the body.
	assertEqual(t, req.Body["reason"], "Customer changed mind", "body.reason")

	// id should NOT be in the body.
	if _, exists := req.Body["id"]; exists {
		t.Error("body should not contain 'id'")
	}
}

func TestCommand_HeadersForwardedToBackend(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	h.MockBackend("orders-svc").OnOperation("cancelOrder").
		RespondWith(200, OrderFixture("ord-1", "ORD-001", "cancelled"))

	h.POST("/ui/commands/orders.cancel", map[string]any{
		"input": map[string]any{
			"id":     "ord-1",
			"reason": "test",
		},
	}, token)

	req := h.MockBackend("orders-svc").LastRequest("cancelOrder")
	if req == nil {
		t.Fatal("expected recorded request")
	}

	if req.Headers.Get("Authorization") == "" {
		t.Error("backend request missing Authorization header")
	}
	assertEqual(t, req.Headers.Get("X-Tenant-Id"), "acme-corp", "X-Tenant-Id")
	assertEqual(t, req.Headers.Get("X-Request-Subject"), "user-manager", "X-Request-Subject")
	if req.Headers.Get("X-Correlation-Id") == "" {
		t.Error("backend request missing X-Correlation-Id header")
	}
}

// ==========================================================================
// Backend Error Translation
// ==========================================================================

func TestCommand_BackendKnownErrorCode(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	// Backend returns 422 with ORDER_SHIPPED error code.
	h.MockBackend("orders-svc").OnOperation("cancelOrder").
		RespondWithError(422, "ORDER_SHIPPED", "Cannot cancel a shipped order")

	resp := h.POST("/ui/commands/orders.cancel", map[string]any{
		"input": map[string]any{
			"id":     "ord-1",
			"reason": "test",
		},
	}, token)

	// The BFF translates backend 4xx errors to BAD_REQUEST (400).
	h.AssertStatus(t, resp, http.StatusBadRequest)

	var body map[string]any
	h.ParseJSON(resp, &body)

	// The error response wraps the translated error.
	errObj, _ := body["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected error object")
	}
	// The error message should be the translated message from the error_map.
	// error_map: ORDER_SHIPPED → "Cannot cancel a shipped order."
	assertEqual(t, errObj["code"], "BAD_REQUEST", "error.code")
}

func TestCommand_BackendKnownErrorCode_Update(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	// Backend returns 422 with ORDER_LOCKED error code.
	h.MockBackend("orders-svc").OnOperation("updateOrder").
		RespondWithError(422, "ORDER_LOCKED", "Order is locked")

	resp := h.POST("/ui/commands/orders.update", map[string]any{
		"input": map[string]any{
			"id":               "ord-1",
			"shipping_address": "123 St",
			"priority":         "normal",
			"notes":            "",
			"internal_code":    "",
		},
	}, token)

	h.AssertStatus(t, resp, http.StatusBadRequest)

	var body map[string]any
	h.ParseJSON(resp, &body)

	errObj := body["error"].(map[string]any)
	// The message should be from the error_map: "This order is locked and cannot be edited."
	assertEqual(t, errObj["message"], "This order is locked and cannot be edited.", "error.message")
}

func TestCommand_BackendUnknownErrorCode(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	// Backend returns 422 with an unknown error code.
	h.MockBackend("orders-svc").OnOperation("cancelOrder").
		RespondWithError(422, "UNKNOWN_ERROR", "Some backend detail")

	resp := h.POST("/ui/commands/orders.cancel", map[string]any{
		"input": map[string]any{
			"id":     "ord-1",
			"reason": "test",
		},
	}, token)

	h.AssertStatus(t, resp, http.StatusBadRequest)

	var body map[string]any
	h.ParseJSON(resp, &body)

	// Unknown error codes are NOT translated but the backend message passes through.
	errObj := body["error"].(map[string]any)
	assertEqual(t, errObj["code"], "BAD_REQUEST", "error.code")
}

func TestCommand_BackendServerError(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	// Backend returns 500.
	h.MockBackend("orders-svc").OnOperation("cancelOrder").
		RespondWith(500, map[string]any{
			"error": "internal server error",
		})

	resp := h.POST("/ui/commands/orders.cancel", map[string]any{
		"input": map[string]any{
			"id":     "ord-1",
			"reason": "test",
		},
	}, token)

	// The executor maps backend 5xx to BAD_REQUEST with a generic message.
	h.AssertStatus(t, resp, http.StatusBadRequest)

	var body map[string]any
	h.ParseJSON(resp, &body)

	errObj := body["error"].(map[string]any)
	assertEqual(t, errObj["code"], "BAD_REQUEST", "error.code")
	// Generic message, no backend details leaked.
	assertEqual(t, errObj["message"], "An internal error occurred. Please try again later.", "error.message")
}

// ==========================================================================
// Idempotency
// ==========================================================================

func TestCommand_IdempotencyCachedResult(t *testing.T) {
	h := NewTestHarness(t, WithIdempotency())
	token := h.GenerateToken(ManagerClaims())

	updatedOrder := OrderFixture("ord-1", "ORD-001", "pending")
	updatedOrder["shipping_address"] = "456 New St"
	h.MockBackend("orders-svc").OnOperation("updateOrder").
		RespondWith(200, updatedOrder)

	input := map[string]any{
		"input": map[string]any{
			"id":               "ord-1",
			"shipping_address": "456 New St",
			"priority":         "normal",
			"notes":            "Rush order",
			"internal_code":    "IC-001",
		},
	}

	// First request with idempotency key.
	resp := h.POSTWithHeaders("/ui/commands/orders.update", input, token, map[string]string{
		"X-Idempotency-Key": "idem-key-001",
	})
	h.AssertStatus(t, resp, http.StatusOK)

	var firstBody map[string]any
	h.ParseJSON(resp, &firstBody)
	if firstBody["success"] != true {
		t.Fatal("first request should succeed")
	}

	h.MockBackend("orders-svc").AssertCalled(t, "updateOrder", 1)

	// Second request with same idempotency key and same input.
	resp2 := h.POSTWithHeaders("/ui/commands/orders.update", input, token, map[string]string{
		"X-Idempotency-Key": "idem-key-001",
	})
	h.AssertStatus(t, resp2, http.StatusOK)

	var secondBody map[string]any
	h.ParseJSON(resp2, &secondBody)
	if secondBody["success"] != true {
		t.Fatal("cached request should succeed")
	}

	// Backend should NOT have been called a second time.
	h.MockBackend("orders-svc").AssertCalled(t, "updateOrder", 1)
}

func TestCommand_IdempotencyConflict(t *testing.T) {
	h := NewTestHarness(t, WithIdempotency())
	token := h.GenerateToken(ManagerClaims())

	h.MockBackend("orders-svc").OnOperation("updateOrder").
		RespondWith(200, OrderFixture("ord-1", "ORD-001", "pending"))

	// First request.
	resp := h.POSTWithHeaders("/ui/commands/orders.update", map[string]any{
		"input": map[string]any{
			"id":               "ord-1",
			"shipping_address": "456 New St",
			"priority":         "normal",
			"notes":            "First request",
			"internal_code":    "IC-001",
		},
	}, token, map[string]string{
		"X-Idempotency-Key": "idem-key-conflict",
	})
	h.AssertStatus(t, resp, http.StatusOK)
	h.ReadBody(resp)

	// Second request with SAME key but DIFFERENT input → 409 CONFLICT.
	resp2 := h.POSTWithHeaders("/ui/commands/orders.update", map[string]any{
		"input": map[string]any{
			"id":               "ord-1",
			"shipping_address": "789 Different St",
			"priority":         "urgent",
			"notes":            "Different request",
			"internal_code":    "IC-002",
		},
	}, token, map[string]string{
		"X-Idempotency-Key": "idem-key-conflict",
	})
	h.AssertStatus(t, resp2, http.StatusConflict)
}

func TestCommand_IdempotencyWithoutKey(t *testing.T) {
	h := NewTestHarness(t, WithIdempotency())
	token := h.GenerateToken(ManagerClaims())

	h.MockBackend("orders-svc").OnOperation("updateOrder").
		RespondWith(200, OrderFixture("ord-1", "ORD-001", "pending"))

	input := map[string]any{
		"input": map[string]any{
			"id":               "ord-1",
			"shipping_address": "456 New St",
			"priority":         "normal",
			"notes":            "Rush order",
			"internal_code":    "IC-001",
		},
	}

	// Without idempotency key, each request should invoke the backend.
	h.POST("/ui/commands/orders.update", input, token)
	h.POST("/ui/commands/orders.update", input, token)

	h.MockBackend("orders-svc").AssertCalled(t, "updateOrder", 2)
}

// ==========================================================================
// Authorization
// ==========================================================================

func TestCommand_InsufficientCapability(t *testing.T) {
	h := NewTestHarness(t)

	t.Run("viewer cannot cancel orders", func(t *testing.T) {
		token := h.GenerateToken(ViewerClaims())

		h.MockBackend("orders-svc").ResetOperation("cancelOrder")

		resp := h.POST("/ui/commands/orders.cancel", map[string]any{
			"input": map[string]any{
				"id":     "ord-1",
				"reason": "test",
			},
		}, token)
		h.AssertStatus(t, resp, http.StatusForbidden)

		// Backend should NOT be called.
		h.MockBackend("orders-svc").AssertNotCalled(t, "cancelOrder")
	})

	t.Run("viewer cannot update orders", func(t *testing.T) {
		token := h.GenerateToken(ViewerClaims())

		resp := h.POST("/ui/commands/orders.update", map[string]any{
			"input": map[string]any{
				"id":               "ord-1",
				"shipping_address": "123 St",
				"priority":         "high",
				"notes":            "",
				"internal_code":    "",
			},
		}, token)
		h.AssertStatus(t, resp, http.StatusForbidden)
	})

	t.Run("clerk can update but not cancel", func(t *testing.T) {
		token := h.GenerateToken(ClerkClaims())

		h.MockBackend("orders-svc").ResetOperation("updateOrder")
		h.MockBackend("orders-svc").OnOperation("updateOrder").
			RespondWith(200, OrderFixture("ord-1", "ORD-001", "pending"))

		// Clerk has orders:edit → can update.
		resp := h.POST("/ui/commands/orders.update", map[string]any{
			"input": map[string]any{
				"id":               "ord-1",
				"shipping_address": "123 St",
				"priority":         "normal",
				"notes":            "",
				"internal_code":    "",
			},
		}, token)
		h.AssertStatus(t, resp, http.StatusOK)

		// Clerk does NOT have orders:cancel → cannot cancel.
		h.MockBackend("orders-svc").ResetOperation("cancelOrder")
		resp = h.POST("/ui/commands/orders.cancel", map[string]any{
			"input": map[string]any{
				"id":     "ord-1",
				"reason": "test",
			},
		}, token)
		h.AssertStatus(t, resp, http.StatusForbidden)
	})
}

func TestCommand_NotFound(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	resp := h.POST("/ui/commands/nonexistent.command", map[string]any{
		"input": map[string]any{},
	}, token)
	h.AssertStatus(t, resp, http.StatusNotFound)
}

// ==========================================================================
// Output Mapping
// ==========================================================================

func TestCommand_OutputFullResponse(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	// The orders.cancel command has output type "full" → entire body returned.
	cancelledOrder := OrderFixture("ord-1", "ORD-001", "cancelled")
	cancelledOrder["notes"] = "Cancelled by manager"
	h.MockBackend("orders-svc").OnOperation("cancelOrder").
		RespondWith(200, cancelledOrder)

	resp := h.POST("/ui/commands/orders.cancel", map[string]any{
		"input": map[string]any{
			"id":     "ord-1",
			"reason": "Customer request",
		},
	}, token)
	h.AssertStatus(t, resp, http.StatusOK)

	var body map[string]any
	h.ParseJSON(resp, &body)

	if body["success"] != true {
		t.Error("expected success = true")
	}

	result := body["result"].(map[string]any)
	assertEqual(t, result["id"], "ord-1", "result.id")
	assertEqual(t, result["order_number"], "ORD-001", "result.order_number")
	assertEqual(t, result["status"], "cancelled", "result.status")
	assertEqual(t, result["notes"], "Cancelled by manager", "result.notes")
}

// ==========================================================================
// Confirm Command (different role)
// ==========================================================================

func TestCommand_ConfirmRequiresApproverRole(t *testing.T) {
	h := NewTestHarness(t)

	t.Run("approver can confirm", func(t *testing.T) {
		token := h.GenerateToken(ApproverClaims())

		h.MockBackend("orders-svc").OnOperation("confirmOrder").
			RespondWith(200, OrderFixture("ord-1", "ORD-001", "approved"))

		resp := h.POST("/ui/commands/orders.confirm", map[string]any{
			"input": map[string]any{
				"id":             "ord-1",
				"approval_notes": "Looks good, approved.",
			},
		}, token)
		h.AssertStatus(t, resp, http.StatusOK)

		h.MockBackend("orders-svc").AssertCalled(t, "confirmOrder", 1)

		// Verify the body was projected correctly.
		req := h.MockBackend("orders-svc").LastRequest("confirmOrder")
		assertEqual(t, req.Body["approval_notes"], "Looks good, approved.", "body.approval_notes")
		assertEqual(t, req.Path, "/api/orders/ord-1/confirm", "request path")
	})

	t.Run("manager cannot confirm", func(t *testing.T) {
		token := h.GenerateToken(ManagerClaims())

		h.MockBackend("orders-svc").ResetOperation("confirmOrder")

		resp := h.POST("/ui/commands/orders.confirm", map[string]any{
			"input": map[string]any{
				"id":             "ord-1",
				"approval_notes": "test",
			},
		}, token)
		h.AssertStatus(t, resp, http.StatusForbidden)

		h.MockBackend("orders-svc").AssertNotCalled(t, "confirmOrder")
	})
}

// ==========================================================================
// Error Response Structure
// ==========================================================================

func TestCommand_ErrorResponseStructure(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	h.MockBackend("orders-svc").OnOperation("cancelOrder").
		RespondWithError(422, "ORDER_ALREADY_CANCELLED", "This order has already been cancelled")

	resp := h.POST("/ui/commands/orders.cancel", map[string]any{
		"input": map[string]any{
			"id":     "ord-1",
			"reason": "test",
		},
	}, token)
	h.AssertStatus(t, resp, http.StatusBadRequest)

	var body map[string]any
	h.ParseJSON(resp, &body)

	errObj := body["error"].(map[string]any)
	// Error code should be BAD_REQUEST (not the backend error code).
	assertEqual(t, errObj["code"], "BAD_REQUEST", "error.code")
	// The translated message from the error_map.
	assertEqual(t, errObj["message"], "This order has already been cancelled.", "error.message")
}

func TestCommand_InvalidJSONBody(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	// Send raw invalid JSON.
	resp := h.POSTWithHeaders("/ui/commands/orders.cancel", nil, token, map[string]string{
		"Content-Type": "application/json",
	})

	// Should get 400 BAD_REQUEST for invalid JSON body.
	// nil body causes Decode to fail with EOF.
	h.AssertStatus(t, resp, http.StatusBadRequest)
}
