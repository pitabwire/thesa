package command

import (
	"testing"

	"github.com/pitabwire/thesa/model"
)

func testResolver() *ExpressionResolver {
	return &ExpressionResolver{
		Input: map[string]any{
			"name":  "Alice",
			"email": "alice@example.com",
			"age":   float64(30),
			"address": map[string]any{
				"city":    "Springfield",
				"country": "US",
				"geo": map[string]any{
					"lat": 39.78,
				},
			},
		},
		RouteParams: map[string]string{
			"id":   "user-123",
			"mode": "edit",
		},
		Context: &model.RequestContext{
			SubjectID:   "sub-456",
			TenantID:    "tenant-acme",
			PartitionID: "partition-1",
			Email:       "admin@acme.com",
		},
		WorkflowState: map[string]any{
			"order_id":      "ord-789",
			"approved_by":   "user-bob",
			"approval_notes": "Looks good",
		},
	}
}

// --- input expressions ---

func TestExpressionResolver_input_simple(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("input.name")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != "Alice" {
		t.Errorf("val = %v, want Alice", val)
	}
}

func TestExpressionResolver_input_numeric(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("input.age")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != float64(30) {
		t.Errorf("val = %v, want 30", val)
	}
}

func TestExpressionResolver_input_nested(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("input.address.city")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != "Springfield" {
		t.Errorf("val = %v, want Springfield", val)
	}
}

func TestExpressionResolver_input_deepNested(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("input.address.geo.lat")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != 39.78 {
		t.Errorf("val = %v, want 39.78", val)
	}
}

func TestExpressionResolver_input_notFound(t *testing.T) {
	r := testResolver()
	_, err := r.Resolve("input.nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent input field")
	}
}

func TestExpressionResolver_input_nil(t *testing.T) {
	r := &ExpressionResolver{}
	_, err := r.Resolve("input.name")
	if err == nil {
		t.Fatal("expected error for nil input")
	}
}

// --- route expressions ---

func TestExpressionResolver_route(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("route.id")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != "user-123" {
		t.Errorf("val = %v, want user-123", val)
	}
}

func TestExpressionResolver_route_notFound(t *testing.T) {
	r := testResolver()
	_, err := r.Resolve("route.nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent route param")
	}
}

func TestExpressionResolver_route_nil(t *testing.T) {
	r := &ExpressionResolver{}
	_, err := r.Resolve("route.id")
	if err == nil {
		t.Fatal("expected error for nil route params")
	}
}

// --- context expressions ---

func TestExpressionResolver_context_subjectID(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("context.subject_id")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != "sub-456" {
		t.Errorf("val = %v, want sub-456", val)
	}
}

func TestExpressionResolver_context_tenantID(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("context.tenant_id")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != "tenant-acme" {
		t.Errorf("val = %v, want tenant-acme", val)
	}
}

func TestExpressionResolver_context_partitionID(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("context.partition_id")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != "partition-1" {
		t.Errorf("val = %v, want partition-1", val)
	}
}

func TestExpressionResolver_context_email(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("context.email")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != "admin@acme.com" {
		t.Errorf("val = %v, want admin@acme.com", val)
	}
}

func TestExpressionResolver_context_unknownField(t *testing.T) {
	r := testResolver()
	_, err := r.Resolve("context.unknown")
	if err == nil {
		t.Fatal("expected error for unknown context field")
	}
}

func TestExpressionResolver_context_nil(t *testing.T) {
	r := &ExpressionResolver{}
	_, err := r.Resolve("context.subject_id")
	if err == nil {
		t.Fatal("expected error for nil context")
	}
}

// --- workflow expressions ---

func TestExpressionResolver_workflow(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("workflow.order_id")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != "ord-789" {
		t.Errorf("val = %v, want ord-789", val)
	}
}

func TestExpressionResolver_workflow_notFound(t *testing.T) {
	r := testResolver()
	_, err := r.Resolve("workflow.nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent workflow field")
	}
}

func TestExpressionResolver_workflow_nil(t *testing.T) {
	r := &ExpressionResolver{}
	_, err := r.Resolve("workflow.field")
	if err == nil {
		t.Fatal("expected error for nil workflow state")
	}
}

// --- literal expressions ---

func TestExpressionResolver_literalString(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("'hello world'")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != "hello world" {
		t.Errorf("val = %v, want hello world", val)
	}
}

func TestExpressionResolver_literalStringEmpty(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("''")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != "" {
		t.Errorf("val = %q, want empty string", val)
	}
}

func TestExpressionResolver_literalInteger(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("42")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != int64(42) {
		t.Errorf("val = %v (%T), want int64(42)", val, val)
	}
}

func TestExpressionResolver_literalNegativeInteger(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("-5")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != int64(-5) {
		t.Errorf("val = %v (%T), want int64(-5)", val, val)
	}
}

func TestExpressionResolver_literalFloat(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("99.99")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != 99.99 {
		t.Errorf("val = %v (%T), want 99.99", val, val)
	}
}

func TestExpressionResolver_literalZero(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("0")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != int64(0) {
		t.Errorf("val = %v (%T), want int64(0)", val, val)
	}
}

// --- error cases ---

func TestExpressionResolver_emptyExpression(t *testing.T) {
	r := testResolver()
	_, err := r.Resolve("")
	if err == nil {
		t.Fatal("expected error for empty expression")
	}
}

func TestExpressionResolver_unknownPrefix(t *testing.T) {
	r := testResolver()
	_, err := r.Resolve("unknown.field")
	if err == nil {
		t.Fatal("expected error for unknown prefix")
	}
}

func TestExpressionResolver_noPrefix(t *testing.T) {
	r := testResolver()
	_, err := r.Resolve("noDotHere")
	if err == nil {
		t.Fatal("expected error for expression without prefix")
	}
}

func TestExpressionResolver_emptyPath(t *testing.T) {
	r := testResolver()
	_, err := r.Resolve("input.")
	if err == nil {
		t.Fatal("expected error for empty path after prefix")
	}
}

func TestExpressionResolver_whitespace(t *testing.T) {
	r := testResolver()
	val, err := r.Resolve("  input.name  ")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if val != "Alice" {
		t.Errorf("val = %v, want Alice", val)
	}
}

// --- helper function tests ---

func TestIsNumericLiteral(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"42", true},
		{"-5", true},
		{"+5", true},
		{"99.99", true},
		{"-0.5", true},
		{"0", true},
		{"", false},
		{"abc", false},
		{"12abc", false},
		{"1.2.3", false},
		{"-", false},
		{"+", false},
		{"'42'", false},
		{"input.field", false},
	}
	for _, tt := range tests {
		got := isNumericLiteral(tt.input)
		if got != tt.want {
			t.Errorf("isNumericLiteral(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestNavigatePath(t *testing.T) {
	data := map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": "deep",
			},
		},
		"top": "level",
	}

	if v := navigatePath(data, "top"); v != "level" {
		t.Errorf("navigatePath(top) = %v", v)
	}
	if v := navigatePath(data, "a.b.c"); v != "deep" {
		t.Errorf("navigatePath(a.b.c) = %v", v)
	}
	if v := navigatePath(data, "nonexistent"); v != nil {
		t.Errorf("navigatePath(nonexistent) = %v, want nil", v)
	}
	if v := navigatePath(data, "a.nonexistent"); v != nil {
		t.Errorf("navigatePath(a.nonexistent) = %v, want nil", v)
	}
}
