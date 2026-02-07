package workflow

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/model"
)

// --- Test helpers ---

func testRctx() *model.RequestContext {
	return &model.RequestContext{
		SubjectID:   "user-alice",
		TenantID:    "tenant-1",
		PartitionID: "partition-1",
		Email:       "alice@example.com",
	}
}

// mockCapResolver always returns the given capabilities.
type mockCapResolver struct {
	caps model.CapabilitySet
}

func (m *mockCapResolver) Resolve(_ *model.RequestContext) (model.CapabilitySet, error) {
	return m.caps, nil
}
func (m *mockCapResolver) Invalidate(_, _ string) {}

// mockInvoker records invocations and returns a configurable result.
type mockInvoker struct {
	calls      []mockInvokerCall
	resultFunc func(binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error)
}

type mockInvokerCall struct {
	Binding model.OperationBinding
	Input   model.InvocationInput
}

func (m *mockInvoker) Invoke(_ context.Context, _ *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
	m.calls = append(m.calls, mockInvokerCall{Binding: binding, Input: input})
	if m.resultFunc != nil {
		return m.resultFunc(binding, input)
	}
	return model.InvocationResult{StatusCode: 200, Body: map[string]any{"result_key": "result_val"}}, nil
}

func (m *mockInvoker) Supports(_ model.OperationBinding) bool { return true }

// testWorkflowDefinitions returns test definitions with various workflow shapes.
func testWorkflowDefinitions() []model.DomainDefinition {
	return []model.DomainDefinition{
		{
			Domain: "orders",
			Workflows: []model.WorkflowDefinition{
				{
					ID:           "orders.approval",
					Name:         "Order Approval",
					Capabilities: []string{"orders:approve:execute"},
					InitialStep:  "review",
					Timeout:      "72h",
					OnTimeout:    "expired",
					Steps: []model.StepDefinition{
						{ID: "review", Name: "Review", Type: "action", Capabilities: []string{"orders:approve:review"}},
						{
							ID: "process", Name: "Process Order", Type: "system",
							Operation: &model.OperationBinding{Type: "openapi", OperationID: "processOrder"},
							Input:     &model.InputMapping{BodyMapping: "passthrough"},
							Output:    &model.OutputMapping{Fields: map[string]string{"confirmed_at": "confirmed_at"}},
						},
						{
							ID: "notify", Name: "Send Notification", Type: "notification",
							Operation: &model.OperationBinding{Type: "openapi", OperationID: "sendNotification"},
						},
						{ID: "approved", Name: "Approved", Type: "terminal"},
						{ID: "expired", Name: "Expired", Type: "terminal"},
						{ID: "rejected", Name: "Rejected", Type: "terminal"},
					},
					Transitions: []model.TransitionDefinition{
						{From: "review", To: "process", Event: "approved"},
						{From: "review", To: "rejected", Event: "rejected"},
						{From: "process", To: "notify", Event: "completed"},
						{From: "notify", To: "approved", Event: "completed"},
					},
				},
				// Workflow with system initial step.
				{
					ID:          "orders.auto-process",
					Name:        "Auto Process",
					InitialStep: "compute",
					Steps: []model.StepDefinition{
						{
							ID: "compute", Name: "Compute", Type: "system",
							Operation: &model.OperationBinding{Type: "sdk", Handler: "compute"},
							Output:    &model.OutputMapping{Fields: map[string]string{"total": "total"}},
						},
						{ID: "review", Name: "Review", Type: "action"},
						{ID: "done", Name: "Done", Type: "terminal"},
					},
					Transitions: []model.TransitionDefinition{
						{From: "compute", To: "review", Event: "completed"},
						{From: "review", To: "done", Event: "approved"},
					},
				},
				// Workflow with conditional transitions.
				{
					ID:          "orders.conditional",
					Name:        "Conditional Flow",
					InitialStep: "evaluate",
					Steps: []model.StepDefinition{
						{ID: "evaluate", Name: "Evaluate", Type: "action"},
						{ID: "fast-track", Name: "Fast Track", Type: "terminal"},
						{ID: "standard", Name: "Standard", Type: "terminal"},
					},
					Transitions: []model.TransitionDefinition{
						{From: "evaluate", To: "fast-track", Event: "submitted", Condition: "priority == 'high'"},
						{From: "evaluate", To: "standard", Event: "submitted", Condition: "priority != 'high'"},
					},
				},
				// Workflow with error transition on system step.
				{
					ID:          "orders.with-error-handling",
					Name:        "With Error Handling",
					InitialStep: "fetch",
					Steps: []model.StepDefinition{
						{
							ID: "fetch", Name: "Fetch Data", Type: "system",
							Operation: &model.OperationBinding{Type: "openapi", OperationID: "fetchData"},
						},
						{ID: "done", Name: "Done", Type: "terminal"},
						{ID: "error-step", Name: "Error", Type: "terminal"},
					},
					Transitions: []model.TransitionDefinition{
						{From: "fetch", To: "done", Event: "completed"},
						{From: "fetch", To: "error-step", Event: "error"},
					},
				},
				// Workflow with guarded transition.
				{
					ID:          "orders.guarded",
					Name:        "Guarded Flow",
					InitialStep: "start",
					Steps: []model.StepDefinition{
						{ID: "start", Name: "Start", Type: "action"},
						{ID: "admin-only", Name: "Admin Only", Type: "terminal"},
					},
					Transitions: []model.TransitionDefinition{
						{From: "start", To: "admin-only", Event: "proceed", Guard: "admin:access"},
					},
				},
				// Workflow that chains system steps to test chain limit.
				{
					ID:          "orders.chain-loop",
					Name:        "Chain Loop",
					InitialStep: "step-a",
					Steps: []model.StepDefinition{
						{
							ID: "step-a", Name: "Step A", Type: "system",
							Operation: &model.OperationBinding{Type: "sdk", Handler: "a"},
						},
						{
							ID: "step-b", Name: "Step B", Type: "system",
							Operation: &model.OperationBinding{Type: "sdk", Handler: "b"},
						},
					},
					Transitions: []model.TransitionDefinition{
						{From: "step-a", To: "step-b", Event: "completed"},
						{From: "step-b", To: "step-a", Event: "completed"},
					},
				},
			},
		},
	}
}

func newTestEngine(inv *mockInvoker, caps model.CapabilitySet) (*Engine, *MemoryWorkflowStore) {
	store := NewMemoryWorkflowStore()
	reg := definition.NewRegistry(testWorkflowDefinitions())
	invReg := invoker.NewRegistry()
	if inv != nil {
		invReg.Register(inv)
	}
	capRes := &mockCapResolver{caps: caps}
	engine := NewEngine(reg, store, invReg, capRes)
	return engine, store
}

// --- Start tests ---

func TestEngine_Start_success(t *testing.T) {
	inv := &mockInvoker{}
	engine, store := newTestEngine(inv, model.CapabilitySet{"orders:approve:execute": true})
	ctx := context.Background()
	rctx := testRctx()

	inst, err := engine.Start(ctx, rctx, "orders.approval", map[string]any{"order_id": "ord-123"})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if inst.ID == "" {
		t.Error("expected non-empty instance ID")
	}
	if inst.WorkflowID != "orders.approval" {
		t.Errorf("WorkflowID = %q", inst.WorkflowID)
	}
	if inst.CurrentStep != "review" {
		t.Errorf("CurrentStep = %q, want review", inst.CurrentStep)
	}
	if inst.Status != model.WorkflowStatusActive {
		t.Errorf("Status = %q, want active", inst.Status)
	}
	if inst.TenantID != "tenant-1" {
		t.Errorf("TenantID = %q", inst.TenantID)
	}
	if inst.State["order_id"] != "ord-123" {
		t.Errorf("State[order_id] = %v", inst.State["order_id"])
	}
	if inst.ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set (72h timeout)")
	}
	if store.Len() != 1 {
		t.Errorf("store.Len() = %d", store.Len())
	}

	// Verify step_entered event was recorded.
	events, _ := store.GetEvents(ctx, "tenant-1", inst.ID)
	if len(events) != 1 {
		t.Fatalf("events count = %d, want 1", len(events))
	}
	if events[0].Event != "step_entered" {
		t.Errorf("events[0].Event = %q", events[0].Event)
	}
	if events[0].StepID != "review" {
		t.Errorf("events[0].StepID = %q", events[0].StepID)
	}
}

func TestEngine_Start_notFound(t *testing.T) {
	engine, _ := newTestEngine(nil, model.CapabilitySet{"*": true})
	ctx := context.Background()
	rctx := testRctx()

	_, err := engine.Start(ctx, rctx, "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrNotFound {
		t.Errorf("code = %s", envErr.Code)
	}
}

func TestEngine_Start_forbidden(t *testing.T) {
	engine, _ := newTestEngine(nil, model.CapabilitySet{}) // No capabilities.
	ctx := context.Background()
	rctx := testRctx()

	_, err := engine.Start(ctx, rctx, "orders.approval", nil)
	if err == nil {
		t.Fatal("expected forbidden error")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrForbidden {
		t.Errorf("code = %s", envErr.Code)
	}
}

func TestEngine_Start_systemInitialStep(t *testing.T) {
	inv := &mockInvoker{
		resultFunc: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			return model.InvocationResult{
				StatusCode: 200,
				Body:       map[string]any{"total": 42.0},
			}, nil
		},
	}
	engine, store := newTestEngine(inv, model.CapabilitySet{"*": true})
	ctx := context.Background()
	rctx := testRctx()

	inst, err := engine.Start(ctx, rctx, "orders.auto-process", map[string]any{"items": 3})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	// Should auto-execute "compute" and land on "review" (human step).
	if inst.CurrentStep != "review" {
		t.Errorf("CurrentStep = %q, want review (after auto-execute)", inst.CurrentStep)
	}
	if inst.Status != model.WorkflowStatusActive {
		t.Errorf("Status = %q, want active", inst.Status)
	}

	// Verify total was merged into state from output mapping.
	got, _ := store.Get(ctx, "tenant-1", inst.ID)
	if got.State["total"] != 42.0 {
		t.Errorf("State[total] = %v, want 42", got.State["total"])
	}

	// Verify invoker was called.
	if len(inv.calls) != 1 {
		t.Errorf("invoker calls = %d, want 1", len(inv.calls))
	}
}

// --- Advance tests ---

func TestEngine_Advance_success(t *testing.T) {
	inv := &mockInvoker{
		resultFunc: func(b model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			if b.OperationID == "processOrder" {
				return model.InvocationResult{
					StatusCode: 200,
					Body:       map[string]any{"confirmed_at": "2025-01-15T10:45:00Z"},
				}, nil
			}
			return model.InvocationResult{StatusCode: 200, Body: map[string]any{}}, nil
		},
	}
	engine, store := newTestEngine(inv, model.CapabilitySet{
		"orders:approve:execute": true,
		"orders:approve:review":  true,
	})
	ctx := context.Background()
	rctx := testRctx()

	// Start workflow.
	inst, _ := engine.Start(ctx, rctx, "orders.approval", map[string]any{"order_id": "ord-123"})

	// Advance with "approved".
	updated, err := engine.Advance(ctx, rctx, inst.ID, "approved", map[string]any{"approval_notes": "LGTM"})
	if err != nil {
		t.Fatalf("Advance error: %v", err)
	}

	// Should auto-execute process → notify → approved (terminal).
	if updated.CurrentStep != "approved" {
		t.Errorf("CurrentStep = %q, want approved", updated.CurrentStep)
	}
	if updated.Status != model.WorkflowStatusCompleted {
		t.Errorf("Status = %q, want completed", updated.Status)
	}

	// Verify state has merged data.
	got, _ := store.Get(ctx, "tenant-1", inst.ID)
	if got.State["approval_notes"] != "LGTM" {
		t.Errorf("State[approval_notes] = %v", got.State["approval_notes"])
	}
	if got.State["confirmed_at"] != "2025-01-15T10:45:00Z" {
		t.Errorf("State[confirmed_at] = %v", got.State["confirmed_at"])
	}

	// Verify events were recorded.
	events, _ := store.GetEvents(ctx, "tenant-1", inst.ID)
	// step_entered(review) + step_completed(review) + approved(review) +
	// step_entered(process) + step_completed(process) + step_entered(notify) +
	// step_completed(notify) + step_entered(approved) + workflow_completed
	if len(events) < 7 {
		t.Errorf("events count = %d, want at least 7", len(events))
	}

	// Verify invoker was called for both system steps.
	if len(inv.calls) != 2 {
		t.Errorf("invoker calls = %d, want 2 (process + notify)", len(inv.calls))
	}
}

func TestEngine_Advance_rejection(t *testing.T) {
	engine, _ := newTestEngine(nil, model.CapabilitySet{
		"orders:approve:execute": true,
		"orders:approve:review":  true,
	})
	ctx := context.Background()
	rctx := testRctx()

	inst, _ := engine.Start(ctx, rctx, "orders.approval", nil)
	updated, err := engine.Advance(ctx, rctx, inst.ID, "rejected", map[string]any{"reason": "Budget exceeded"})
	if err != nil {
		t.Fatalf("Advance error: %v", err)
	}
	if updated.CurrentStep != "rejected" {
		t.Errorf("CurrentStep = %q, want rejected", updated.CurrentStep)
	}
	if updated.Status != model.WorkflowStatusCompleted {
		t.Errorf("Status = %q, want completed", updated.Status)
	}
}

func TestEngine_Advance_notActive(t *testing.T) {
	engine, _ := newTestEngine(nil, model.CapabilitySet{
		"orders:approve:execute": true,
		"orders:approve:review":  true,
	})
	ctx := context.Background()
	rctx := testRctx()

	inst, _ := engine.Start(ctx, rctx, "orders.approval", nil)
	// Advance to terminal (rejected).
	_, _ = engine.Advance(ctx, rctx, inst.ID, "rejected", nil)

	// Try to advance again.
	_, err := engine.Advance(ctx, rctx, inst.ID, "approved", nil)
	if err == nil {
		t.Fatal("expected not active error")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrWorkflowNotActive {
		t.Errorf("code = %s", envErr.Code)
	}
}

func TestEngine_Advance_invalidTransition(t *testing.T) {
	engine, _ := newTestEngine(nil, model.CapabilitySet{
		"orders:approve:execute": true,
		"orders:approve:review":  true,
	})
	ctx := context.Background()
	rctx := testRctx()

	inst, _ := engine.Start(ctx, rctx, "orders.approval", nil)

	_, err := engine.Advance(ctx, rctx, inst.ID, "nonexistent_event", nil)
	if err == nil {
		t.Fatal("expected invalid transition error")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrInvalidTransition {
		t.Errorf("code = %s", envErr.Code)
	}
}

func TestEngine_Advance_stepCapability(t *testing.T) {
	// User has workflow capability but NOT the step capability.
	engine, _ := newTestEngine(nil, model.CapabilitySet{
		"orders:approve:execute": true,
		// Missing: "orders:approve:review"
	})
	ctx := context.Background()
	rctx := testRctx()

	inst, _ := engine.Start(ctx, rctx, "orders.approval", nil)

	_, err := engine.Advance(ctx, rctx, inst.ID, "approved", nil)
	if err == nil {
		t.Fatal("expected forbidden error for step capability")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrForbidden {
		t.Errorf("code = %s", envErr.Code)
	}
}

func TestEngine_Advance_tenantIsolation(t *testing.T) {
	engine, _ := newTestEngine(nil, model.CapabilitySet{
		"orders:approve:execute": true,
		"orders:approve:review":  true,
	})
	ctx := context.Background()
	rctx := testRctx()

	inst, _ := engine.Start(ctx, rctx, "orders.approval", nil)

	// Different tenant.
	rctx2 := &model.RequestContext{
		SubjectID: "user-bob",
		TenantID:  "tenant-2",
	}
	_, err := engine.Advance(ctx, rctx2, inst.ID, "approved", nil)
	if err == nil {
		t.Fatal("expected not found error for different tenant")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrNotFound {
		t.Errorf("code = %s, want NOT_FOUND", envErr.Code)
	}
}

func TestEngine_Advance_conditionalTransition(t *testing.T) {
	engine, _ := newTestEngine(nil, model.CapabilitySet{"*": true})
	ctx := context.Background()
	rctx := testRctx()

	// High priority should go to fast-track.
	inst, _ := engine.Start(ctx, rctx, "orders.conditional", map[string]any{"priority": "high"})
	updated, err := engine.Advance(ctx, rctx, inst.ID, "submitted", nil)
	if err != nil {
		t.Fatalf("Advance error: %v", err)
	}
	if updated.CurrentStep != "fast-track" {
		t.Errorf("CurrentStep = %q, want fast-track", updated.CurrentStep)
	}
	if updated.Status != model.WorkflowStatusCompleted {
		t.Errorf("Status = %q, want completed", updated.Status)
	}

	// Low priority should go to standard.
	inst2, _ := engine.Start(ctx, rctx, "orders.conditional", map[string]any{"priority": "low"})
	updated2, err := engine.Advance(ctx, rctx, inst2.ID, "submitted", nil)
	if err != nil {
		t.Fatalf("Advance error: %v", err)
	}
	if updated2.CurrentStep != "standard" {
		t.Errorf("CurrentStep = %q, want standard", updated2.CurrentStep)
	}
}

func TestEngine_Advance_guardedTransition(t *testing.T) {
	// User without admin:access capability.
	engine, _ := newTestEngine(nil, model.CapabilitySet{"*": true})
	// Override cap resolver to not have admin:access.
	engine.capResolver = &mockCapResolver{caps: model.CapabilitySet{"basic:access": true}}
	ctx := context.Background()
	rctx := testRctx()

	inst, err := engine.Start(ctx, rctx, "orders.guarded", nil)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	_, err = engine.Advance(ctx, rctx, inst.ID, "proceed", nil)
	if err == nil {
		t.Fatal("expected forbidden error for guarded transition")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrForbidden {
		t.Errorf("code = %s", envErr.Code)
	}
}

func TestEngine_Advance_guardedTransition_allowed(t *testing.T) {
	engine, _ := newTestEngine(nil, model.CapabilitySet{"admin:access": true})
	ctx := context.Background()
	rctx := testRctx()

	inst, _ := engine.Start(ctx, rctx, "orders.guarded", nil)
	updated, err := engine.Advance(ctx, rctx, inst.ID, "proceed", nil)
	if err != nil {
		t.Fatalf("Advance error: %v", err)
	}
	if updated.CurrentStep != "admin-only" {
		t.Errorf("CurrentStep = %q, want admin-only", updated.CurrentStep)
	}
}

// --- System step error handling ---

func TestEngine_SystemStep_errorWithTransition(t *testing.T) {
	inv := &mockInvoker{
		resultFunc: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			return model.InvocationResult{}, fmt.Errorf("backend unavailable")
		},
	}
	engine, store := newTestEngine(inv, model.CapabilitySet{"*": true})
	ctx := context.Background()
	rctx := testRctx()

	inst, err := engine.Start(ctx, rctx, "orders.with-error-handling", nil)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	// System step should fail and transition to error-step.
	got, _ := store.Get(ctx, "tenant-1", inst.ID)
	if got.CurrentStep != "error-step" {
		t.Errorf("CurrentStep = %q, want error-step", got.CurrentStep)
	}
	if got.Status != model.WorkflowStatusCompleted {
		t.Errorf("Status = %q, want completed (error-step is terminal)", got.Status)
	}
	if got.State["_last_error"] == nil {
		t.Error("expected _last_error in state")
	}
}

func TestEngine_SystemStep_errorWithoutTransition(t *testing.T) {
	inv := &mockInvoker{
		resultFunc: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			return model.InvocationResult{}, fmt.Errorf("backend unavailable")
		},
	}
	// Use auto-process which has no error transition.
	engine, store := newTestEngine(inv, model.CapabilitySet{"*": true})
	ctx := context.Background()
	rctx := testRctx()

	inst, err := engine.Start(ctx, rctx, "orders.auto-process", nil)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	got, _ := store.Get(ctx, "tenant-1", inst.ID)
	if got.Status != model.WorkflowStatusSuspended {
		t.Errorf("Status = %q, want suspended (no error transition)", got.Status)
	}
}

func TestEngine_NotificationStep_errorIsBestEffort(t *testing.T) {
	callCount := 0
	inv := &mockInvoker{
		resultFunc: func(b model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			callCount++
			if b.OperationID == "sendNotification" {
				return model.InvocationResult{}, fmt.Errorf("notification service down")
			}
			return model.InvocationResult{
				StatusCode: 200,
				Body:       map[string]any{"confirmed_at": "2025-01-15"},
			}, nil
		},
	}
	engine, store := newTestEngine(inv, model.CapabilitySet{
		"orders:approve:execute": true,
		"orders:approve:review":  true,
	})
	ctx := context.Background()
	rctx := testRctx()

	inst, _ := engine.Start(ctx, rctx, "orders.approval", nil)
	updated, err := engine.Advance(ctx, rctx, inst.ID, "approved", nil)
	if err != nil {
		t.Fatalf("Advance error: %v", err)
	}

	// Notification failure should not block: workflow should still complete.
	if updated.CurrentStep != "approved" {
		t.Errorf("CurrentStep = %q, want approved", updated.CurrentStep)
	}
	if updated.Status != model.WorkflowStatusCompleted {
		t.Errorf("Status = %q, want completed", updated.Status)
	}

	// Verify _last_error was recorded.
	got, _ := store.Get(ctx, "tenant-1", inst.ID)
	if got.State["_last_error"] == nil {
		t.Error("expected _last_error from notification failure")
	}
}

// --- Chain limit ---

func TestEngine_ChainLimit(t *testing.T) {
	inv := &mockInvoker{}
	engine, store := newTestEngine(inv, model.CapabilitySet{"*": true})
	engine.chainLimit = 5 // Lower limit for testing.
	ctx := context.Background()
	rctx := testRctx()

	_, err := engine.Start(ctx, rctx, "orders.chain-loop", nil)
	if err == nil {
		t.Fatal("expected chain limit error")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrWorkflowChainLimit {
		t.Errorf("code = %s", envErr.Code)
	}

	// Verify the workflow is suspended.
	// Find the only instance.
	all, _ := store.FindActive(ctx, "tenant-1", WorkflowFilters{})
	if len(all) != 0 {
		t.Errorf("should have no active instances (suspended is not active)")
	}
}

// --- Get ---

func TestEngine_Get_success(t *testing.T) {
	engine, _ := newTestEngine(nil, model.CapabilitySet{
		"orders:approve:execute": true,
		"orders:approve:review":  true,
	})
	ctx := context.Background()
	rctx := testRctx()

	inst, _ := engine.Start(ctx, rctx, "orders.approval", nil)

	desc, err := engine.Get(ctx, rctx, inst.ID)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if desc.ID != inst.ID {
		t.Errorf("desc.ID = %q", desc.ID)
	}
	if desc.WorkflowID != "orders.approval" {
		t.Errorf("desc.WorkflowID = %q", desc.WorkflowID)
	}
	if desc.Name != "Order Approval" {
		t.Errorf("desc.Name = %q", desc.Name)
	}
	if desc.Status != model.WorkflowStatusActive {
		t.Errorf("desc.Status = %q", desc.Status)
	}
	if desc.CurrentStep == nil {
		t.Fatal("expected current step descriptor")
	}
	if desc.CurrentStep.ID != "review" {
		t.Errorf("CurrentStep.ID = %q", desc.CurrentStep.ID)
	}
	if desc.CurrentStep.Type != "action" {
		t.Errorf("CurrentStep.Type = %q", desc.CurrentStep.Type)
	}
	if len(desc.Steps) == 0 {
		t.Error("expected step summaries")
	}
	if len(desc.History) == 0 {
		t.Error("expected history entries")
	}
}

func TestEngine_Get_notFound(t *testing.T) {
	engine, _ := newTestEngine(nil, model.CapabilitySet{"*": true})
	ctx := context.Background()
	rctx := testRctx()

	_, err := engine.Get(ctx, rctx, "nonexistent")
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestEngine_Get_tenantIsolation(t *testing.T) {
	engine, _ := newTestEngine(nil, model.CapabilitySet{
		"orders:approve:execute": true,
	})
	ctx := context.Background()
	rctx := testRctx()

	inst, _ := engine.Start(ctx, rctx, "orders.approval", nil)

	rctx2 := &model.RequestContext{SubjectID: "user-bob", TenantID: "tenant-2"}
	_, err := engine.Get(ctx, rctx2, inst.ID)
	if err == nil {
		t.Fatal("expected not found error for different tenant")
	}
}

// --- Cancel ---

func TestEngine_Cancel_success(t *testing.T) {
	engine, store := newTestEngine(nil, model.CapabilitySet{
		"orders:approve:execute": true,
	})
	ctx := context.Background()
	rctx := testRctx()

	inst, _ := engine.Start(ctx, rctx, "orders.approval", nil)

	err := engine.Cancel(ctx, rctx, inst.ID, "Customer requested cancellation")
	if err != nil {
		t.Fatalf("Cancel error: %v", err)
	}

	got, _ := store.Get(ctx, "tenant-1", inst.ID)
	if got.Status != model.WorkflowStatusCancelled {
		t.Errorf("Status = %q, want cancelled", got.Status)
	}

	// Verify cancelled event.
	events, _ := store.GetEvents(ctx, "tenant-1", inst.ID)
	var found bool
	for _, e := range events {
		if e.Event == "cancelled" {
			found = true
			if e.Comment != "Customer requested cancellation" {
				t.Errorf("Comment = %q", e.Comment)
			}
		}
	}
	if !found {
		t.Error("expected cancelled event")
	}
}

func TestEngine_Cancel_alreadyCompleted(t *testing.T) {
	engine, _ := newTestEngine(nil, model.CapabilitySet{
		"orders:approve:execute": true,
		"orders:approve:review":  true,
	})
	ctx := context.Background()
	rctx := testRctx()

	inst, _ := engine.Start(ctx, rctx, "orders.approval", nil)
	_, _ = engine.Advance(ctx, rctx, inst.ID, "rejected", nil)

	err := engine.Cancel(ctx, rctx, inst.ID, "too late")
	if err == nil {
		t.Fatal("expected not active error")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrWorkflowNotActive {
		t.Errorf("code = %s", envErr.Code)
	}
}

func TestEngine_Cancel_tenantIsolation(t *testing.T) {
	engine, _ := newTestEngine(nil, model.CapabilitySet{
		"orders:approve:execute": true,
	})
	ctx := context.Background()
	rctx := testRctx()

	inst, _ := engine.Start(ctx, rctx, "orders.approval", nil)

	rctx2 := &model.RequestContext{SubjectID: "user-bob", TenantID: "tenant-2"}
	err := engine.Cancel(ctx, rctx2, inst.ID, "unauthorized cancel")
	if err == nil {
		t.Fatal("expected not found error for different tenant")
	}
}

// --- List ---

func TestEngine_List_success(t *testing.T) {
	engine, _ := newTestEngine(nil, model.CapabilitySet{
		"orders:approve:execute": true,
	})
	ctx := context.Background()
	rctx := testRctx()

	_, _ = engine.Start(ctx, rctx, "orders.approval", nil)
	_, _ = engine.Start(ctx, rctx, "orders.approval", nil)

	summaries, total, err := engine.List(ctx, rctx, model.WorkflowFilters{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(summaries) != 2 {
		t.Errorf("summaries count = %d, want 2", len(summaries))
	}
	if summaries[0].Name != "Order Approval" {
		t.Errorf("summaries[0].Name = %q", summaries[0].Name)
	}
}

func TestEngine_List_pagination(t *testing.T) {
	engine, _ := newTestEngine(nil, model.CapabilitySet{
		"orders:approve:execute": true,
	})
	ctx := context.Background()
	rctx := testRctx()

	for i := 0; i < 5; i++ {
		_, _ = engine.Start(ctx, rctx, "orders.approval", nil)
	}

	summaries, total, err := engine.List(ctx, rctx, model.WorkflowFilters{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(summaries) != 2 {
		t.Errorf("summaries count = %d, want 2", len(summaries))
	}
}

func TestEngine_List_tenantIsolation(t *testing.T) {
	engine, _ := newTestEngine(nil, model.CapabilitySet{
		"orders:approve:execute": true,
	})
	ctx := context.Background()
	rctx := testRctx()

	_, _ = engine.Start(ctx, rctx, "orders.approval", nil)

	rctx2 := &model.RequestContext{SubjectID: "user-bob", TenantID: "tenant-2"}
	summaries, total, err := engine.List(ctx, rctx2, model.WorkflowFilters{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(summaries) != 0 {
		t.Errorf("summaries count = %d, want 0", len(summaries))
	}
}

// --- ProcessTimeouts ---

func TestEngine_ProcessTimeouts_withHandler(t *testing.T) {
	engine, store := newTestEngine(nil, model.CapabilitySet{
		"orders:approve:execute": true,
	})
	ctx := context.Background()
	rctx := testRctx()

	inst, _ := engine.Start(ctx, rctx, "orders.approval", nil)

	// Manually set expiry to past.
	got, _ := store.Get(ctx, "tenant-1", inst.ID)
	past := time.Now().Add(-1 * time.Hour)
	got.ExpiresAt = &past
	_ = store.Update(ctx, got)

	// Process timeouts.
	err := engine.ProcessTimeouts(ctx)
	if err != nil {
		t.Fatalf("ProcessTimeouts error: %v", err)
	}

	// Workflow should transition to "expired" (terminal).
	updated, _ := store.Get(ctx, "tenant-1", inst.ID)
	if updated.CurrentStep != "expired" {
		t.Errorf("CurrentStep = %q, want expired", updated.CurrentStep)
	}
	if updated.Status != model.WorkflowStatusCompleted {
		t.Errorf("Status = %q, want completed", updated.Status)
	}

	// Verify timeout event.
	events, _ := store.GetEvents(ctx, "tenant-1", inst.ID)
	var foundTimeout bool
	for _, e := range events {
		if e.Event == "timeout" {
			foundTimeout = true
		}
	}
	if !foundTimeout {
		t.Error("expected timeout event")
	}
}

func TestEngine_ProcessTimeouts_noHandler(t *testing.T) {
	engine, store := newTestEngine(nil, model.CapabilitySet{"*": true})
	ctx := context.Background()
	rctx := testRctx()

	// auto-process has no timeout handler.
	// We need to start a workflow that stops at a human step.
	inv := &mockInvoker{
		resultFunc: func(_ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
			return model.InvocationResult{StatusCode: 200, Body: map[string]any{"total": 10}}, nil
		},
	}
	engine.invokers = invoker.NewRegistry()
	engine.invokers.Register(inv)

	inst, _ := engine.Start(ctx, rctx, "orders.auto-process", nil)

	// Manually set expiry to past.
	got, _ := store.Get(ctx, "tenant-1", inst.ID)
	past := time.Now().Add(-1 * time.Hour)
	got.ExpiresAt = &past
	_ = store.Update(ctx, got)

	err := engine.ProcessTimeouts(ctx)
	if err != nil {
		t.Fatalf("ProcessTimeouts error: %v", err)
	}

	// No timeout handler: should fail the workflow.
	updated, _ := store.Get(ctx, "tenant-1", inst.ID)
	if updated.Status != model.WorkflowStatusFailed {
		t.Errorf("Status = %q, want failed (no timeout handler)", updated.Status)
	}
}

// --- Helper tests ---

func TestFindStep(t *testing.T) {
	wfDef := testWorkflowDefinitions()[0].Workflows[0]

	step := findStep(wfDef, "review")
	if step == nil {
		t.Fatal("expected to find step")
	}
	if step.ID != "review" {
		t.Errorf("step.ID = %q", step.ID)
	}

	missing := findStep(wfDef, "nonexistent")
	if missing != nil {
		t.Error("expected nil for missing step")
	}
}

func TestFindTransition(t *testing.T) {
	wfDef := testWorkflowDefinitions()[0].Workflows[0]

	tr := findTransition(wfDef, "review", "approved", nil)
	if tr == nil {
		t.Fatal("expected to find transition")
	}
	if tr.To != "process" {
		t.Errorf("transition.To = %q", tr.To)
	}

	missing := findTransition(wfDef, "review", "nonexistent", nil)
	if missing != nil {
		t.Error("expected nil for missing transition")
	}
}

func TestEvaluateCondition(t *testing.T) {
	state := map[string]any{"priority": "high", "status": "open"}

	tests := []struct {
		condition string
		want      bool
	}{
		{"priority == 'high'", true},
		{"priority == 'low'", false},
		{"priority != 'low'", true},
		{"priority != 'high'", false},
		{"status == 'open'", true},
		{"unparseable condition", true}, // Unparseable → permissive.
	}

	for _, tt := range tests {
		got := evaluateCondition(tt.condition, state)
		if got != tt.want {
			t.Errorf("evaluateCondition(%q) = %v, want %v", tt.condition, got, tt.want)
		}
	}
}

func TestIsAutoStep(t *testing.T) {
	if !isAutoStep("system") {
		t.Error("system should be auto")
	}
	if !isAutoStep("notification") {
		t.Error("notification should be auto")
	}
	if isAutoStep("action") {
		t.Error("action should not be auto")
	}
	if isAutoStep("terminal") {
		t.Error("terminal should not be auto")
	}
}

func TestApplyOutputMapping(t *testing.T) {
	result := model.InvocationResult{
		StatusCode: 200,
		Body:       map[string]any{"confirmed_at": "2025-01-15", "extra": "data"},
	}

	// With mapping.
	mapping := &model.OutputMapping{
		Fields: map[string]string{"confirmed": "confirmed_at"},
	}
	merged := applyOutputMapping(mapping, result)
	if merged["confirmed"] != "2025-01-15" {
		t.Errorf("merged[confirmed] = %v", merged["confirmed"])
	}
	if _, ok := merged["extra"]; ok {
		t.Error("should not include unmapped fields")
	}

	// Without mapping: merge entire body.
	allMerged := applyOutputMapping(nil, result)
	if allMerged["confirmed_at"] != "2025-01-15" {
		t.Errorf("allMerged[confirmed_at] = %v", allMerged["confirmed_at"])
	}
	if allMerged["extra"] != "data" {
		t.Errorf("allMerged[extra] = %v", allMerged["extra"])
	}
}

func TestComputeStepStatus(t *testing.T) {
	inst := model.WorkflowInstance{
		CurrentStep: "review",
		Status:      model.WorkflowStatusActive,
	}

	if s := computeStepStatus("review", inst); s != model.StepStatusInProgress {
		t.Errorf("current step status = %q, want in_progress", s)
	}
	if s := computeStepStatus("other", inst); s != model.StepStatusFuture {
		t.Errorf("other step status = %q, want future", s)
	}

	// Completed workflow.
	inst.Status = model.WorkflowStatusCompleted
	if s := computeStepStatus("review", inst); s != model.WorkflowStatusCompleted {
		t.Errorf("current step on completed workflow = %q, want completed", s)
	}
}
