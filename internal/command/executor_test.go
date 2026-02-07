package command

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	openapiIndex "github.com/pitabwire/thesa/internal/openapi"
	"github.com/pitabwire/thesa/model"
)

// --- test helpers ---

func testCommandDefinitions() []model.DomainDefinition {
	return []model.DomainDefinition{
		{
			Domain: "orders",
			Commands: []model.CommandDefinition{
				{
					ID:           "orders.cancel",
					Capabilities: []string{"orders:cancel:execute"},
					Operation: model.OperationBinding{
						Type:        "openapi",
						ServiceID:   "orders-svc",
						OperationID: "cancelOrder",
					},
					Input: model.InputMapping{
						PathParams: map[string]string{
							"orderId": "route.id",
						},
						BodyMapping: "projection",
						FieldProjection: map[string]string{
							"cancellationReason": "input.reason",
							"refundType":         "input.refund_type",
						},
					},
					Output: model.OutputMapping{
						SuccessMessage: "Order cancelled successfully",
						Fields: map[string]string{
							"id":     "id",
							"status": "status",
						},
						ErrorMap: map[string]string{
							"ORDER_SHIPPED": "Cannot cancel an order that has already shipped.",
						},
					},
					Idempotency: &model.IdempotencyConfig{
						KeySource: "header",
						TTL:       "1h",
					},
					RateLimit: &model.RateLimitConfig{
						MaxRequests: 10,
						Window:      "1m",
						Scope:       "user",
					},
				},
				{
					ID: "orders.create",
					Operation: model.OperationBinding{
						Type:        "openapi",
						ServiceID:   "orders-svc",
						OperationID: "createOrder",
					},
					Input: model.InputMapping{
						BodyMapping: "passthrough",
					},
					Output: model.OutputMapping{
						SuccessMessage: "Order created",
					},
				},
				{
					ID: "orders.simple",
					Operation: model.OperationBinding{
						Type:    "sdk",
						Handler: "simpleHandler",
					},
					Input: model.InputMapping{
						BodyMapping: "passthrough",
					},
					Output: model.OutputMapping{
						SuccessMessage: "Done",
					},
				},
			},
		},
	}
}

// mockOperationInvoker is a mock invoker for testing.
type mockOperationInvoker struct {
	invokeFn func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error)
}

func (m *mockOperationInvoker) Supports(binding model.OperationBinding) bool { return true }

func (m *mockOperationInvoker) Invoke(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
	if m.invokeFn != nil {
		return m.invokeFn(ctx, rctx, binding, input)
	}
	return model.InvocationResult{StatusCode: 200, Body: map[string]any{}}, nil
}

// mockRateLimiter is a mock rate limiter for testing.
type mockRateLimiter struct {
	allow bool
}

func (m *mockRateLimiter) Allow(ctx context.Context, commandID string, scope string, rctx *model.RequestContext) bool {
	return m.allow
}

// mockObserver records command events.
type mockObserver struct {
	mu     sync.Mutex
	events []CommandEvent
}

func (m *mockObserver) OnCommandExecuted(ctx context.Context, event CommandEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *mockObserver) lastEvent() CommandEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.events) == 0 {
		return CommandEvent{}
	}
	return m.events[len(m.events)-1]
}

func testRctxForExecutor() *model.RequestContext {
	return &model.RequestContext{
		SubjectID:   "user-alice",
		TenantID:    "acme-corp",
		PartitionID: "p1",
		Email:       "alice@acme.com",
	}
}

func newTestExecutor(invokeFn func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error), opts ...CommandExecutorOption) *CommandExecutor {
	reg := definition.NewRegistry(testCommandDefinitions())
	invReg := invoker.NewRegistry()
	invReg.Register(&mockOperationInvoker{invokeFn: invokeFn})

	return NewCommandExecutor(reg, invReg, nil, opts...)
}

func newTestExecutorWithIndex(invokeFn func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error), opts ...CommandExecutorOption) *CommandExecutor {
	reg := definition.NewRegistry(testCommandDefinitions())
	invReg := invoker.NewRegistry()
	invReg.Register(&mockOperationInvoker{invokeFn: invokeFn})

	idx := loadTestOAIndex()

	return NewCommandExecutor(reg, invReg, idx, opts...)
}

// loadTestOAIndex creates a minimal OpenAPI index for testing schema validation.
func loadTestOAIndex() *openapiIndex.Index {
	spec := `openapi: "3.0.0"
info:
  title: Orders Service
  version: "1.0"
paths:
  /orders:
    post:
      operationId: createOrder
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [customer_id, items]
              properties:
                customer_id:
                  type: string
                items:
                  type: array
                  items:
                    type: object
      responses:
        "200":
          description: OK
  /orders/{orderId}/cancel:
    post:
      operationId: cancelOrder
      parameters:
        - name: orderId
          in: path
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [cancellationReason]
              properties:
                cancellationReason:
                  type: string
                refundType:
                  type: string
      responses:
        "200":
          description: OK
`

	dir := os.TempDir()
	specFile := filepath.Join(dir, "test-orders-spec.yaml")
	os.WriteFile(specFile, []byte(spec), 0644)

	idx := openapiIndex.NewIndex()
	if err := idx.Load([]openapiIndex.SpecSource{
		{ServiceID: "orders-svc", BaseURL: "https://orders.internal", SpecPath: specFile},
	}); err != nil {
		panic(fmt.Sprintf("failed to load test spec: %v", err))
	}
	return idx
}

// --- Step 1: Lookup ---

func TestExecutor_notFound(t *testing.T) {
	e := newTestExecutor(nil)

	_, err := e.Execute(context.Background(), testRctxForExecutor(), model.CapabilitySet{}, "nonexistent", model.CommandInput{})
	if err == nil {
		t.Fatal("expected not found error")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrNotFound {
		t.Errorf("code = %s, want %s", envErr.Code, model.ErrNotFound)
	}
}

// --- Step 2: Capabilities ---

func TestExecutor_forbidden(t *testing.T) {
	e := newTestExecutor(nil)

	_, err := e.Execute(context.Background(), testRctxForExecutor(), model.CapabilitySet{}, "orders.cancel", model.CommandInput{})
	if err == nil {
		t.Fatal("expected forbidden error")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrForbidden {
		t.Errorf("code = %s, want %s", envErr.Code, model.ErrForbidden)
	}
}

func TestExecutor_noCapabilitiesRequired(t *testing.T) {
	e := newTestExecutor(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		return model.InvocationResult{StatusCode: 200, Body: map[string]any{}}, nil
	})

	// orders.create has no capability requirements.
	resp, err := e.Execute(context.Background(), testRctxForExecutor(), model.CapabilitySet{}, "orders.create", model.CommandInput{Input: map[string]any{}})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !resp.Success {
		t.Error("Success = false")
	}
}

// --- Step 3: Idempotency ---

func TestExecutor_idempotency_returnsCached(t *testing.T) {
	store := NewMemoryIdempotencyStore()

	// Pre-populate cached result.
	input := model.CommandInput{
		Input:          map[string]any{"reason": "test", "refund_type": "full"},
		RouteParams:    map[string]string{"id": "ord-123"},
		IdempotencyKey: "idem-key-1",
	}
	hash := hashInput(input)
	cachedResp := model.CommandResponse{Success: true, Message: "Cached result"}
	_ = store.Store(context.Background(), FormatIdempotencyKey("orders.cancel", "idem-key-1"), hash, cachedResp, 1*time.Hour)

	// Backend should NOT be called.
	invoked := false
	e := newTestExecutor(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		invoked = true
		return model.InvocationResult{}, nil
	}, WithIdempotencyStore(store))

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	resp, err := e.Execute(context.Background(), testRctxForExecutor(), caps, "orders.cancel", input)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if invoked {
		t.Error("backend was invoked, should return cached result")
	}
	if resp.Message != "Cached result" {
		t.Errorf("Message = %q, want Cached result", resp.Message)
	}
}

func TestExecutor_idempotency_conflict(t *testing.T) {
	store := NewMemoryIdempotencyStore()

	// Store a result with one hash.
	_ = store.Store(context.Background(), FormatIdempotencyKey("orders.cancel", "idem-key-1"), "different-hash", model.CommandResponse{Success: true}, 1*time.Hour)

	e := newTestExecutor(nil, WithIdempotencyStore(store))

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	input := model.CommandInput{
		Input:          map[string]any{"reason": "test", "refund_type": "full"},
		RouteParams:    map[string]string{"id": "ord-123"},
		IdempotencyKey: "idem-key-1",
	}

	_, err := e.Execute(context.Background(), testRctxForExecutor(), caps, "orders.cancel", input)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrConflict {
		t.Errorf("code = %s, want %s", envErr.Code, model.ErrConflict)
	}
}

func TestExecutor_idempotency_storesOnSuccess(t *testing.T) {
	store := NewMemoryIdempotencyStore()

	e := newTestExecutor(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		return model.InvocationResult{
			StatusCode: 200,
			Body: map[string]any{
				"id":     "ord-123",
				"status": "cancelled",
			},
		}, nil
	}, WithIdempotencyStore(store))

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	input := model.CommandInput{
		Input:          map[string]any{"reason": "test", "refund_type": "full"},
		RouteParams:    map[string]string{"id": "ord-123"},
		IdempotencyKey: "idem-key-2",
	}

	resp, err := e.Execute(context.Background(), testRctxForExecutor(), caps, "orders.cancel", input)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !resp.Success {
		t.Error("Success = false")
	}

	// Verify result was stored.
	if store.Len() != 1 {
		t.Errorf("store.Len() = %d, want 1", store.Len())
	}
}

// --- Step 4: Rate limiting ---

func TestExecutor_rateLimited(t *testing.T) {
	e := newTestExecutor(nil, WithRateLimiter(&mockRateLimiter{allow: false}))

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	input := model.CommandInput{
		Input:       map[string]any{"reason": "test", "refund_type": "full"},
		RouteParams: map[string]string{"id": "ord-123"},
	}

	_, err := e.Execute(context.Background(), testRctxForExecutor(), caps, "orders.cancel", input)
	if err == nil {
		t.Fatal("expected rate limit error")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrRateLimited {
		t.Errorf("code = %s, want %s", envErr.Code, model.ErrRateLimited)
	}
}

func TestExecutor_rateLimitAllowed(t *testing.T) {
	e := newTestExecutor(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		return model.InvocationResult{StatusCode: 200, Body: map[string]any{}}, nil
	}, WithRateLimiter(&mockRateLimiter{allow: true}))

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	input := model.CommandInput{
		Input:       map[string]any{"reason": "test", "refund_type": "full"},
		RouteParams: map[string]string{"id": "ord-123"},
	}

	resp, err := e.Execute(context.Background(), testRctxForExecutor(), caps, "orders.cancel", input)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !resp.Success {
		t.Error("Success = false")
	}
}

// --- Step 5: Input mapping ---

func TestExecutor_inputMappingApplied(t *testing.T) {
	var captured model.InvocationInput
	e := newTestExecutor(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		captured = input
		return model.InvocationResult{StatusCode: 200, Body: map[string]any{}}, nil
	})

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	input := model.CommandInput{
		Input: map[string]any{
			"reason":      "Customer requested",
			"refund_type": "full",
		},
		RouteParams: map[string]string{"id": "ord-123"},
	}

	_, err := e.Execute(context.Background(), testRctxForExecutor(), caps, "orders.cancel", input)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	// Path params should be resolved.
	if captured.PathParams["orderId"] != "ord-123" {
		t.Errorf("PathParams[orderId] = %q, want ord-123", captured.PathParams["orderId"])
	}

	// Body should be projected.
	body, ok := captured.Body.(map[string]any)
	if !ok {
		t.Fatalf("body type = %T", captured.Body)
	}
	if body["cancellationReason"] != "Customer requested" {
		t.Errorf("cancellationReason = %v", body["cancellationReason"])
	}
	if body["refundType"] != "full" {
		t.Errorf("refundType = %v", body["refundType"])
	}
}

func TestExecutor_inputMappingError(t *testing.T) {
	e := newTestExecutor(nil)

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	// Missing route.id needed for path param.
	input := model.CommandInput{
		Input:       map[string]any{"reason": "test", "refund_type": "full"},
		RouteParams: map[string]string{}, // Missing "id".
	}

	_, err := e.Execute(context.Background(), testRctxForExecutor(), caps, "orders.cancel", input)
	if err == nil {
		t.Fatal("expected error for missing route param")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrBadRequest {
		t.Errorf("code = %s, want %s", envErr.Code, model.ErrBadRequest)
	}
}

// --- Step 6: Schema validation ---

func TestExecutor_schemaValidation_valid(t *testing.T) {
	e := newTestExecutorWithIndex(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		return model.InvocationResult{StatusCode: 200, Body: map[string]any{"id": "ord-123", "status": "cancelled"}}, nil
	})

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	input := model.CommandInput{
		Input:       map[string]any{"reason": "Customer requested", "refund_type": "full"},
		RouteParams: map[string]string{"id": "ord-123"},
	}

	resp, err := e.Execute(context.Background(), testRctxForExecutor(), caps, "orders.cancel", input)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !resp.Success {
		t.Errorf("Success = false, errors: %+v", resp.Errors)
	}
}

func TestExecutor_schemaValidation_missingRequired(t *testing.T) {
	e := newTestExecutorWithIndex(nil)

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	// Input missing "reason" → body will miss "cancellationReason" which is required.
	input := model.CommandInput{
		Input:       map[string]any{"refund_type": "full"},
		RouteParams: map[string]string{"id": "ord-123"},
	}

	_, err := e.Execute(context.Background(), testRctxForExecutor(), caps, "orders.cancel", input)
	if err == nil {
		t.Fatal("expected validation error")
	}

	// First, input mapping will fail because "reason" field is missing from input.
	// Let's check the error type.
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	// This will be BAD_REQUEST because input.reason is not found.
	if envErr.Code != model.ErrBadRequest {
		t.Errorf("code = %s", envErr.Code)
	}
}

func TestExecutor_schemaValidation_createOrderMissingFields(t *testing.T) {
	e := newTestExecutorWithIndex(nil)

	// orders.create uses passthrough mapping and requires customer_id and items.
	input := model.CommandInput{
		Input: map[string]any{
			"notes": "some notes", // Missing customer_id and items.
		},
	}

	_, err := e.Execute(context.Background(), testRctxForExecutor(), model.CapabilitySet{}, "orders.create", input)
	if err == nil {
		t.Fatal("expected validation error")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrValidationError {
		t.Errorf("code = %s, want %s", envErr.Code, model.ErrValidationError)
	}
	if len(envErr.Details) < 2 {
		t.Errorf("expected at least 2 validation errors, got %d", len(envErr.Details))
	}
}

// --- Step 7: Backend invocation ---

func TestExecutor_backendError(t *testing.T) {
	e := newTestExecutor(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		return model.InvocationResult{}, fmt.Errorf("connection refused")
	})

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	input := model.CommandInput{
		Input:       map[string]any{"reason": "test", "refund_type": "full"},
		RouteParams: map[string]string{"id": "ord-123"},
	}

	_, err := e.Execute(context.Background(), testRctxForExecutor(), caps, "orders.cancel", input)
	if err == nil {
		t.Fatal("expected backend error")
	}
}

// --- Step 8: Response handling ---

func TestExecutor_successResponse(t *testing.T) {
	e := newTestExecutor(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		return model.InvocationResult{
			StatusCode: 200,
			Body: map[string]any{
				"id":          "ord-123",
				"status":      "cancelled",
				"cancelledAt": "2025-01-15T10:30:00Z",
				"extra_field": "should be filtered by output mapping",
			},
		}, nil
	})

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	input := model.CommandInput{
		Input:       map[string]any{"reason": "test", "refund_type": "full"},
		RouteParams: map[string]string{"id": "ord-123"},
	}

	resp, err := e.Execute(context.Background(), testRctxForExecutor(), caps, "orders.cancel", input)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !resp.Success {
		t.Error("Success = false")
	}
	if resp.Message != "Order cancelled successfully" {
		t.Errorf("Message = %q", resp.Message)
	}
	// Output mapping should select id and status only.
	if resp.Result["id"] != "ord-123" {
		t.Errorf("Result[id] = %v", resp.Result["id"])
	}
	if resp.Result["status"] != "cancelled" {
		t.Errorf("Result[status] = %v", resp.Result["status"])
	}
	// extra_field should not be in result (only mapped fields).
	if _, exists := resp.Result["extra_field"]; exists {
		t.Error("extra_field should not be in result")
	}
}

func TestExecutor_clientError_withErrorMap(t *testing.T) {
	e := newTestExecutor(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		return model.InvocationResult{
			StatusCode: 422,
			Body: map[string]any{
				"error": map[string]any{
					"code":    "ORDER_SHIPPED",
					"message": "Cannot cancel order in 'shipped' state",
				},
			},
		}, nil
	})

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	input := model.CommandInput{
		Input:       map[string]any{"reason": "test", "refund_type": "full"},
		RouteParams: map[string]string{"id": "ord-123"},
	}

	resp, err := e.Execute(context.Background(), testRctxForExecutor(), caps, "orders.cancel", input)
	if err == nil {
		t.Fatal("expected error for 4xx response")
	}
	if resp.Success {
		t.Error("Success = true, want false")
	}
	// Error map should translate ORDER_SHIPPED.
	if resp.Message != "Cannot cancel an order that has already shipped." {
		t.Errorf("Message = %q", resp.Message)
	}
}

func TestExecutor_clientError_withFieldErrors(t *testing.T) {
	e := newTestExecutor(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		return model.InvocationResult{
			StatusCode: 422,
			Body: map[string]any{
				"error": map[string]any{
					"code":    "VALIDATION_ERROR",
					"message": "Validation failed",
					"details": []any{
						map[string]any{"field": "cancellationReason", "code": "REQUIRED", "message": "Reason is required"},
					},
				},
			},
		}, nil
	})

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	input := model.CommandInput{
		Input:       map[string]any{"reason": "test", "refund_type": "full"},
		RouteParams: map[string]string{"id": "ord-123"},
	}

	resp, err := e.Execute(context.Background(), testRctxForExecutor(), caps, "orders.cancel", input)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(resp.Errors) != 1 {
		t.Fatalf("len(Errors) = %d, want 1", len(resp.Errors))
	}
	// Field name should be reversed: cancellationReason → reason.
	if resp.Errors[0].Field != "reason" {
		t.Errorf("Errors[0].Field = %q, want reason", resp.Errors[0].Field)
	}
}

func TestExecutor_serverError(t *testing.T) {
	e := newTestExecutor(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		return model.InvocationResult{
			StatusCode: 500,
			Body:       map[string]any{"error": "internal server error"},
		}, nil
	})

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	input := model.CommandInput{
		Input:       map[string]any{"reason": "test", "refund_type": "full"},
		RouteParams: map[string]string{"id": "ord-123"},
	}

	resp, err := e.Execute(context.Background(), testRctxForExecutor(), caps, "orders.cancel", input)
	if err == nil {
		t.Fatal("expected error for 5xx response")
	}
	if resp.Success {
		t.Error("Success = true, want false")
	}
	if resp.Message != "An internal error occurred. Please try again later." {
		t.Errorf("Message = %q", resp.Message)
	}
}

// --- Steps 9-10: Observers ---

func TestExecutor_observerNotified(t *testing.T) {
	obs := &mockObserver{}

	e := newTestExecutor(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		return model.InvocationResult{StatusCode: 200, Body: map[string]any{}}, nil
	}, WithObserver(obs))

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	input := model.CommandInput{
		Input:       map[string]any{"reason": "test", "refund_type": "full"},
		RouteParams: map[string]string{"id": "ord-123"},
	}

	_, err := e.Execute(context.Background(), testRctxForExecutor(), caps, "orders.cancel", input)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	event := obs.lastEvent()
	if event.CommandID != "orders.cancel" {
		t.Errorf("CommandID = %q", event.CommandID)
	}
	if event.SubjectID != "user-alice" {
		t.Errorf("SubjectID = %q", event.SubjectID)
	}
	if event.TenantID != "acme-corp" {
		t.Errorf("TenantID = %q", event.TenantID)
	}
	if !event.Success {
		t.Error("Success = false")
	}
	if event.StatusCode != 200 {
		t.Errorf("StatusCode = %d", event.StatusCode)
	}
	if event.Duration <= 0 {
		t.Error("Duration <= 0")
	}
}

func TestExecutor_observerOnError(t *testing.T) {
	obs := &mockObserver{}

	e := newTestExecutor(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		return model.InvocationResult{}, fmt.Errorf("backend down")
	}, WithObserver(obs))

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	input := model.CommandInput{
		Input:       map[string]any{"reason": "test", "refund_type": "full"},
		RouteParams: map[string]string{"id": "ord-123"},
	}

	_, _ = e.Execute(context.Background(), testRctxForExecutor(), caps, "orders.cancel", input)

	event := obs.lastEvent()
	if event.Success {
		t.Error("Success = true, want false")
	}
	if event.Error == "" {
		t.Error("Error is empty, want error message")
	}
}

// --- Validate (dry-run) ---

func TestExecutor_Validate_valid(t *testing.T) {
	e := newTestExecutorWithIndex(nil)

	caps := model.CapabilitySet{"orders:cancel:execute": true}
	input := model.CommandInput{
		Input:       map[string]any{"reason": "test reason", "refund_type": "full"},
		RouteParams: map[string]string{"id": "ord-123"},
	}

	errors := e.Validate(testRctxForExecutor(), caps, "orders.cancel", input)
	if len(errors) != 0 {
		t.Errorf("Validate returned %d errors, want 0: %+v", len(errors), errors)
	}
}

func TestExecutor_Validate_missingFields(t *testing.T) {
	e := newTestExecutorWithIndex(nil)

	input := model.CommandInput{
		Input: map[string]any{"notes": "just notes"}, // Missing customer_id and items.
	}

	errors := e.Validate(testRctxForExecutor(), model.CapabilitySet{}, "orders.create", input)
	if len(errors) < 2 {
		t.Errorf("Validate returned %d errors, want at least 2: %+v", len(errors), errors)
	}
}

func TestExecutor_Validate_notFound(t *testing.T) {
	e := newTestExecutor(nil)

	errors := e.Validate(testRctxForExecutor(), model.CapabilitySet{}, "nonexistent", model.CommandInput{})
	if len(errors) != 1 {
		t.Fatalf("len(errors) = %d, want 1", len(errors))
	}
	if errors[0].Code != "NOT_FOUND" {
		t.Errorf("errors[0].Code = %q, want NOT_FOUND", errors[0].Code)
	}
}

func TestExecutor_Validate_forbidden(t *testing.T) {
	e := newTestExecutor(nil)

	errors := e.Validate(testRctxForExecutor(), model.CapabilitySet{}, "orders.cancel", model.CommandInput{})
	if len(errors) != 1 {
		t.Fatalf("len(errors) = %d, want 1", len(errors))
	}
	if errors[0].Code != "FORBIDDEN" {
		t.Errorf("errors[0].Code = %q, want FORBIDDEN", errors[0].Code)
	}
}

// --- Helper function tests ---

func TestHashInput(t *testing.T) {
	input1 := model.CommandInput{Input: map[string]any{"name": "Alice"}}
	input2 := model.CommandInput{Input: map[string]any{"name": "Bob"}}
	input3 := model.CommandInput{Input: map[string]any{"name": "Alice"}}

	h1 := hashInput(input1)
	h2 := hashInput(input2)
	h3 := hashInput(input3)

	if h1 == h2 {
		t.Error("different inputs should have different hashes")
	}
	if h1 != h3 {
		t.Error("same inputs should have same hash")
	}
	if len(h1) != 64 { // SHA-256 hex.
		t.Errorf("hash length = %d, want 64", len(h1))
	}
}

func TestParseIdempotencyTTL(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"1h", 1 * time.Hour},
		{"30m", 30 * time.Minute},
		{"24h", 24 * time.Hour},
		{"invalid", 24 * time.Hour}, // Default.
		{"", 24 * time.Hour},        // Default.
	}
	for _, tt := range tests {
		got := parseIdempotencyTTL(tt.input)
		if got != tt.want {
			t.Errorf("parseIdempotencyTTL(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestApplyOutputMapping(t *testing.T) {
	body := map[string]any{
		"id":          "ord-123",
		"status":      "cancelled",
		"cancelledAt": "2025-01-15T10:30:00Z",
	}
	output := model.OutputMapping{
		Fields: map[string]string{
			"order_id":    "id",
			"order_status": "status",
		},
	}

	result := applyOutputMapping(body, output)
	if result["order_id"] != "ord-123" {
		t.Errorf("order_id = %v", result["order_id"])
	}
	if result["order_status"] != "cancelled" {
		t.Errorf("order_status = %v", result["order_status"])
	}
	// cancelledAt not mapped → should not appear.
	if _, exists := result["cancelledAt"]; exists {
		t.Error("cancelledAt should not be in mapped result")
	}
}

func TestApplyOutputMapping_noFields(t *testing.T) {
	body := map[string]any{"id": "123", "name": "test"}
	output := model.OutputMapping{} // No field mapping.

	result := applyOutputMapping(body, output)
	if result["id"] != "123" {
		t.Errorf("id = %v, want 123 (passthrough)", result["id"])
	}
}

func TestExtractString(t *testing.T) {
	data := map[string]any{
		"error": map[string]any{
			"code":    "ERR_CODE",
			"message": "Something failed",
		},
		"status": float64(422),
	}

	if extractString(data, "error.code") != "ERR_CODE" {
		t.Errorf("error.code = %q", extractString(data, "error.code"))
	}
	if extractString(data, "error.message") != "Something failed" {
		t.Errorf("error.message = %q", extractString(data, "error.message"))
	}
	if extractString(data, "status") != "422" {
		t.Errorf("status = %q, want 422 (formatted)", extractString(data, "status"))
	}
	if extractString(data, "nonexistent") != "" {
		t.Error("nonexistent should return empty string")
	}
}

func TestExtractFieldErrors(t *testing.T) {
	body := map[string]any{
		"error": map[string]any{
			"details": []any{
				map[string]any{"field": "name", "code": "REQUIRED", "message": "Name is required"},
				map[string]any{"field": "email", "code": "INVALID", "message": "Invalid email"},
			},
		},
	}

	errors := extractFieldErrors(body)
	if len(errors) != 2 {
		t.Fatalf("len(errors) = %d, want 2", len(errors))
	}
	if errors[0].Field != "name" || errors[0].Code != "REQUIRED" {
		t.Errorf("errors[0] = %+v", errors[0])
	}
}

func TestExtractFieldErrors_noDetails(t *testing.T) {
	body := map[string]any{
		"error": map[string]any{
			"code": "ERR",
		},
	}

	errors := extractFieldErrors(body)
	if len(errors) != 0 {
		t.Errorf("len(errors) = %d, want 0", len(errors))
	}
}

func TestContainsWord(t *testing.T) {
	tests := []struct {
		s, word string
		want    bool
	}{
		{"customer_id is required", "required", true},
		{"REQUIRED field", "required", true},
		{"not here", "required", false},
		{"", "required", false},
	}
	for _, tt := range tests {
		got := containsWord(tt.s, tt.word)
		if got != tt.want {
			t.Errorf("containsWord(%q, %q) = %v, want %v", tt.s, tt.word, got, tt.want)
		}
	}
}

func TestRateLimitScopeKey(t *testing.T) {
	rctx := &model.RequestContext{SubjectID: "user-1", TenantID: "tenant-1"}

	if key := RateLimitScopeKey("cmd", "user", rctx); key != "rl:cmd:user:user-1" {
		t.Errorf("user scope = %q", key)
	}
	if key := RateLimitScopeKey("cmd", "tenant", rctx); key != "rl:cmd:tenant:tenant-1" {
		t.Errorf("tenant scope = %q", key)
	}
	if key := RateLimitScopeKey("cmd", "global", rctx); key != "rl:cmd:global" {
		t.Errorf("global scope = %q", key)
	}
}
