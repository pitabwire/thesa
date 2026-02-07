package search

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/model"
)

func testLookupDefinitions() []model.DomainDefinition {
	return []model.DomainDefinition{
		{
			Domain: "orders",
			Lookups: []model.LookupDefinition{
				{
					ID:         "orders.statuses",
					Operation:  model.OperationBinding{Type: "openapi", OperationID: "getStatuses"},
					LabelField: "name",
					ValueField: "code",
					Cache: &model.CacheConfig{
						TTL:   "10m",
						Scope: "global",
					},
				},
				{
					ID:         "orders.categories",
					Operation:  model.OperationBinding{Type: "openapi", OperationID: "getCategories"},
					LabelField: "label",
					ValueField: "id",
					Cache: &model.CacheConfig{
						TTL:   "5m",
						Scope: "tenant",
					},
				},
				{
					ID:         "orders.warehouses",
					Operation:  model.OperationBinding{Type: "openapi", OperationID: "getWarehouses"},
					LabelField: "name",
					ValueField: "id",
					Cache: &model.CacheConfig{
						TTL:   "2m",
						Scope: "partition",
					},
				},
				{
					ID:         "orders.no-cache",
					Operation:  model.OperationBinding{Type: "openapi", OperationID: "getThings"},
					LabelField: "name",
					ValueField: "id",
					// No cache config — uses defaults.
				},
			},
		},
	}
}

func statusesResponse() model.InvocationResult {
	return model.InvocationResult{
		StatusCode: 200,
		Body: []any{
			map[string]any{"code": "pending", "name": "Pending"},
			map[string]any{"code": "active", "name": "Active"},
			map[string]any{"code": "completed", "name": "Completed"},
		},
	}
}

func categoriesResponse() model.InvocationResult {
	return model.InvocationResult{
		StatusCode: 200,
		Body: map[string]any{
			"data": []any{
				map[string]any{"id": "cat-1", "label": "Electronics"},
				map[string]any{"id": "cat-2", "label": "Furniture"},
			},
		},
	}
}

func newTestLookupProvider(inv *mockSearchInvoker) *LookupProvider {
	reg := definition.NewRegistry(testLookupDefinitions())
	invReg := invoker.NewRegistry()
	invReg.Register(inv)
	return NewLookupProvider(reg, invReg, 5*time.Minute, 100)
}

// --- GetLookup tests ---

func TestLookupProvider_GetLookup_success(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			return statusesResponse(), nil
		},
	}
	lp := newTestLookupProvider(inv)
	ctx := context.Background()
	rctx := testRctx()

	resp, err := lp.GetLookup(ctx, rctx, "orders.statuses", "")
	if err != nil {
		t.Fatalf("GetLookup error: %v", err)
	}
	if len(resp.Data.Options) != 3 {
		t.Fatalf("Options count = %d, want 3", len(resp.Data.Options))
	}
	if resp.Data.Options[0].Label != "Pending" {
		t.Errorf("Options[0].Label = %q", resp.Data.Options[0].Label)
	}
	if resp.Data.Options[0].Value != "pending" {
		t.Errorf("Options[0].Value = %q", resp.Data.Options[0].Value)
	}
	// First call should not be cached.
	if resp.Meta["cached"] != false {
		t.Errorf("Meta[cached] = %v, want false", resp.Meta["cached"])
	}
}

func TestLookupProvider_GetLookup_cached(t *testing.T) {
	callCount := 0
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			callCount++
			return statusesResponse(), nil
		},
	}
	lp := newTestLookupProvider(inv)
	ctx := context.Background()
	rctx := testRctx()

	// First call.
	_, _ = lp.GetLookup(ctx, rctx, "orders.statuses", "")
	if callCount != 1 {
		t.Fatalf("expected 1 backend call, got %d", callCount)
	}

	// Second call should be cached.
	resp, err := lp.GetLookup(ctx, rctx, "orders.statuses", "")
	if err != nil {
		t.Fatalf("GetLookup error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected still 1 backend call after cache hit, got %d", callCount)
	}
	if resp.Meta["cached"] != true {
		t.Errorf("Meta[cached] = %v, want true", resp.Meta["cached"])
	}
	if len(resp.Data.Options) != 3 {
		t.Errorf("Options count = %d, want 3", len(resp.Data.Options))
	}
}

func TestLookupProvider_GetLookup_notFound(t *testing.T) {
	lp := newTestLookupProvider(&mockSearchInvoker{})
	ctx := context.Background()
	rctx := testRctx()

	_, err := lp.GetLookup(ctx, rctx, "nonexistent", "")
	if err == nil {
		t.Fatal("expected not found error")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrNotFound {
		t.Errorf("code = %s", envErr.Code)
	}
}

func TestLookupProvider_GetLookup_queryFilter(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			return statusesResponse(), nil
		},
	}
	lp := newTestLookupProvider(inv)
	ctx := context.Background()
	rctx := testRctx()

	resp, err := lp.GetLookup(ctx, rctx, "orders.statuses", "act")
	if err != nil {
		t.Fatalf("GetLookup error: %v", err)
	}
	if len(resp.Data.Options) != 1 {
		t.Fatalf("Options count = %d, want 1 (Active matches 'act')", len(resp.Data.Options))
	}
	if resp.Data.Options[0].Label != "Active" {
		t.Errorf("Options[0].Label = %q", resp.Data.Options[0].Label)
	}
}

func TestLookupProvider_GetLookup_queryFilterCaseInsensitive(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			return statusesResponse(), nil
		},
	}
	lp := newTestLookupProvider(inv)
	ctx := context.Background()
	rctx := testRctx()

	resp, err := lp.GetLookup(ctx, rctx, "orders.statuses", "PEND")
	if err != nil {
		t.Fatalf("GetLookup error: %v", err)
	}
	if len(resp.Data.Options) != 1 {
		t.Fatalf("Options count = %d, want 1", len(resp.Data.Options))
	}
	if resp.Data.Options[0].Label != "Pending" {
		t.Errorf("Options[0].Label = %q", resp.Data.Options[0].Label)
	}
}

func TestLookupProvider_GetLookup_queryFilterOnCache(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			return statusesResponse(), nil
		},
	}
	lp := newTestLookupProvider(inv)
	ctx := context.Background()
	rctx := testRctx()

	// First call populates cache with all options.
	_, _ = lp.GetLookup(ctx, rctx, "orders.statuses", "")

	// Second call with query should filter cached results.
	resp, err := lp.GetLookup(ctx, rctx, "orders.statuses", "comp")
	if err != nil {
		t.Fatalf("GetLookup error: %v", err)
	}
	if resp.Meta["cached"] != true {
		t.Errorf("expected cached hit")
	}
	if len(resp.Data.Options) != 1 {
		t.Fatalf("Options count = %d, want 1 (Completed)", len(resp.Data.Options))
	}
}

func TestLookupProvider_GetLookup_dataWrapper(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			return categoriesResponse(), nil
		},
	}
	lp := newTestLookupProvider(inv)
	ctx := context.Background()
	rctx := testRctx()

	resp, err := lp.GetLookup(ctx, rctx, "orders.categories", "")
	if err != nil {
		t.Fatalf("GetLookup error: %v", err)
	}
	if len(resp.Data.Options) != 2 {
		t.Fatalf("Options count = %d, want 2", len(resp.Data.Options))
	}
	if resp.Data.Options[0].Label != "Electronics" {
		t.Errorf("Options[0].Label = %q", resp.Data.Options[0].Label)
	}
}

func TestLookupProvider_GetLookup_backendError(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			return model.InvocationResult{}, fmt.Errorf("connection refused")
		},
	}
	lp := newTestLookupProvider(inv)
	ctx := context.Background()
	rctx := testRctx()

	_, err := lp.GetLookup(ctx, rctx, "orders.statuses", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLookupProvider_GetLookup_backendNon2xx(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			return model.InvocationResult{StatusCode: 500}, nil
		},
	}
	lp := newTestLookupProvider(inv)
	ctx := context.Background()
	rctx := testRctx()

	_, err := lp.GetLookup(ctx, rctx, "orders.statuses", "")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

// --- Cache scoping tests ---

func TestLookupProvider_CacheScope_global(t *testing.T) {
	callCount := 0
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			callCount++
			return statusesResponse(), nil
		},
	}
	lp := newTestLookupProvider(inv)
	ctx := context.Background()

	rctx1 := &model.RequestContext{SubjectID: "alice", TenantID: "tenant-1"}
	rctx2 := &model.RequestContext{SubjectID: "bob", TenantID: "tenant-2"}

	_, _ = lp.GetLookup(ctx, rctx1, "orders.statuses", "")
	_, _ = lp.GetLookup(ctx, rctx2, "orders.statuses", "")

	// Global scope: both tenants share the same cache entry.
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (global cache shared)", callCount)
	}
}

func TestLookupProvider_CacheScope_tenant(t *testing.T) {
	callCount := 0
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			callCount++
			return categoriesResponse(), nil
		},
	}
	lp := newTestLookupProvider(inv)
	ctx := context.Background()

	rctx1 := &model.RequestContext{SubjectID: "alice", TenantID: "tenant-1"}
	rctx2 := &model.RequestContext{SubjectID: "bob", TenantID: "tenant-2"}

	_, _ = lp.GetLookup(ctx, rctx1, "orders.categories", "")
	_, _ = lp.GetLookup(ctx, rctx2, "orders.categories", "")

	// Tenant scope: different tenants get separate cache entries.
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (tenant-scoped cache)", callCount)
	}
}

func TestLookupProvider_CacheScope_partition(t *testing.T) {
	callCount := 0
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			callCount++
			return model.InvocationResult{
				StatusCode: 200,
				Body:       []any{map[string]any{"id": "wh-1", "name": "Warehouse 1"}},
			}, nil
		},
	}
	lp := newTestLookupProvider(inv)
	ctx := context.Background()

	rctx1 := &model.RequestContext{SubjectID: "alice", TenantID: "tenant-1", PartitionID: "part-a"}
	rctx2 := &model.RequestContext{SubjectID: "bob", TenantID: "tenant-1", PartitionID: "part-b"}

	_, _ = lp.GetLookup(ctx, rctx1, "orders.warehouses", "")
	_, _ = lp.GetLookup(ctx, rctx2, "orders.warehouses", "")

	// Partition scope: different partitions get separate entries.
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (partition-scoped cache)", callCount)
	}
}

// --- Invalidation ---

func TestLookupProvider_Invalidate(t *testing.T) {
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			return statusesResponse(), nil
		},
	}
	lp := newTestLookupProvider(inv)
	ctx := context.Background()
	rctx := testRctx()

	_, _ = lp.GetLookup(ctx, rctx, "orders.statuses", "")
	if lp.CacheLen() != 1 {
		t.Fatalf("CacheLen = %d, want 1", lp.CacheLen())
	}

	lp.Invalidate("orders.statuses", "")
	if lp.CacheLen() != 0 {
		t.Errorf("CacheLen after invalidate = %d, want 0", lp.CacheLen())
	}
}

func TestLookupProvider_Invalidate_tenantSpecific(t *testing.T) {
	callCount := 0
	inv := &mockSearchInvoker{
		handler: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			callCount++
			return categoriesResponse(), nil
		},
	}
	lp := newTestLookupProvider(inv)
	ctx := context.Background()

	rctx1 := &model.RequestContext{SubjectID: "alice", TenantID: "tenant-1"}
	rctx2 := &model.RequestContext{SubjectID: "bob", TenantID: "tenant-2"}

	_, _ = lp.GetLookup(ctx, rctx1, "orders.categories", "")
	_, _ = lp.GetLookup(ctx, rctx2, "orders.categories", "")
	if lp.CacheLen() != 2 {
		t.Fatalf("CacheLen = %d, want 2", lp.CacheLen())
	}

	// Invalidate only tenant-1.
	lp.Invalidate("orders.categories", "tenant-1")
	if lp.CacheLen() != 1 {
		t.Errorf("CacheLen after tenant-1 invalidate = %d, want 1", lp.CacheLen())
	}
}

// --- Helper tests ---

func TestFilterOptions(t *testing.T) {
	options := []model.OptionDescriptor{
		{Label: "Active", Value: "active"},
		{Label: "Pending", Value: "pending"},
		{Label: "Completed", Value: "completed"},
	}

	tests := []struct {
		query string
		want  int
	}{
		{"", 3},
		{"act", 1},
		{"PEND", 1},
		{"ed", 1}, // Completed
		{"xyz", 0},
	}

	for _, tt := range tests {
		got := filterOptions(options, tt.query)
		if len(got) != tt.want {
			t.Errorf("filterOptions(%q) = %d, want %d", tt.query, len(got), tt.want)
		}
	}
}

func TestMapLookupResults_directArray(t *testing.T) {
	body := []any{
		map[string]any{"code": "a", "name": "Alpha"},
		map[string]any{"code": "b", "name": "Beta"},
	}
	def := model.LookupDefinition{LabelField: "name", ValueField: "code"}

	options := mapLookupResults(body, def)
	if len(options) != 2 {
		t.Fatalf("options count = %d, want 2", len(options))
	}
	if options[0].Label != "Alpha" || options[0].Value != "a" {
		t.Errorf("options[0] = {%q, %q}", options[0].Label, options[0].Value)
	}
}

func TestMapLookupResults_dataWrapper(t *testing.T) {
	body := map[string]any{
		"data": []any{
			map[string]any{"id": "1", "label": "One"},
		},
	}
	def := model.LookupDefinition{LabelField: "label", ValueField: "id"}

	options := mapLookupResults(body, def)
	if len(options) != 1 {
		t.Fatalf("options count = %d, want 1", len(options))
	}
}

func TestMapLookupResults_itemsWrapper(t *testing.T) {
	body := map[string]any{
		"items": []any{
			map[string]any{"id": "1", "name": "Item One"},
		},
	}
	def := model.LookupDefinition{LabelField: "name", ValueField: "id"}

	options := mapLookupResults(body, def)
	if len(options) != 1 {
		t.Fatalf("options count = %d, want 1", len(options))
	}
}

func TestMapLookupResults_skipsEmptyItems(t *testing.T) {
	body := []any{
		map[string]any{"code": "a", "name": "Alpha"},
		map[string]any{"other": "data"}, // No label or value field.
	}
	def := model.LookupDefinition{LabelField: "name", ValueField: "code"}

	options := mapLookupResults(body, def)
	if len(options) != 1 {
		t.Errorf("options count = %d, want 1 (skip empty)", len(options))
	}
}

func TestExtractLookupItems_nilBody(t *testing.T) {
	items := extractLookupItems(nil)
	if items != nil {
		t.Error("expected nil for nil body")
	}
}

func TestBuildCacheKey(t *testing.T) {
	lp := &LookupProvider{}
	rctx := &model.RequestContext{TenantID: "t1", PartitionID: "p1"}

	tests := []struct {
		def  model.LookupDefinition
		want string
	}{
		{
			def:  model.LookupDefinition{ID: "test", Cache: &model.CacheConfig{Scope: "global"}},
			want: "lookup:test",
		},
		{
			def:  model.LookupDefinition{ID: "test", Cache: &model.CacheConfig{Scope: "tenant"}},
			want: "lookup:test:t1",
		},
		{
			def:  model.LookupDefinition{ID: "test", Cache: &model.CacheConfig{Scope: "partition"}},
			want: "lookup:test:t1:p1",
		},
		{
			def:  model.LookupDefinition{ID: "test"}, // No cache config.
			want: "lookup:test",
		},
	}

	for _, tt := range tests {
		got := lp.buildCacheKey(tt.def, rctx)
		if got != tt.want {
			t.Errorf("buildCacheKey(%v) = %q, want %q", tt.def.Cache, got, tt.want)
		}
	}
}
