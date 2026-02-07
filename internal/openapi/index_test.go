package openapi

import (
	"testing"
)

func loadTestIndex(t *testing.T) *Index {
	t.Helper()
	idx := NewIndex()
	err := idx.Load([]SpecSource{
		{ServiceID: "orders-svc", BaseURL: "https://orders.internal", SpecPath: "testdata/orders-svc.yaml"},
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	return idx
}

func TestIndex_Load(t *testing.T) {
	idx := loadTestIndex(t)
	ids := idx.AllOperationIDs("orders-svc")
	if len(ids) != 5 {
		t.Fatalf("AllOperationIDs() = %v (len %d), want 5 operations", ids, len(ids))
	}
}

func TestIndex_GetOperation_found(t *testing.T) {
	idx := loadTestIndex(t)

	op, ok := idx.GetOperation("orders-svc", "listOrders")
	if !ok {
		t.Fatal("GetOperation(listOrders) not found")
	}
	if op.Method != "GET" {
		t.Errorf("Method = %q, want GET", op.Method)
	}
	if op.PathTemplate != "/orders" {
		t.Errorf("PathTemplate = %q, want /orders", op.PathTemplate)
	}
	if op.BaseURL != "https://orders.internal" {
		t.Errorf("BaseURL = %q, want https://orders.internal", op.BaseURL)
	}
}

func TestIndex_GetOperation_with_path_params(t *testing.T) {
	idx := loadTestIndex(t)

	op, ok := idx.GetOperation("orders-svc", "getOrder")
	if !ok {
		t.Fatal("GetOperation(getOrder) not found")
	}
	if op.Method != "GET" {
		t.Errorf("Method = %q, want GET", op.Method)
	}
	if op.PathTemplate != "/orders/{orderId}" {
		t.Errorf("PathTemplate = %q, want /orders/{orderId}", op.PathTemplate)
	}

	// Should have path-level orderId parameter.
	found := false
	for _, p := range op.Parameters {
		if p.Name == "orderId" && p.In == "path" {
			found = true
		}
	}
	if !found {
		t.Error("Expected orderId path parameter not found")
	}
}

func TestIndex_GetOperation_not_found(t *testing.T) {
	idx := loadTestIndex(t)

	_, ok := idx.GetOperation("orders-svc", "nonexistent")
	if ok {
		t.Error("GetOperation(nonexistent) should return false")
	}

	_, ok = idx.GetOperation("unknown-svc", "listOrders")
	if ok {
		t.Error("GetOperation(unknown-svc) should return false")
	}
}

func TestIndex_AllOperationIDs(t *testing.T) {
	idx := loadTestIndex(t)

	ids := idx.AllOperationIDs("orders-svc")
	expected := []string{"createOrder", "getOrder", "listOrders", "searchOrders", "updateOrder"}
	if len(ids) != len(expected) {
		t.Fatalf("AllOperationIDs() = %v, want %v", ids, expected)
	}
	for i, id := range ids {
		if id != expected[i] {
			t.Errorf("ids[%d] = %q, want %q", i, id, expected[i])
		}
	}
}

func TestIndex_AllOperationIDs_unknown(t *testing.T) {
	idx := loadTestIndex(t)
	ids := idx.AllOperationIDs("unknown-svc")
	if len(ids) != 0 {
		t.Errorf("AllOperationIDs(unknown-svc) = %v, want empty", ids)
	}
}

func TestIndex_ValidateRequest_valid(t *testing.T) {
	idx := loadTestIndex(t)
	errs := idx.ValidateRequest("orders-svc", "createOrder", map[string]any{
		"customer_id": "cust-1",
		"items":       []any{},
	})
	if len(errs) != 0 {
		t.Errorf("ValidateRequest() = %v, want no errors", errs)
	}
}

func TestIndex_ValidateRequest_missing_required(t *testing.T) {
	idx := loadTestIndex(t)
	errs := idx.ValidateRequest("orders-svc", "createOrder", map[string]any{
		"notes": "hello",
	})
	if len(errs) != 2 {
		t.Fatalf("ValidateRequest() = %v (len %d), want 2 errors", errs, len(errs))
	}
}

func TestIndex_ValidateRequest_no_body(t *testing.T) {
	idx := loadTestIndex(t)
	errs := idx.ValidateRequest("orders-svc", "listOrders", map[string]any{})
	if len(errs) != 0 {
		t.Errorf("ValidateRequest(listOrders) = %v, want no errors (no request body)", errs)
	}
}

func TestIndex_ValidateRequest_unknown_operation(t *testing.T) {
	idx := loadTestIndex(t)
	errs := idx.ValidateRequest("orders-svc", "nonexistent", map[string]any{})
	if len(errs) != 1 {
		t.Errorf("ValidateRequest(nonexistent) = %v, want 1 error", errs)
	}
}

func TestIndex_Load_bad_file(t *testing.T) {
	idx := NewIndex()
	err := idx.Load([]SpecSource{
		{ServiceID: "bad-svc", SpecPath: "testdata/nonexistent.yaml"},
	})
	if err == nil {
		t.Fatal("Load() with bad file should return error")
	}
}

func TestIndex_BaseURL_from_spec(t *testing.T) {
	idx := NewIndex()
	err := idx.Load([]SpecSource{
		{ServiceID: "orders-svc", SpecPath: "testdata/orders-svc.yaml"}, // no BaseURL provided
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	op, ok := idx.GetOperation("orders-svc", "listOrders")
	if !ok {
		t.Fatal("GetOperation(listOrders) not found")
	}
	if op.BaseURL != "https://orders.internal" {
		t.Errorf("BaseURL = %q, want https://orders.internal (from spec servers)", op.BaseURL)
	}
}
