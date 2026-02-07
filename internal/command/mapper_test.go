package command

import (
	"testing"

	"github.com/pitabwire/thesa/model"
)

func testRctx() *model.RequestContext {
	return &model.RequestContext{
		SubjectID:   "sub-456",
		TenantID:    "tenant-acme",
		PartitionID: "partition-1",
		Email:       "admin@acme.com",
	}
}

// --- Passthrough body mapping ---

func TestInputMapper_passthrough(t *testing.T) {
	m := NewInputMapper()

	mapping := model.InputMapping{
		BodyMapping: "passthrough",
	}
	input := model.CommandInput{
		Input: map[string]any{
			"name":  "Alice",
			"email": "alice@example.com",
		},
	}

	result, err := m.MapInput(mapping, input, testRctx(), nil)
	if err != nil {
		t.Fatalf("MapInput error: %v", err)
	}

	body, ok := result.Body.(map[string]any)
	if !ok {
		t.Fatalf("body type = %T, want map[string]any", result.Body)
	}
	if body["name"] != "Alice" {
		t.Errorf("body[name] = %v, want Alice", body["name"])
	}
	if body["email"] != "alice@example.com" {
		t.Errorf("body[email] = %v", body["email"])
	}
}

func TestInputMapper_passthrough_default(t *testing.T) {
	m := NewInputMapper()

	// Empty body_mapping defaults to passthrough.
	mapping := model.InputMapping{}
	input := model.CommandInput{
		Input: map[string]any{"key": "val"},
	}

	result, err := m.MapInput(mapping, input, testRctx(), nil)
	if err != nil {
		t.Fatalf("MapInput error: %v", err)
	}
	body, ok := result.Body.(map[string]any)
	if !ok {
		t.Fatalf("body type = %T", result.Body)
	}
	if body["key"] != "val" {
		t.Errorf("body[key] = %v", body["key"])
	}
}

// --- Template body mapping ---

func TestInputMapper_template(t *testing.T) {
	m := NewInputMapper()

	mapping := model.InputMapping{
		BodyMapping: "template",
		BodyTemplate: map[string]string{
			"customerId":      "input.customer_id",
			"shippingAddress": "input.shipping_address",
			"updatedBy":       "context.subject_id",
			"source":          "'bff'",
		},
	}
	input := model.CommandInput{
		Input: map[string]any{
			"customer_id":      "cust-002",
			"shipping_address": "456 Oak Ave",
		},
	}

	result, err := m.MapInput(mapping, input, testRctx(), nil)
	if err != nil {
		t.Fatalf("MapInput error: %v", err)
	}

	body, ok := result.Body.(map[string]any)
	if !ok {
		t.Fatalf("body type = %T", result.Body)
	}
	if body["customerId"] != "cust-002" {
		t.Errorf("customerId = %v", body["customerId"])
	}
	if body["shippingAddress"] != "456 Oak Ave" {
		t.Errorf("shippingAddress = %v", body["shippingAddress"])
	}
	if body["updatedBy"] != "sub-456" {
		t.Errorf("updatedBy = %v, want sub-456", body["updatedBy"])
	}
	if body["source"] != "bff" {
		t.Errorf("source = %v, want bff", body["source"])
	}
}

func TestInputMapper_template_emptyTemplate(t *testing.T) {
	m := NewInputMapper()

	mapping := model.InputMapping{
		BodyMapping:  "template",
		BodyTemplate: map[string]string{},
	}
	input := model.CommandInput{Input: map[string]any{}}

	result, err := m.MapInput(mapping, input, testRctx(), nil)
	if err != nil {
		t.Fatalf("MapInput error: %v", err)
	}
	body, ok := result.Body.(map[string]any)
	if !ok {
		t.Fatalf("body type = %T", result.Body)
	}
	if len(body) != 0 {
		t.Errorf("len(body) = %d, want 0", len(body))
	}
}

func TestInputMapper_template_missingField(t *testing.T) {
	m := NewInputMapper()

	mapping := model.InputMapping{
		BodyMapping: "template",
		BodyTemplate: map[string]string{
			"name": "input.nonexistent",
		},
	}
	input := model.CommandInput{Input: map[string]any{}}

	_, err := m.MapInput(mapping, input, testRctx(), nil)
	if err == nil {
		t.Fatal("expected error for missing input field in template")
	}
}

// --- Projection body mapping ---

func TestInputMapper_projection(t *testing.T) {
	m := NewInputMapper()

	mapping := model.InputMapping{
		BodyMapping: "projection",
		FieldProjection: map[string]string{
			"cancellationReason": "input.reason",
			"refundType":         "input.refund_type",
		},
	}
	input := model.CommandInput{
		Input: map[string]any{
			"reason":        "Customer requested cancellation",
			"refund_type":   "full",
			"extra_field":   "should be excluded",
		},
	}

	result, err := m.MapInput(mapping, input, testRctx(), nil)
	if err != nil {
		t.Fatalf("MapInput error: %v", err)
	}

	body, ok := result.Body.(map[string]any)
	if !ok {
		t.Fatalf("body type = %T", result.Body)
	}
	if body["cancellationReason"] != "Customer requested cancellation" {
		t.Errorf("cancellationReason = %v", body["cancellationReason"])
	}
	if body["refundType"] != "full" {
		t.Errorf("refundType = %v", body["refundType"])
	}
	// extra_field should not be in the body.
	if _, exists := body["extra_field"]; exists {
		t.Error("extra_field should not be in projected body")
	}
	if len(body) != 2 {
		t.Errorf("len(body) = %d, want 2", len(body))
	}
}

func TestInputMapper_projection_withContextAndRoute(t *testing.T) {
	m := NewInputMapper()

	mapping := model.InputMapping{
		BodyMapping: "projection",
		FieldProjection: map[string]string{
			"name":     "input.name",
			"userId":   "context.subject_id",
			"orderId":  "route.id",
			"priority": "42",
		},
	}
	input := model.CommandInput{
		Input:       map[string]any{"name": "Alice"},
		RouteParams: map[string]string{"id": "ord-123"},
	}

	result, err := m.MapInput(mapping, input, testRctx(), nil)
	if err != nil {
		t.Fatalf("MapInput error: %v", err)
	}

	body, ok := result.Body.(map[string]any)
	if !ok {
		t.Fatalf("body type = %T", result.Body)
	}
	if body["name"] != "Alice" {
		t.Errorf("name = %v", body["name"])
	}
	if body["userId"] != "sub-456" {
		t.Errorf("userId = %v", body["userId"])
	}
	if body["orderId"] != "ord-123" {
		t.Errorf("orderId = %v", body["orderId"])
	}
	if body["priority"] != int64(42) {
		t.Errorf("priority = %v (%T), want int64(42)", body["priority"], body["priority"])
	}
}

// --- Path params ---

func TestInputMapper_pathParams(t *testing.T) {
	m := NewInputMapper()

	mapping := model.InputMapping{
		PathParams: map[string]string{
			"orderId":  "route.id",
			"tenantId": "context.tenant_id",
		},
		BodyMapping: "passthrough",
	}
	input := model.CommandInput{
		Input:       map[string]any{},
		RouteParams: map[string]string{"id": "ord-123"},
	}

	result, err := m.MapInput(mapping, input, testRctx(), nil)
	if err != nil {
		t.Fatalf("MapInput error: %v", err)
	}
	if result.PathParams["orderId"] != "ord-123" {
		t.Errorf("PathParams[orderId] = %q", result.PathParams["orderId"])
	}
	if result.PathParams["tenantId"] != "tenant-acme" {
		t.Errorf("PathParams[tenantId] = %q", result.PathParams["tenantId"])
	}
}

func TestInputMapper_pathParams_error(t *testing.T) {
	m := NewInputMapper()

	mapping := model.InputMapping{
		PathParams: map[string]string{
			"id": "route.nonexistent",
		},
		BodyMapping: "passthrough",
	}
	input := model.CommandInput{
		Input:       map[string]any{},
		RouteParams: map[string]string{},
	}

	_, err := m.MapInput(mapping, input, testRctx(), nil)
	if err == nil {
		t.Fatal("expected error for missing route param")
	}
}

// --- Query params ---

func TestInputMapper_queryParams(t *testing.T) {
	m := NewInputMapper()

	mapping := model.InputMapping{
		QueryParams: map[string]string{
			"expand": "'items,customer'",
			"format": "input.format",
		},
		BodyMapping: "passthrough",
	}
	input := model.CommandInput{
		Input: map[string]any{"format": "json"},
	}

	result, err := m.MapInput(mapping, input, testRctx(), nil)
	if err != nil {
		t.Fatalf("MapInput error: %v", err)
	}
	if result.QueryParams["expand"] != "items,customer" {
		t.Errorf("QueryParams[expand] = %q, want items,customer", result.QueryParams["expand"])
	}
	if result.QueryParams["format"] != "json" {
		t.Errorf("QueryParams[format] = %q", result.QueryParams["format"])
	}
}

// --- Header params ---

func TestInputMapper_headerParams(t *testing.T) {
	m := NewInputMapper()

	mapping := model.InputMapping{
		HeaderParams: map[string]string{
			"X-Tenant-Id": "context.tenant_id",
			"X-Custom":    "'static-value'",
		},
		BodyMapping: "passthrough",
	}
	input := model.CommandInput{Input: map[string]any{}}

	result, err := m.MapInput(mapping, input, testRctx(), nil)
	if err != nil {
		t.Fatalf("MapInput error: %v", err)
	}
	if result.Headers["X-Tenant-Id"] != "tenant-acme" {
		t.Errorf("Headers[X-Tenant-Id] = %q", result.Headers["X-Tenant-Id"])
	}
	if result.Headers["X-Custom"] != "static-value" {
		t.Errorf("Headers[X-Custom] = %q", result.Headers["X-Custom"])
	}
}

// --- Workflow state ---

func TestInputMapper_workflowState(t *testing.T) {
	m := NewInputMapper()

	mapping := model.InputMapping{
		PathParams: map[string]string{
			"orderId": "workflow.order_id",
		},
		BodyMapping: "template",
		BodyTemplate: map[string]string{
			"approvedBy":    "workflow.approved_by",
			"approvalNotes": "workflow.approval_notes",
		},
	}
	input := model.CommandInput{Input: map[string]any{}}

	workflowState := map[string]any{
		"order_id":       "ord-789",
		"approved_by":    "user-alice",
		"approval_notes": "Verified with warehouse",
	}

	result, err := m.MapInput(mapping, input, testRctx(), workflowState)
	if err != nil {
		t.Fatalf("MapInput error: %v", err)
	}

	if result.PathParams["orderId"] != "ord-789" {
		t.Errorf("PathParams[orderId] = %q", result.PathParams["orderId"])
	}

	body, ok := result.Body.(map[string]any)
	if !ok {
		t.Fatalf("body type = %T", result.Body)
	}
	if body["approvedBy"] != "user-alice" {
		t.Errorf("approvedBy = %v", body["approvedBy"])
	}
	if body["approvalNotes"] != "Verified with warehouse" {
		t.Errorf("approvalNotes = %v", body["approvalNotes"])
	}
}

// --- Unknown body mapping ---

func TestInputMapper_unknownStrategy(t *testing.T) {
	m := NewInputMapper()

	mapping := model.InputMapping{
		BodyMapping: "unknown_strategy",
	}
	input := model.CommandInput{Input: map[string]any{}}

	_, err := m.MapInput(mapping, input, testRctx(), nil)
	if err == nil {
		t.Fatal("expected error for unknown body mapping strategy")
	}
}

// --- Full pipeline ---

func TestInputMapper_fullPipeline(t *testing.T) {
	m := NewInputMapper()

	mapping := model.InputMapping{
		PathParams: map[string]string{
			"orderId": "route.id",
		},
		QueryParams: map[string]string{
			"expand": "'items,customer'",
		},
		HeaderParams: map[string]string{
			"X-Tenant-Id": "context.tenant_id",
		},
		BodyMapping: "projection",
		FieldProjection: map[string]string{
			"cancellationReason": "input.reason",
			"refundType":         "input.refund_type",
		},
	}

	input := model.CommandInput{
		Input: map[string]any{
			"reason":      "Customer requested cancellation",
			"refund_type": "full",
		},
		RouteParams: map[string]string{"id": "ord-123"},
	}

	result, err := m.MapInput(mapping, input, testRctx(), nil)
	if err != nil {
		t.Fatalf("MapInput error: %v", err)
	}

	// Path params.
	if result.PathParams["orderId"] != "ord-123" {
		t.Errorf("PathParams[orderId] = %q", result.PathParams["orderId"])
	}

	// Query params.
	if result.QueryParams["expand"] != "items,customer" {
		t.Errorf("QueryParams[expand] = %q", result.QueryParams["expand"])
	}

	// Headers.
	if result.Headers["X-Tenant-Id"] != "tenant-acme" {
		t.Errorf("Headers[X-Tenant-Id] = %q", result.Headers["X-Tenant-Id"])
	}

	// Body.
	body, ok := result.Body.(map[string]any)
	if !ok {
		t.Fatalf("body type = %T", result.Body)
	}
	if body["cancellationReason"] != "Customer requested cancellation" {
		t.Errorf("cancellationReason = %v", body["cancellationReason"])
	}
	if body["refundType"] != "full" {
		t.Errorf("refundType = %v", body["refundType"])
	}
}

// --- ReverseFieldMap ---

func TestReverseFieldMap(t *testing.T) {
	projection := map[string]string{
		"cancellationReason": "input.reason",
		"refundType":         "input.refund_type",
		"updatedBy":          "context.subject_id", // Not an input field.
		"source":             "'bff'",              // Literal.
	}

	reverse := ReverseFieldMap(projection)
	if reverse["cancellationReason"] != "reason" {
		t.Errorf("reverse[cancellationReason] = %q, want reason", reverse["cancellationReason"])
	}
	if reverse["refundType"] != "refund_type" {
		t.Errorf("reverse[refundType] = %q, want refund_type", reverse["refundType"])
	}
	// Non-input fields should not be in reverse map.
	if _, exists := reverse["updatedBy"]; exists {
		t.Error("updatedBy should not be in reverse map (context field)")
	}
	if _, exists := reverse["source"]; exists {
		t.Error("source should not be in reverse map (literal)")
	}
	if len(reverse) != 2 {
		t.Errorf("len(reverse) = %d, want 2", len(reverse))
	}
}

func TestReverseFieldMap_empty(t *testing.T) {
	reverse := ReverseFieldMap(nil)
	if reverse != nil {
		t.Errorf("reverse = %v, want nil", reverse)
	}
}

func TestReverseFieldMap_noInputFields(t *testing.T) {
	projection := map[string]string{
		"field": "context.subject_id",
	}
	reverse := ReverseFieldMap(projection)
	if len(reverse) != 0 {
		t.Errorf("len(reverse) = %d, want 0", len(reverse))
	}
}
