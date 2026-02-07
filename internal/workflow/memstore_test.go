package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/pitabwire/thesa/model"
)

func testInstance(id, tenantID, workflowID, step string) model.WorkflowInstance {
	return model.WorkflowInstance{
		ID:          id,
		WorkflowID:  workflowID,
		TenantID:    tenantID,
		PartitionID: "partition-1",
		SubjectID:   "user-alice",
		CurrentStep: step,
		Status:      model.WorkflowStatusActive,
		State:       map[string]any{"key": "val"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		Version:     1,
	}
}

// --- Create ---

func TestMemoryWorkflowStore_Create(t *testing.T) {
	store := NewMemoryWorkflowStore()
	inst := testInstance("wf-1", "tenant-1", "approval", "review")

	err := store.Create(context.Background(), inst)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if store.Len() != 1 {
		t.Errorf("Len() = %d, want 1", store.Len())
	}
}

func TestMemoryWorkflowStore_Create_duplicate(t *testing.T) {
	store := NewMemoryWorkflowStore()
	inst := testInstance("wf-1", "tenant-1", "approval", "review")

	_ = store.Create(context.Background(), inst)
	err := store.Create(context.Background(), inst)
	if err == nil {
		t.Fatal("expected conflict error for duplicate")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrConflict {
		t.Errorf("code = %s, want %s", envErr.Code, model.ErrConflict)
	}
}

// --- Get ---

func TestMemoryWorkflowStore_Get(t *testing.T) {
	store := NewMemoryWorkflowStore()
	inst := testInstance("wf-1", "tenant-1", "approval", "review")
	_ = store.Create(context.Background(), inst)

	got, err := store.Get(context.Background(), "tenant-1", "wf-1")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.ID != "wf-1" {
		t.Errorf("ID = %q", got.ID)
	}
	if got.CurrentStep != "review" {
		t.Errorf("CurrentStep = %q", got.CurrentStep)
	}
}

func TestMemoryWorkflowStore_Get_notFound(t *testing.T) {
	store := NewMemoryWorkflowStore()

	_, err := store.Get(context.Background(), "tenant-1", "nonexistent")
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

func TestMemoryWorkflowStore_Get_tenantIsolation(t *testing.T) {
	store := NewMemoryWorkflowStore()
	inst := testInstance("wf-1", "tenant-1", "approval", "review")
	_ = store.Create(context.Background(), inst)

	// Different tenant should not see it.
	_, err := store.Get(context.Background(), "tenant-2", "wf-1")
	if err == nil {
		t.Fatal("expected not found error (tenant isolation)")
	}
}

// --- Update ---

func TestMemoryWorkflowStore_Update(t *testing.T) {
	store := NewMemoryWorkflowStore()
	inst := testInstance("wf-1", "tenant-1", "approval", "review")
	_ = store.Create(context.Background(), inst)

	inst.CurrentStep = "approve"
	inst.State["approved"] = true
	err := store.Update(context.Background(), inst)
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	got, _ := store.Get(context.Background(), "tenant-1", "wf-1")
	if got.CurrentStep != "approve" {
		t.Errorf("CurrentStep = %q, want approve", got.CurrentStep)
	}
	if got.Version != 2 {
		t.Errorf("Version = %d, want 2", got.Version)
	}
	if got.State["approved"] != true {
		t.Errorf("State[approved] = %v", got.State["approved"])
	}
}

func TestMemoryWorkflowStore_Update_versionConflict(t *testing.T) {
	store := NewMemoryWorkflowStore()
	inst := testInstance("wf-1", "tenant-1", "approval", "review")
	_ = store.Create(context.Background(), inst)

	// First update succeeds.
	inst.CurrentStep = "approve"
	_ = store.Update(context.Background(), inst)

	// Second update with same version should fail.
	inst.CurrentStep = "reject"
	err := store.Update(context.Background(), inst)
	if err == nil {
		t.Fatal("expected version conflict error")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if envErr.Code != model.ErrConflict {
		t.Errorf("code = %s, want %s", envErr.Code, model.ErrConflict)
	}
}

func TestMemoryWorkflowStore_Update_notFound(t *testing.T) {
	store := NewMemoryWorkflowStore()
	inst := testInstance("wf-1", "tenant-1", "approval", "review")

	err := store.Update(context.Background(), inst)
	if err == nil {
		t.Fatal("expected not found error")
	}
}

// --- Events ---

func TestMemoryWorkflowStore_AppendAndGetEvents(t *testing.T) {
	store := NewMemoryWorkflowStore()
	inst := testInstance("wf-1", "tenant-1", "approval", "review")
	_ = store.Create(context.Background(), inst)

	now := time.Now().UTC()
	events := []model.WorkflowEvent{
		{ID: "evt-1", WorkflowInstanceID: "wf-1", StepID: "review", Event: "step_entered", ActorID: "system", Timestamp: now},
		{ID: "evt-2", WorkflowInstanceID: "wf-1", StepID: "review", Event: "approved", ActorID: "user-alice", Timestamp: now.Add(time.Minute), Comment: "LGTM"},
	}

	for _, evt := range events {
		if err := store.AppendEvent(context.Background(), evt); err != nil {
			t.Fatalf("AppendEvent error: %v", err)
		}
	}

	got, err := store.GetEvents(context.Background(), "tenant-1", "wf-1")
	if err != nil {
		t.Fatalf("GetEvents error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(got))
	}
	if got[0].Event != "step_entered" {
		t.Errorf("events[0].Event = %q", got[0].Event)
	}
	if got[1].Comment != "LGTM" {
		t.Errorf("events[1].Comment = %q", got[1].Comment)
	}
}

func TestMemoryWorkflowStore_GetEvents_tenantIsolation(t *testing.T) {
	store := NewMemoryWorkflowStore()
	inst := testInstance("wf-1", "tenant-1", "approval", "review")
	_ = store.Create(context.Background(), inst)

	_ = store.AppendEvent(context.Background(), model.WorkflowEvent{
		ID: "evt-1", WorkflowInstanceID: "wf-1", Event: "test", ActorID: "system", Timestamp: time.Now(),
	})

	_, err := store.GetEvents(context.Background(), "tenant-2", "wf-1")
	if err == nil {
		t.Fatal("expected not found error (tenant isolation)")
	}
}

func TestMemoryWorkflowStore_GetEvents_sortedByTimestamp(t *testing.T) {
	store := NewMemoryWorkflowStore()
	inst := testInstance("wf-1", "tenant-1", "approval", "review")
	_ = store.Create(context.Background(), inst)

	now := time.Now().UTC()
	// Insert in reverse order.
	_ = store.AppendEvent(context.Background(), model.WorkflowEvent{
		ID: "evt-2", WorkflowInstanceID: "wf-1", Event: "second", ActorID: "system", Timestamp: now.Add(time.Minute),
	})
	_ = store.AppendEvent(context.Background(), model.WorkflowEvent{
		ID: "evt-1", WorkflowInstanceID: "wf-1", Event: "first", ActorID: "system", Timestamp: now,
	})

	got, _ := store.GetEvents(context.Background(), "tenant-1", "wf-1")
	if got[0].Event != "first" {
		t.Error("events should be sorted by timestamp ascending")
	}
}

// --- FindActive ---

func TestMemoryWorkflowStore_FindActive(t *testing.T) {
	store := NewMemoryWorkflowStore()

	inst1 := testInstance("wf-1", "tenant-1", "approval", "review")
	inst1.CreatedAt = time.Now().Add(-2 * time.Hour)
	inst2 := testInstance("wf-2", "tenant-1", "approval", "submit")
	inst2.CreatedAt = time.Now().Add(-1 * time.Hour)
	inst3 := testInstance("wf-3", "tenant-1", "approval", "done")
	inst3.Status = model.WorkflowStatusCompleted // Not active.
	inst4 := testInstance("wf-4", "tenant-2", "approval", "review") // Different tenant.

	_ = store.Create(context.Background(), inst1)
	_ = store.Create(context.Background(), inst2)
	_ = store.Create(context.Background(), inst3)
	_ = store.Create(context.Background(), inst4)

	result, err := store.FindActive(context.Background(), "tenant-1", WorkflowFilters{})
	if err != nil {
		t.Fatalf("FindActive error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2 (active only, same tenant)", len(result))
	}
	// Should be sorted by created_at descending.
	if result[0].ID != "wf-2" {
		t.Errorf("result[0].ID = %q, want wf-2 (most recent)", result[0].ID)
	}
}

func TestMemoryWorkflowStore_FindActive_withFilter(t *testing.T) {
	store := NewMemoryWorkflowStore()

	inst1 := testInstance("wf-1", "tenant-1", "approval", "review")
	inst2 := testInstance("wf-2", "tenant-1", "onboarding", "submit")

	_ = store.Create(context.Background(), inst1)
	_ = store.Create(context.Background(), inst2)

	result, err := store.FindActive(context.Background(), "tenant-1", WorkflowFilters{WorkflowID: "approval"})
	if err != nil {
		t.Fatalf("FindActive error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}
	if result[0].WorkflowID != "approval" {
		t.Errorf("WorkflowID = %q", result[0].WorkflowID)
	}
}

func TestMemoryWorkflowStore_FindActive_pagination(t *testing.T) {
	store := NewMemoryWorkflowStore()

	for i := 0; i < 5; i++ {
		inst := testInstance(
			"wf-"+string(rune('a'+i)),
			"tenant-1", "approval", "review",
		)
		inst.CreatedAt = time.Now().Add(time.Duration(i) * time.Hour)
		_ = store.Create(context.Background(), inst)
	}

	result, err := store.FindActive(context.Background(), "tenant-1", WorkflowFilters{Limit: 2})
	if err != nil {
		t.Fatalf("FindActive error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2 (limit)", len(result))
	}

	result, err = store.FindActive(context.Background(), "tenant-1", WorkflowFilters{Offset: 3})
	if err != nil {
		t.Fatalf("FindActive error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("len(result) = %d, want 2 (offset 3 of 5)", len(result))
	}
}

// --- FindExpired ---

func TestMemoryWorkflowStore_FindExpired(t *testing.T) {
	store := NewMemoryWorkflowStore()
	now := time.Now().UTC()

	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	inst1 := testInstance("wf-1", "tenant-1", "approval", "review")
	inst1.ExpiresAt = &past // Expired.

	inst2 := testInstance("wf-2", "tenant-1", "approval", "review")
	inst2.ExpiresAt = &future // Not expired.

	inst3 := testInstance("wf-3", "tenant-2", "approval", "review")
	// No expiry.

	inst4 := testInstance("wf-4", "tenant-1", "approval", "done")
	inst4.Status = model.WorkflowStatusCompleted // Not active.
	inst4.ExpiresAt = &past

	_ = store.Create(context.Background(), inst1)
	_ = store.Create(context.Background(), inst2)
	_ = store.Create(context.Background(), inst3)
	_ = store.Create(context.Background(), inst4)

	result, err := store.FindExpired(context.Background(), now)
	if err != nil {
		t.Fatalf("FindExpired error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1 (only wf-1)", len(result))
	}
	if result[0].ID != "wf-1" {
		t.Errorf("result[0].ID = %q, want wf-1", result[0].ID)
	}
}

// --- Delete ---

func TestMemoryWorkflowStore_Delete(t *testing.T) {
	store := NewMemoryWorkflowStore()
	inst := testInstance("wf-1", "tenant-1", "approval", "review")
	_ = store.Create(context.Background(), inst)
	_ = store.AppendEvent(context.Background(), model.WorkflowEvent{
		ID: "evt-1", WorkflowInstanceID: "wf-1", Event: "test", ActorID: "system", Timestamp: time.Now(),
	})

	err := store.Delete(context.Background(), "tenant-1", "wf-1")
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if store.Len() != 0 {
		t.Errorf("Len() = %d, want 0", store.Len())
	}

	// Events should also be cleaned up.
	_, err = store.GetEvents(context.Background(), "tenant-1", "wf-1")
	if err == nil {
		t.Error("expected not found error after delete")
	}
}

func TestMemoryWorkflowStore_Delete_notFound(t *testing.T) {
	store := NewMemoryWorkflowStore()

	err := store.Delete(context.Background(), "tenant-1", "nonexistent")
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestMemoryWorkflowStore_Delete_tenantIsolation(t *testing.T) {
	store := NewMemoryWorkflowStore()
	inst := testInstance("wf-1", "tenant-1", "approval", "review")
	_ = store.Create(context.Background(), inst)

	err := store.Delete(context.Background(), "tenant-2", "wf-1")
	if err == nil {
		t.Fatal("expected not found error (tenant isolation)")
	}
	// Instance should still exist.
	if store.Len() != 1 {
		t.Errorf("Len() = %d, want 1", store.Len())
	}
}
