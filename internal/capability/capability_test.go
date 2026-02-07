package capability

import (
	"testing"
	"time"

	"github.com/pitabwire/thesa/model"
)

func testRctx(roles ...string) *model.RequestContext {
	return &model.RequestContext{
		SubjectID:   "user-1",
		TenantID:    "tenant-1",
		PartitionID: "part-1",
		Roles:       roles,
	}
}

// --- StaticPolicyEvaluator tests ---

func TestStaticPolicyEvaluator_ResolveCapabilities(t *testing.T) {
	e, err := NewStaticPolicyEvaluator("testdata/policies.yaml")
	if err != nil {
		t.Fatalf("NewStaticPolicyEvaluator() error = %v", err)
	}

	caps, err := e.ResolveCapabilities(testRctx("order_viewer"))
	if err != nil {
		t.Fatalf("ResolveCapabilities() error = %v", err)
	}

	if !caps.Has("orders:list:view") {
		t.Error("order_viewer should have orders:list:view")
	}
	if caps.Has("orders:detail:edit") {
		t.Error("order_viewer should not have orders:detail:edit")
	}
}

func TestStaticPolicyEvaluator_MultipleRoles(t *testing.T) {
	e, _ := NewStaticPolicyEvaluator("testdata/policies.yaml")
	caps, _ := e.ResolveCapabilities(testRctx("order_viewer", "order_manager"))

	if !caps.Has("orders:detail:edit") {
		t.Error("order_manager should add orders:detail:edit")
	}
	if !caps.Has("orders:list:view") {
		t.Error("combined roles should have orders:list:view")
	}
}

func TestStaticPolicyEvaluator_Wildcard(t *testing.T) {
	e, _ := NewStaticPolicyEvaluator("testdata/policies.yaml")
	caps, _ := e.ResolveCapabilities(testRctx("admin"))

	if !caps.Has("orders:anything:at:all") {
		t.Error("admin with orders:* should match any orders: capability")
	}
	if !caps.Has("inventory:list:view") {
		t.Error("admin with inventory:* should match inventory:list:view")
	}
}

func TestStaticPolicyEvaluator_UnknownRole(t *testing.T) {
	e, _ := NewStaticPolicyEvaluator("testdata/policies.yaml")
	caps, _ := e.ResolveCapabilities(testRctx("nonexistent"))

	if len(caps) != 0 {
		t.Errorf("unknown role should return empty capabilities, got %v", caps)
	}
}

func TestStaticPolicyEvaluator_Evaluate(t *testing.T) {
	e, _ := NewStaticPolicyEvaluator("testdata/policies.yaml")
	ok, err := e.Evaluate(testRctx("order_viewer"), "orders:list:view", nil)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !ok {
		t.Error("Evaluate(orders:list:view) = false, want true")
	}

	ok, _ = e.Evaluate(testRctx("order_viewer"), "orders:cancel:execute", nil)
	if ok {
		t.Error("Evaluate(orders:cancel:execute) = true, want false for viewer")
	}
}

func TestStaticPolicyEvaluator_EvaluateAll(t *testing.T) {
	e, _ := NewStaticPolicyEvaluator("testdata/policies.yaml")
	result, err := e.EvaluateAll(testRctx("order_viewer"),
		[]string{"orders:list:view", "orders:cancel:execute"}, nil)
	if err != nil {
		t.Fatalf("EvaluateAll() error = %v", err)
	}
	if !result["orders:list:view"] {
		t.Error("EvaluateAll: orders:list:view should be true")
	}
	if result["orders:cancel:execute"] {
		t.Error("EvaluateAll: orders:cancel:execute should be false for viewer")
	}
}

func TestStaticPolicyEvaluator_BadFile(t *testing.T) {
	_, err := NewStaticPolicyEvaluator("testdata/nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing policy file")
	}
}

// --- Resolver tests ---

func TestResolver_Resolve_and_Cache(t *testing.T) {
	e, _ := NewStaticPolicyEvaluator("testdata/policies.yaml")
	r := NewResolver(e, 5*time.Minute)

	rctx := testRctx("order_viewer")

	// First call — cache miss.
	caps1, err := r.Resolve(rctx)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !caps1.Has("orders:list:view") {
		t.Error("should have orders:list:view")
	}

	// Second call — cache hit (same result).
	caps2, err := r.Resolve(rctx)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !caps2.Has("orders:list:view") {
		t.Error("cached result should have orders:list:view")
	}
}

func TestResolver_Invalidate(t *testing.T) {
	callCount := 0
	mock := &mockEvaluator{
		resolveFunc: func(rctx *model.RequestContext) (model.CapabilitySet, error) {
			callCount++
			return model.CapabilitySet{"orders:list:view": true}, nil
		},
	}
	r := NewResolver(mock, 5*time.Minute)
	rctx := testRctx()

	r.Resolve(rctx)
	if callCount != 1 {
		t.Fatalf("callCount = %d, want 1", callCount)
	}

	r.Resolve(rctx)
	if callCount != 1 {
		t.Fatalf("callCount = %d after cache hit, want 1", callCount)
	}

	r.Invalidate("user-1", "tenant-1")

	r.Resolve(rctx)
	if callCount != 2 {
		t.Fatalf("callCount = %d after invalidate, want 2", callCount)
	}
}

func TestResolver_TTLExpiry(t *testing.T) {
	callCount := 0
	mock := &mockEvaluator{
		resolveFunc: func(rctx *model.RequestContext) (model.CapabilitySet, error) {
			callCount++
			return model.CapabilitySet{"orders:list:view": true}, nil
		},
	}
	r := NewResolver(mock, 1*time.Millisecond) // very short TTL
	rctx := testRctx()

	r.Resolve(rctx)
	time.Sleep(5 * time.Millisecond)
	r.Resolve(rctx) // should be expired

	if callCount != 2 {
		t.Fatalf("callCount = %d, want 2 (TTL expired)", callCount)
	}
}

// --- Mock PolicyEvaluator ---

type mockEvaluator struct {
	resolveFunc func(rctx *model.RequestContext) (model.CapabilitySet, error)
}

func (m *mockEvaluator) ResolveCapabilities(rctx *model.RequestContext) (model.CapabilitySet, error) {
	return m.resolveFunc(rctx)
}

func (m *mockEvaluator) Evaluate(*model.RequestContext, string, map[string]any) (bool, error) {
	return false, nil
}

func (m *mockEvaluator) EvaluateAll(*model.RequestContext, []string, map[string]any) (map[string]bool, error) {
	return nil, nil
}

func (m *mockEvaluator) Sync() error { return nil }
