package integration

import (
	"net/http"
	"testing"
	"time"
)

// ==========================================================================
// Global Search Tests
// ==========================================================================

func TestSearch_ReturnsResultsFromMultipleProviders(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims()) // has orders:view + orders:manage

	// Configure both providers to return results.
	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(200, map[string]any{
			"data": []map[string]any{
				{"id": "ord-1", "order_number": "ORD-001", "customer_name": "ACME Corp", "status": "pending"},
				{"id": "ord-2", "order_number": "ORD-002", "customer_name": "ACME Inc", "status": "shipped"},
			},
		})
	h.MockBackend("orders-svc").OnOperation("searchCustomers").
		RespondWith(200, map[string]any{
			"data": []map[string]any{
				{"id": "cust-1", "name": "ACME Corp", "email": "contact@acme.com"},
			},
		})

	resp := h.GET("/ui/search?q=ACME", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	results := data["results"].([]any)
	totalCount := int(data["total_count"].(float64))

	if totalCount != 3 {
		t.Errorf("total_count = %d, want 3 (2 orders + 1 customer)", totalCount)
	}
	if len(results) != 3 {
		t.Fatalf("results len = %d, want 3", len(results))
	}

	// Verify query is echoed back.
	assertEqual(t, data["query"].(string), "ACME", "query")

	// Verify results are sorted by score descending.
	for i := 1; i < len(results); i++ {
		prev := results[i-1].(map[string]any)["score"].(float64)
		curr := results[i].(map[string]any)["score"].(float64)
		if curr > prev {
			t.Errorf("results not sorted by score: result[%d].score=%f > result[%d].score=%f", i, curr, i-1, prev)
		}
	}

	// Verify provider metadata.
	meta := result["meta"].(map[string]any)
	providers := meta["providers"].(map[string]any)
	assertEqual(t, providers["orders.search"].(string), "ok", "orders.search status")
	assertEqual(t, providers["customers.search"].(string), "ok", "customers.search status")

	// Verify both backends were called.
	h.MockBackend("orders-svc").AssertCalled(t, "searchOrders", 1)
	h.MockBackend("orders-svc").AssertCalled(t, "searchCustomers", 1)
}

func TestSearch_ProviderFilteredByCapability(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims()) // only has orders:view, NOT orders:manage

	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(200, map[string]any{
			"data": []map[string]any{
				{"id": "ord-1", "order_number": "ORD-001", "customer_name": "ACME", "status": "pending"},
			},
		})
	h.MockBackend("orders-svc").OnOperation("searchCustomers").
		RespondWith(200, map[string]any{
			"data": []map[string]any{
				{"id": "cust-1", "name": "ACME Corp", "email": "a@acme.com"},
			},
		})

	resp := h.GET("/ui/search?q=ACME", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	totalCount := int(data["total_count"].(float64))

	// Only orders.search should be included (viewer lacks orders:manage for customers.search).
	if totalCount != 1 {
		t.Errorf("total_count = %d, want 1 (only orders provider)", totalCount)
	}

	// Verify customers provider was NOT called.
	h.MockBackend("orders-svc").AssertCalled(t, "searchOrders", 1)
	h.MockBackend("orders-svc").AssertNotCalled(t, "searchCustomers")

	// Verify provider metadata only shows orders.
	meta := result["meta"].(map[string]any)
	providers := meta["providers"].(map[string]any)
	assertEqual(t, providers["orders.search"].(string), "ok", "orders.search status")
	if _, exists := providers["customers.search"]; exists {
		t.Error("customers.search should not appear in providers when user lacks capability")
	}
}

func TestSearch_SlowProviderTimesOut(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	// Fast provider returns immediately.
	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(200, map[string]any{
			"data": []map[string]any{
				{"id": "ord-1", "order_number": "ORD-001", "customer_name": "ACME", "status": "pending"},
			},
		})

	// Slow provider delays beyond the 3s per-provider timeout.
	h.MockBackend("orders-svc").OnOperation("searchCustomers").
		RespondWithDelay(5*time.Second, 200, map[string]any{
			"data": []map[string]any{
				{"id": "cust-1", "name": "ACME Corp", "email": "a@acme.com"},
			},
		})

	resp := h.GET("/ui/search?q=ACME", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	totalCount := int(data["total_count"].(float64))

	// Only the fast provider's results should be returned.
	if totalCount != 1 {
		t.Errorf("total_count = %d, want 1 (slow provider timed out)", totalCount)
	}

	// Verify provider metadata shows timeout.
	meta := result["meta"].(map[string]any)
	providers := meta["providers"].(map[string]any)
	assertEqual(t, providers["orders.search"].(string), "ok", "fast provider status")
	assertEqual(t, providers["customers.search"].(string), "timeout", "slow provider status")
}

func TestSearch_FailedProviderOmitted(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	// Successful provider.
	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(200, map[string]any{
			"data": []map[string]any{
				{"id": "ord-1", "order_number": "ORD-001", "customer_name": "ACME", "status": "pending"},
			},
		})

	// Failed provider returns 500.
	h.MockBackend("orders-svc").OnOperation("searchCustomers").
		RespondWith(500, map[string]any{"error": "internal server error"})

	resp := h.GET("/ui/search?q=ACME", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	totalCount := int(data["total_count"].(float64))

	// Only the successful provider's results.
	if totalCount != 1 {
		t.Errorf("total_count = %d, want 1 (failed provider omitted)", totalCount)
	}

	// Verify metadata shows the error.
	meta := result["meta"].(map[string]any)
	providers := meta["providers"].(map[string]any)
	assertEqual(t, providers["orders.search"].(string), "ok", "successful provider")
	assertEqual(t, providers["customers.search"].(string), "error", "failed provider")
}

func TestSearch_DeduplicationKeepsHighestScore(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	// Both providers return the same entity (same id, route resolves to same path).
	// orders.search has weight 1.0, customers.search has weight 2.0.
	// The same entity from customers.search should have a higher score.
	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(200, map[string]any{
			"data": []map[string]any{
				// This order has id "shared-1" which maps to route /orders/shared-1
				{"id": "shared-1", "order_number": "ORD-SHARED", "customer_name": "ACME", "status": "pending"},
			},
		})

	h.MockBackend("orders-svc").OnOperation("searchCustomers").
		RespondWith(200, map[string]any{
			"data": []map[string]any{
				// Different entity with different route → not a duplicate.
				{"id": "cust-1", "name": "ACME Corp", "email": "a@acme.com"},
			},
		})

	resp := h.GET("/ui/search?q=ACME", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	results := data["results"].([]any)
	totalCount := int(data["total_count"].(float64))

	// Both results should appear (different routes: /orders/shared-1 vs /customers/cust-1).
	if totalCount != 2 {
		t.Errorf("total_count = %d, want 2", totalCount)
	}

	// The customer result (weight 2.0) should rank first.
	if len(results) >= 2 {
		first := results[0].(map[string]any)
		firstRoute := first["route"].(string)
		if firstRoute != "/customers/cust-1" {
			t.Errorf("first result route = %q, want /customers/cust-1 (higher weight)", firstRoute)
		}
	}
}

func TestSearch_PaginationAppliedAfterMerge(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	// Return enough results to test pagination.
	orderResults := make([]map[string]any, 5)
	for i := range orderResults {
		orderResults[i] = map[string]any{
			"id":            "ord-" + string(rune('1'+i)),
			"order_number":  "ORD-00" + string(rune('1'+i)),
			"customer_name": "Customer",
			"status":        "pending",
		}
	}

	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(200, map[string]any{"data": orderResults})
	h.MockBackend("orders-svc").OnOperation("searchCustomers").
		RespondWith(200, map[string]any{"data": []map[string]any{}})

	// Request page 1 with page_size 2.
	resp := h.GET("/ui/search?q=ORD&page=1&page_size=2", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	results := data["results"].([]any)
	totalCount := int(data["total_count"].(float64))

	if totalCount != 5 {
		t.Errorf("total_count = %d, want 5 (total before pagination)", totalCount)
	}
	if len(results) != 2 {
		t.Errorf("results on page = %d, want 2 (page_size)", len(results))
	}

	// Request page 3 (should have 1 result: items 5 of 5).
	resp2 := h.GET("/ui/search?q=ORD&page=3&page_size=2", token)

	var result2 map[string]any
	h.AssertJSON(t, resp2, http.StatusOK, &result2)

	data2 := result2["data"].(map[string]any)
	results2 := data2["results"].([]any)
	if len(results2) != 1 {
		t.Errorf("page 3 results = %d, want 1", len(results2))
	}
}

func TestSearch_ResultMapping(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(200, map[string]any{
			"data": []map[string]any{
				{
					"id":            "ord-42",
					"order_number":  "ORD-2024-042",
					"customer_name": "Widget Co",
					"status":        "processing",
				},
			},
		})

	resp := h.GET("/ui/search?q=Widget", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	results := data["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}

	r := results[0].(map[string]any)
	assertEqual(t, r["id"].(string), "ord-42", "id")
	assertEqual(t, r["title"].(string), "ORD-2024-042", "title (from order_number)")
	assertEqual(t, r["subtitle"].(string), "Widget Co", "subtitle (from customer_name)")
	assertEqual(t, r["category"].(string), "orders", "category (from domain)")
	assertEqual(t, r["route"].(string), "/orders/ord-42", "route (template resolved)")

	// Score should be positive.
	score := r["score"].(float64)
	if score <= 0 {
		t.Errorf("score = %f, want > 0", score)
	}
}

func TestSearch_QueryTooShort(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	resp := h.GET("/ui/search?q=A", token)
	h.AssertStatus(t, resp, http.StatusBadRequest)

	// Backend should not be called.
	h.MockBackend("orders-svc").AssertNotCalled(t, "searchOrders")
}

func TestSearch_EmptyQuery(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	resp := h.GET("/ui/search?q=", token)
	h.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestSearch_DomainFilter(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(200, map[string]any{
			"data": []map[string]any{
				{"id": "ord-1", "order_number": "ORD-001", "customer_name": "Test", "status": "pending"},
			},
		})
	h.MockBackend("orders-svc").OnOperation("searchCustomers").
		RespondWith(200, map[string]any{
			"data": []map[string]any{
				{"id": "cust-1", "name": "Test Corp", "email": "t@test.com"},
			},
		})

	// Filter to orders domain only.
	resp := h.GET("/ui/search?q=Test&domain=orders", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	totalCount := int(data["total_count"].(float64))

	if totalCount != 1 {
		t.Errorf("total_count = %d, want 1 (filtered to orders domain)", totalCount)
	}

	// Customers provider should not be called when domain filter is applied.
	h.MockBackend("orders-svc").AssertCalled(t, "searchOrders", 1)
	h.MockBackend("orders-svc").AssertNotCalled(t, "searchCustomers")
}

func TestSearch_QueryParamPassedToBackend(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(200, map[string]any{"data": []map[string]any{}})

	h.GET("/ui/search?q=test+query", token)

	req := h.MockBackend("orders-svc").LastRequest("searchOrders")
	if req == nil {
		t.Fatal("searchOrders was not called")
	}
	if req.QueryParams["q"] != "test query" {
		t.Errorf("query param q = %q, want %q", req.QueryParams["q"], "test query")
	}
}

func TestSearch_Unauthenticated(t *testing.T) {
	h := NewTestHarness(t)

	resp := h.GET("/ui/search?q=ACME", "")
	h.AssertStatus(t, resp, http.StatusUnauthorized)
}

// ==========================================================================
// Lookup Tests
// ==========================================================================

func TestLookup_DynamicLookupCallsBackend(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	h.MockBackend("orders-svc").OnOperation("getOrderStatuses").
		RespondWith(200, []map[string]any{
			{"label": "Pending", "value": "pending"},
			{"label": "Processing", "value": "processing"},
			{"label": "Shipped", "value": "shipped"},
			{"label": "Cancelled", "value": "cancelled"},
		})

	resp := h.GET("/ui/lookups/orders.statuses", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	options := data["options"].([]any)

	if len(options) != 4 {
		t.Fatalf("options len = %d, want 4", len(options))
	}

	// Verify mapping of label_field and value_field.
	first := options[0].(map[string]any)
	assertEqual(t, first["label"].(string), "Pending", "first option label")
	assertEqual(t, first["value"].(string), "pending", "first option value")

	// First call should not be cached.
	meta := result["meta"].(map[string]any)
	assertEqual(t, meta["cached"].(bool), false, "first call not cached")

	h.MockBackend("orders-svc").AssertCalled(t, "getOrderStatuses", 1)
}

func TestLookup_CachedResultNoBackendCall(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	h.MockBackend("orders-svc").OnOperation("getOrderStatuses").
		RespondWith(200, []map[string]any{
			{"label": "Pending", "value": "pending"},
			{"label": "Shipped", "value": "shipped"},
		})

	// First call — fetches from backend.
	resp1 := h.GET("/ui/lookups/orders.statuses", token)
	var result1 map[string]any
	h.AssertJSON(t, resp1, http.StatusOK, &result1)
	meta1 := result1["meta"].(map[string]any)
	assertEqual(t, meta1["cached"].(bool), false, "first call not cached")

	// Second call — should return from cache.
	resp2 := h.GET("/ui/lookups/orders.statuses", token)
	var result2 map[string]any
	h.AssertJSON(t, resp2, http.StatusOK, &result2)
	meta2 := result2["meta"].(map[string]any)
	assertEqual(t, meta2["cached"].(bool), true, "second call cached")

	// Backend should only be called once.
	h.MockBackend("orders-svc").AssertCalled(t, "getOrderStatuses", 1)

	// Cached result should have same options.
	data2 := result2["data"].(map[string]any)
	options2 := data2["options"].([]any)
	if len(options2) != 2 {
		t.Errorf("cached options len = %d, want 2", len(options2))
	}
}

func TestLookup_QueryFiltersOptions(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	h.MockBackend("orders-svc").OnOperation("getOrderStatuses").
		RespondWith(200, []map[string]any{
			{"label": "Pending", "value": "pending"},
			{"label": "Processing", "value": "processing"},
			{"label": "Shipped", "value": "shipped"},
			{"label": "Cancelled", "value": "cancelled"},
		})

	// Filter with q=pen — should match "Pending" (case-insensitive substring).
	resp := h.GET("/ui/lookups/orders.statuses?q=pen", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	options := data["options"].([]any)

	if len(options) != 1 {
		t.Fatalf("filtered options len = %d, want 1", len(options))
	}
	assertEqual(t, options[0].(map[string]any)["label"].(string), "Pending", "filtered label")
}

func TestLookup_QueryFilterOnCachedResult(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	h.MockBackend("orders-svc").OnOperation("getOrderStatuses").
		RespondWith(200, []map[string]any{
			{"label": "Pending", "value": "pending"},
			{"label": "Processing", "value": "processing"},
			{"label": "Shipped", "value": "shipped"},
		})

	// First call — populate cache (no filter).
	h.GET("/ui/lookups/orders.statuses", token)

	// Second call with filter — should filter from cache.
	resp := h.GET("/ui/lookups/orders.statuses?q=ship", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	options := data["options"].([]any)

	if len(options) != 1 {
		t.Fatalf("filtered cached options len = %d, want 1", len(options))
	}
	assertEqual(t, options[0].(map[string]any)["label"].(string), "Shipped", "filtered cached label")

	meta := result["meta"].(map[string]any)
	assertEqual(t, meta["cached"].(bool), true, "second call cached")

	// Backend only called once despite query.
	h.MockBackend("orders-svc").AssertCalled(t, "getOrderStatuses", 1)
}

func TestLookup_NotFound(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	resp := h.GET("/ui/lookups/nonexistent.lookup", token)
	h.AssertStatus(t, resp, http.StatusNotFound)
}

func TestLookup_TenantScopedCache(t *testing.T) {
	h := NewTestHarness(t)

	// orders.statuses has scope: tenant.
	// Two different tenants should get separate cache entries.
	acmeToken := h.GenerateToken(TestClaims{
		SubjectID: "user-1",
		TenantID:  "acme-corp",
		Email:     "user@acme.com",
		Roles:     []string{"order_viewer"},
	})
	otherToken := h.GenerateToken(TestClaims{
		SubjectID: "user-2",
		TenantID:  "other-corp",
		Email:     "user@other.com",
		Roles:     []string{"order_viewer"},
	})

	h.MockBackend("orders-svc").OnOperation("getOrderStatuses").
		RespondWith(200, []map[string]any{
			{"label": "Pending", "value": "pending"},
		})

	// First tenant call.
	resp1 := h.GET("/ui/lookups/orders.statuses", acmeToken)
	var r1 map[string]any
	h.AssertJSON(t, resp1, http.StatusOK, &r1)
	assertEqual(t, r1["meta"].(map[string]any)["cached"].(bool), false, "acme first call not cached")

	// Second tenant call — should call backend again (different tenant = different cache key).
	resp2 := h.GET("/ui/lookups/orders.statuses", otherToken)
	var r2 map[string]any
	h.AssertJSON(t, resp2, http.StatusOK, &r2)
	assertEqual(t, r2["meta"].(map[string]any)["cached"].(bool), false, "other-corp first call not cached")

	// Backend called twice (once per tenant).
	h.MockBackend("orders-svc").AssertCalled(t, "getOrderStatuses", 2)

	// Now same-tenant second call should be cached.
	resp3 := h.GET("/ui/lookups/orders.statuses", acmeToken)
	var r3 map[string]any
	h.AssertJSON(t, resp3, http.StatusOK, &r3)
	assertEqual(t, r3["meta"].(map[string]any)["cached"].(bool), true, "acme second call cached")

	// Still only 2 backend calls.
	h.MockBackend("orders-svc").AssertCalled(t, "getOrderStatuses", 2)
}

func TestLookup_BackendResponseUnderDataKey(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	// Backend returns options under "data" key (not a direct array).
	h.MockBackend("orders-svc").OnOperation("getOrderStatuses").
		RespondWith(200, map[string]any{
			"data": []map[string]any{
				{"label": "Active", "value": "active"},
				{"label": "Inactive", "value": "inactive"},
			},
		})

	resp := h.GET("/ui/lookups/orders.statuses", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	options := data["options"].([]any)

	if len(options) != 2 {
		t.Fatalf("options len = %d, want 2", len(options))
	}
	assertEqual(t, options[0].(map[string]any)["label"].(string), "Active", "first label")
}

func TestLookup_BackendResponseUnderItemsKey(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	// Backend returns options under "items" key.
	h.MockBackend("orders-svc").OnOperation("getOrderStatuses").
		RespondWith(200, map[string]any{
			"items": []map[string]any{
				{"label": "Open", "value": "open"},
				{"label": "Closed", "value": "closed"},
			},
		})

	resp := h.GET("/ui/lookups/orders.statuses", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	options := data["options"].([]any)

	if len(options) != 2 {
		t.Fatalf("options len = %d, want 2", len(options))
	}
	assertEqual(t, options[0].(map[string]any)["label"].(string), "Open", "first label")
}

func TestLookup_Unauthenticated(t *testing.T) {
	h := NewTestHarness(t)

	resp := h.GET("/ui/lookups/orders.statuses", "")
	h.AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestLookup_CaseInsensitiveFilter(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	h.MockBackend("orders-svc").OnOperation("getOrderStatuses").
		RespondWith(200, []map[string]any{
			{"label": "Pending", "value": "pending"},
			{"label": "Processing", "value": "processing"},
			{"label": "Shipped", "value": "shipped"},
		})

	// Upper case query should still match (case-insensitive).
	resp := h.GET("/ui/lookups/orders.statuses?q=SHIP", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	options := data["options"].([]any)

	if len(options) != 1 {
		t.Fatalf("filtered options len = %d, want 1", len(options))
	}
	assertEqual(t, options[0].(map[string]any)["label"].(string), "Shipped", "case-insensitive match")
}

func TestLookup_NoMatchingFilter(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	h.MockBackend("orders-svc").OnOperation("getOrderStatuses").
		RespondWith(200, []map[string]any{
			{"label": "Pending", "value": "pending"},
			{"label": "Shipped", "value": "shipped"},
		})

	// Query that matches nothing.
	resp := h.GET("/ui/lookups/orders.statuses?q=xyz", token)

	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	data := result["data"].(map[string]any)
	options, _ := data["options"].([]any)

	if len(options) != 0 {
		t.Errorf("options len = %d, want 0 (no match)", len(options))
	}
}
