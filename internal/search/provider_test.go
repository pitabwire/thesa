package search

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/model"
)

// --- Test helpers ---

type mockSearchInvoker struct {
	handler func(binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error)
}

func (m *mockSearchInvoker) Invoke(_ context.Context, _ *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
	if m.handler != nil {
		return m.handler(binding, input)
	}
	return model.InvocationResult{StatusCode: 200}, nil
}

func (m *mockSearchInvoker) Supports(_ model.OperationBinding) bool { return true }

func testRctx() *model.RequestContext {
	return &model.RequestContext{
		SubjectID: "user-alice",
		TenantID:  "tenant-1",
	}
}

func testSearchDefinitions() []model.DomainDefinition {
	return []model.DomainDefinition{
		{
			Domain: "orders",
			Searches: []model.SearchDefinition{
				{
					ID:           "orders.search",
					Domain:       "orders",
					Capabilities: []string{"orders:search:execute"},
					Operation:    model.OperationBinding{Type: "openapi", OperationID: "searchOrders"},
					ResultMapping: model.SearchResultMapping{
						ItemsPath:     "data.results",
						TitleField:    "orderNumber",
						SubtitleField: "customerName",
						Route:         "/orders/{id}",
						IDField:       "id",
					},
					Weight:     10,
					MaxResults: 5,
				},
			},
		},
		{
			Domain: "customers",
			Searches: []model.SearchDefinition{
				{
					ID:           "customers.search",
					Domain:       "customers",
					Capabilities: []string{"customers:search:execute"},
					Operation:    model.OperationBinding{Type: "openapi", OperationID: "searchCustomers"},
					ResultMapping: model.SearchResultMapping{
						ItemsPath:     "results",
						TitleField:    "name",
						SubtitleField: "email",
						Route:         "/customers/{id}",
						IDField:       "id",
					},
					Weight:     5,
					MaxResults: 3,
				},
			},
		},
	}
}

func ordersResponse() model.InvocationResult {
	return model.InvocationResult{
		StatusCode: 200,
		Body: map[string]any{
			"data": map[string]any{
				"results": []any{
					map[string]any{"id": "ord-001", "orderNumber": "ORD-2024-001", "customerName": "ACME Corp"},
					map[string]any{"id": "ord-002", "orderNumber": "ORD-2024-002", "customerName": "ACME Industries"},
				},
			},
		},
	}
}

func customersResponse() model.InvocationResult {
	return model.InvocationResult{
		StatusCode: 200,
		Body: map[string]any{
			"results": []any{
				map[string]any{"id": "cust-001", "name": "ACME Corp", "email": "info@acme.com"},
			},
		},
	}
}

func newTestProvider(inv *mockSearchInvoker) *SearchProvider {
	reg := definition.NewRegistry(testSearchDefinitions())
	invReg := invoker.NewRegistry()
	invReg.Register(inv)
	return NewSearchProvider(reg, invReg, 3*time.Second, 50)
}

// --- Search tests ---

func TestSearchProvider_Search_success(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(b model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			switch b.OperationID {
			case "searchOrders":
				return ordersResponse(), nil
			case "searchCustomers":
				return customersResponse(), nil
			}
			return model.InvocationResult{StatusCode: 200}, nil
		},
	}
	sp := newTestProvider(inv)
	caps := model.CapabilitySet{
		"orders:search:execute":    true,
		"customers:search:execute": true,
	}

	resp, err := sp.Search(context.Background(), testRctx(), caps, "ACME", model.Pagination{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if resp.Data.Query != "ACME" {
		t.Errorf("Query = %q", resp.Data.Query)
	}
	if resp.Data.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", resp.Data.TotalCount)
	}
	if len(resp.Data.Results) != 3 {
		t.Fatalf("Results count = %d, want 3", len(resp.Data.Results))
	}

	// Verify results are sorted by score (orders weight=10 > customers weight=5).
	if resp.Data.Results[0].Category != "orders" {
		t.Errorf("top result category = %q, want orders", resp.Data.Results[0].Category)
	}

	// Verify result mapping.
	first := resp.Data.Results[0]
	if first.Title != "ORD-2024-001" {
		t.Errorf("first.Title = %q", first.Title)
	}
	if first.Subtitle != "ACME Corp" {
		t.Errorf("first.Subtitle = %q", first.Subtitle)
	}
	if first.Route != "/orders/ord-001" {
		t.Errorf("first.Route = %q", first.Route)
	}
	if first.ID != "ord-001" {
		t.Errorf("first.ID = %q", first.ID)
	}

	// Verify metadata.
	providers, ok := resp.Meta["providers"].(map[string]string)
	if !ok {
		t.Fatalf("Meta[providers] type = %T", resp.Meta["providers"])
	}
	if providers["orders.search"] != "ok" {
		t.Errorf("providers[orders.search] = %q", providers["orders.search"])
	}
	if providers["customers.search"] != "ok" {
		t.Errorf("providers[customers.search] = %q", providers["customers.search"])
	}
	if resp.Meta["query_time_ms"] == nil {
		t.Error("expected query_time_ms in meta")
	}
}

func TestSearchProvider_Search_queryTooShort(t *testing.T) {
	sp := newTestProvider(&mockSearchInvoker{})
	caps := model.CapabilitySet{"*": true}

	_, err := sp.Search(context.Background(), testRctx(), caps, "A", model.Pagination{Page: 1, PageSize: 20})
	if err == nil {
		t.Fatal("expected bad request error")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrBadRequest {
		t.Errorf("code = %s", envErr.Code)
	}
}

func TestSearchProvider_Search_capabilityFilter(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(b model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			switch b.OperationID {
			case "searchOrders":
				return ordersResponse(), nil
			case "searchCustomers":
				return customersResponse(), nil
			}
			return model.InvocationResult{StatusCode: 200}, nil
		},
	}
	sp := newTestProvider(inv)
	// Only has orders capability, not customers.
	caps := model.CapabilitySet{"orders:search:execute": true}

	resp, err := sp.Search(context.Background(), testRctx(), caps, "ACME", model.Pagination{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	// Should only return orders results.
	if resp.Data.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2 (only orders)", resp.Data.TotalCount)
	}
	for _, r := range resp.Data.Results {
		if r.Category != "orders" {
			t.Errorf("unexpected category %q (should be filtered)", r.Category)
		}
	}
}

func TestSearchProvider_Search_domainFilter(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(b model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			if b.OperationID == "searchCustomers" {
				return customersResponse(), nil
			}
			return ordersResponse(), nil
		},
	}
	sp := newTestProvider(inv)
	caps := model.CapabilitySet{"*": true}

	resp, err := sp.Search(context.Background(), testRctx(), caps, "ACME", model.Pagination{
		Page: 1, PageSize: 20, Domain: "customers",
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if resp.Data.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1 (customers only)", resp.Data.TotalCount)
	}
	if len(resp.Data.Results) > 0 && resp.Data.Results[0].Category != "customers" {
		t.Errorf("category = %q, want customers", resp.Data.Results[0].Category)
	}
}

func TestSearchProvider_Search_providerFailure(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(b model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			if b.OperationID == "searchOrders" {
				return ordersResponse(), nil
			}
			// Customers provider fails.
			return model.InvocationResult{}, fmt.Errorf("backend unavailable")
		},
	}
	sp := newTestProvider(inv)
	caps := model.CapabilitySet{"*": true}

	resp, err := sp.Search(context.Background(), testRctx(), caps, "ACME", model.Pagination{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("Search error: %v (should not fail for provider error)", err)
	}
	// Should still have orders results.
	if resp.Data.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2 (only orders succeeded)", resp.Data.TotalCount)
	}

	// Provider metadata should show the failure.
	providers, ok := resp.Meta["providers"].(map[string]string)
	if !ok {
		t.Fatalf("Meta[providers] type = %T", resp.Meta["providers"])
	}
	if providers["orders.search"] != "ok" {
		t.Errorf("providers[orders.search] = %q", providers["orders.search"])
	}
	if providers["customers.search"] != "error" {
		t.Errorf("providers[customers.search] = %q, want error", providers["customers.search"])
	}
}

func TestSearchProvider_Search_providerTimeout(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(b model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			if b.OperationID == "searchCustomers" {
				// Simulate slow provider.
				time.Sleep(200 * time.Millisecond)
				return model.InvocationResult{}, context.DeadlineExceeded
			}
			return ordersResponse(), nil
		},
	}
	sp := NewSearchProvider(
		definition.NewRegistry(testSearchDefinitions()),
		func() *invoker.Registry {
			r := invoker.NewRegistry()
			r.Register(inv)
			return r
		}(),
		100*time.Millisecond, // Very short timeout.
		50,
	)
	caps := model.CapabilitySet{"*": true}

	resp, err := sp.Search(context.Background(), testRctx(), caps, "ACME", model.Pagination{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	// Orders should succeed, customers should timeout.
	if resp.Data.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", resp.Data.TotalCount)
	}
}

func TestSearchProvider_Search_allProvidersFail(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			return model.InvocationResult{}, fmt.Errorf("everything is broken")
		},
	}
	sp := newTestProvider(inv)
	caps := model.CapabilitySet{"*": true}

	resp, err := sp.Search(context.Background(), testRctx(), caps, "ACME", model.Pagination{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("Search should not error when all providers fail: %v", err)
	}
	if resp.Data.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0", resp.Data.TotalCount)
	}
	if len(resp.Data.Results) != 0 {
		t.Errorf("Results count = %d, want 0", len(resp.Data.Results))
	}
}

func TestSearchProvider_Search_pagination(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(b model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			if b.OperationID == "searchOrders" {
				return ordersResponse(), nil
			}
			return customersResponse(), nil
		},
	}
	sp := newTestProvider(inv)
	caps := model.CapabilitySet{"*": true}

	// Page 1, size 2.
	resp, err := sp.Search(context.Background(), testRctx(), caps, "ACME", model.Pagination{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if resp.Data.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", resp.Data.TotalCount)
	}
	if len(resp.Data.Results) != 2 {
		t.Errorf("Results count = %d, want 2", len(resp.Data.Results))
	}

	// Page 2.
	resp2, err := sp.Search(context.Background(), testRctx(), caps, "ACME", model.Pagination{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(resp2.Data.Results) != 1 {
		t.Errorf("Page 2 results = %d, want 1", len(resp2.Data.Results))
	}

	// Page beyond.
	resp3, err := sp.Search(context.Background(), testRctx(), caps, "ACME", model.Pagination{Page: 10, PageSize: 2})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(resp3.Data.Results) != 0 {
		t.Errorf("Beyond page results = %d, want 0", len(resp3.Data.Results))
	}
}

func TestSearchProvider_Search_paginationDefaults(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			return ordersResponse(), nil
		},
	}
	sp := newTestProvider(inv)
	caps := model.CapabilitySet{"*": true}

	// Zero values should get defaults.
	resp, err := sp.Search(context.Background(), testRctx(), caps, "test", model.Pagination{})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(resp.Data.Results) == 0 {
		t.Error("expected results with default pagination")
	}
}

func TestSearchProvider_Search_maxResults(t *testing.T) {
	// Create a provider that returns many results.
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			items := make([]any, 20)
			for i := range items {
				items[i] = map[string]any{
					"id":          fmt.Sprintf("ord-%03d", i),
					"orderNumber": fmt.Sprintf("ORD-%03d", i),
				}
			}
			return model.InvocationResult{
				StatusCode: 200,
				Body:       map[string]any{"data": map[string]any{"results": items}},
			}, nil
		},
	}
	sp := newTestProvider(inv)
	caps := model.CapabilitySet{"*": true}

	resp, err := sp.Search(context.Background(), testRctx(), caps, "ORD", model.Pagination{Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	// Orders max_results is 5, so only 5 should come from orders.
	ordersCount := 0
	for _, r := range resp.Data.Results {
		if r.Category == "orders" {
			ordersCount++
		}
	}
	if ordersCount > 5 {
		t.Errorf("orders results = %d, want <= 5 (max_results)", ordersCount)
	}
}

func TestSearchProvider_Search_queryPassedToBackend(t *testing.T) {
	var mu sync.Mutex
	var capturedInputs []model.InvocationInput
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
			mu.Lock()
			capturedInputs = append(capturedInputs, input)
			mu.Unlock()
			return model.InvocationResult{StatusCode: 200, Body: map[string]any{"data": map[string]any{"results": []any{}}}}, nil
		},
	}
	sp := newTestProvider(inv)
	caps := model.CapabilitySet{"*": true}

	_, _ = sp.Search(context.Background(), testRctx(), caps, "test query", model.Pagination{Page: 1, PageSize: 20})

	mu.Lock()
	defer mu.Unlock()
	for _, input := range capturedInputs {
		if input.QueryParams["q"] != "test query" {
			t.Errorf("query param q = %q, want 'test query'", input.QueryParams["q"])
		}
	}
	if len(capturedInputs) == 0 {
		t.Error("expected at least one invocation")
	}
}

func TestSearchProvider_Search_backendNon2xx(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			return model.InvocationResult{StatusCode: 500, Body: map[string]any{"error": "internal"}}, nil
		},
	}
	sp := newTestProvider(inv)
	caps := model.CapabilitySet{"*": true}

	resp, err := sp.Search(context.Background(), testRctx(), caps, "test", model.Pagination{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("Search should not error: %v", err)
	}
	if resp.Data.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0", resp.Data.TotalCount)
	}
}

// --- Helper function tests ---

func TestExtractItems(t *testing.T) {
	body := map[string]any{
		"data": map[string]any{
			"results": []any{
				map[string]any{"id": "1", "name": "Alice"},
				map[string]any{"id": "2", "name": "Bob"},
			},
		},
	}

	items := extractItems(body, "data.results")
	if len(items) != 2 {
		t.Fatalf("items count = %d, want 2", len(items))
	}
	if items[0]["name"] != "Alice" {
		t.Errorf("items[0][name] = %v", items[0]["name"])
	}
}

func TestExtractItems_emptyPath(t *testing.T) {
	body := []any{
		map[string]any{"id": "1"},
	}
	items := extractItems(body, "")
	if len(items) != 1 {
		t.Errorf("items count = %d, want 1", len(items))
	}
}

func TestExtractItems_invalidPath(t *testing.T) {
	body := map[string]any{"foo": "bar"}
	items := extractItems(body, "nonexistent.path")
	if items != nil {
		t.Errorf("expected nil for invalid path, got %v", items)
	}
}

func TestMapResults_scoring(t *testing.T) {
	items := []map[string]any{
		{"id": "1", "title": "First"},
		{"id": "2", "title": "Second"},
		{"id": "3", "title": "Third"},
	}
	def := model.SearchDefinition{
		Domain: "test",
		Weight: 10,
		ResultMapping: model.SearchResultMapping{
			IDField:    "id",
			TitleField: "title",
			Route:      "/test/{id}",
		},
	}

	results := mapResults(items, def, 10)
	if len(results) != 3 {
		t.Fatalf("results count = %d", len(results))
	}

	// First result should have highest score.
	if results[0].Score <= results[1].Score {
		t.Errorf("first score (%f) should be > second score (%f)", results[0].Score, results[1].Score)
	}
	if results[1].Score <= results[2].Score {
		t.Errorf("second score (%f) should be > third score (%f)", results[1].Score, results[2].Score)
	}

	// Verify route resolution.
	if results[0].Route != "/test/1" {
		t.Errorf("route = %q", results[0].Route)
	}
}

func TestMapResults_defaultWeight(t *testing.T) {
	items := []map[string]any{{"id": "1", "title": "Test"}}
	def := model.SearchDefinition{
		Weight:        0, // Should default to 1.
		ResultMapping: model.SearchResultMapping{IDField: "id", TitleField: "title", Route: "/t/{id}"},
	}

	results := mapResults(items, def, 10)
	if results[0].Score != 1.0 {
		t.Errorf("score = %f, want 1.0 (weight=1, position=1.0)", results[0].Score)
	}
}

func TestDeduplicate(t *testing.T) {
	results := []model.SearchResult{
		{ID: "1", Route: "/test/1", Score: 5.0, Category: "a"},
		{ID: "1", Route: "/test/1", Score: 8.0, Category: "b"},
		{ID: "2", Route: "/test/2", Score: 3.0, Category: "a"},
	}

	deduped := deduplicate(results)
	if len(deduped) != 2 {
		t.Fatalf("deduped count = %d, want 2", len(deduped))
	}

	// The duplicate should keep the higher score.
	for _, r := range deduped {
		if r.ID == "1" && r.Score != 8.0 {
			t.Errorf("duplicate kept score %f, want 8.0", r.Score)
		}
	}
}

func TestDeduplicate_empty(t *testing.T) {
	var results []model.SearchResult
	deduped := deduplicate(results)
	if deduped != nil {
		t.Errorf("expected nil for empty input, got %v", deduped)
	}
}

func TestResolveRoute(t *testing.T) {
	tests := []struct {
		template string
		id       string
		want     string
	}{
		{"/orders/{id}", "ord-001", "/orders/ord-001"},
		{"/orders/{id}/items", "ord-001", "/orders/ord-001/items"},
		{"/simple", "id", "/simple"},
	}
	for _, tt := range tests {
		got := resolveRoute(tt.template, tt.id)
		if got != tt.want {
			t.Errorf("resolveRoute(%q, %q) = %q, want %q", tt.template, tt.id, got, tt.want)
		}
	}
}

func TestGetString(t *testing.T) {
	m := map[string]any{"name": "Alice", "count": 42, "nil_val": nil}

	if getString(m, "name") != "Alice" {
		t.Error("expected Alice")
	}
	if getString(m, "count") != "42" {
		t.Error("expected 42 as string")
	}
	if getString(m, "missing") != "" {
		t.Error("expected empty for missing")
	}
	if getString(m, "") != "" {
		t.Error("expected empty for empty key")
	}
	if getString(m, "nil_val") != "" {
		t.Error("expected empty for nil value")
	}
}
