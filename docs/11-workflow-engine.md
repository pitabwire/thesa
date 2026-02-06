# 11 — Workflow Engine

This document describes the workflow engine — a persistent, definition-driven state
machine that manages multi-step processes like approvals, dispute resolution, onboarding,
and settlement flows.

---

## What Is a Workflow?

A workflow is a **multi-step, stateful interaction** that:

- Spans multiple user actions (not a single request/response).
- May involve multiple users (e.g., submitter and approver).
- May span hours or days.
- Includes automatic (system) steps as well as human steps.
- Must survive server restarts and session changes.
- Must enforce authorization per step.
- Must be auditable (who did what, when).

### Examples

| Workflow | Steps | Duration |
|----------|-------|----------|
| Order approval | Submit → Review → Approve/Reject → Process | Hours |
| Dispute resolution | Open → Evidence → Review → Resolve → Close | Days |
| User onboarding | Register → Verify Email → Configure → Activate | Minutes to days |
| Settlement | Initiate → Validate → Execute → Confirm → Complete | Minutes |
| Expense approval | Submit → Manager Review → Finance Review → Approve/Reject | Hours to days |

---

## Workflow Concepts

### WorkflowDefinition

The declarative blueprint for a workflow. Authored in YAML as part of a domain
definition. Contains steps, transitions, timeouts, and capability requirements.
See [06 — Definition Schema Reference](06-definition-schema-reference.md#workflowdefinition).

### WorkflowInstance

A runtime instantiation of a WorkflowDefinition. Created when a user starts a
workflow. Persisted to the WorkflowStore. Contains the current step, accumulated
state, and metadata.

### Step

A single stage in a workflow. Each step has a type that determines its behavior:

| Type | User Required? | Backend Call? | Description |
|------|---------------|---------------|-------------|
| `action` | Yes | Optional | User provides input through a form |
| `approval` | Yes | Optional | Special action: approve or reject |
| `system` | No | Yes | Automatic backend invocation |
| `wait` | No | No | Pause for duration or external event |
| `notification` | No | Yes (fire-and-forget) | Send notification, non-blocking |
| `terminal` | No | No | End state. Workflow is complete. |

### Transition

A rule that moves the workflow from one step to another when a specific event occurs.
Transitions can be conditional (based on workflow state) and guarded (requiring a
specific capability).

### WorkflowEvent

An immutable audit record of something that happened in the workflow. Events are
append-only — they are never modified or deleted.

---

## Workflow State Model

### WorkflowInstance

```
WorkflowInstance
  ├── ID                string          // Unique instance ID (UUID v4)
  ├── WorkflowID        string          // References WorkflowDefinition.ID
  ├── TenantID          string          // Tenant isolation
  ├── PartitionID       string          // Partition isolation
  ├── SubjectID         string          // Who started the workflow
  ├── CurrentStep       string          // Current step ID
  ├── Status            string          // "active", "completed", "failed", "cancelled", "suspended"
  ├── State             map[string]any  // Accumulated workflow state
  ├── Version           int             // Optimistic locking version
  ├── CreatedAt         time.Time
  ├── UpdatedAt         time.Time
  ├── ExpiresAt         *time.Time      // Computed from workflow timeout
  └── IdempotencyKey    string          // Prevents duplicate workflow creation
```

### Status Lifecycle

```
         start
           │
           ▼
     ┌──────────┐
     │  active   │◄─────────────────────────┐
     └────┬──────┘                          │
          │                                 │
  ┌───────┼──────────┬───────────┐         │
  │       │          │           │         │
  ▼       ▼          ▼           ▼         │
completed failed  cancelled  suspended ────┘
                                  (resume)
```

| Status | Meaning | Can Advance? | Can Cancel? |
|--------|---------|-------------|-------------|
| `active` | Workflow is in progress | Yes | Yes |
| `completed` | All steps done, reached terminal state | No | No |
| `failed` | A step failed and no error transition exists | No | No |
| `cancelled` | Explicitly cancelled by user or system | No | No |
| `suspended` | Paused (e.g., awaiting external event) | No (until resumed) | Yes |

### Workflow State (Accumulator)

The `State` field is a JSON-compatible map that accumulates data across steps.
Each step can read from and write to this state.

```
Workflow start:
  State = { "order_id": "ord-123", "customer_email": "bob@example.com" }

After "review" step (user provides approval notes):
  State = {
    "order_id": "ord-123",
    "customer_email": "bob@example.com",
    "approval_notes": "Looks good",
    "approved_by": "alice@acme-corp.com"
  }

After "process" step (system call returns confirmation):
  State = {
    "order_id": "ord-123",
    "customer_email": "bob@example.com",
    "approval_notes": "Looks good",
    "approved_by": "alice@acme-corp.com",
    "confirmed_at": "2025-01-15T10:45:00Z"
  }
```

Step input mappings use `workflow.{field}` to read from this state, and step output
mappings merge their results back into it.

### WorkflowEvent (Audit Trail)

```
WorkflowEvent
  ├── ID                    string
  ├── WorkflowInstanceID    string
  ├── StepID                string
  ├── Event                 string      // "step_entered", "step_completed", "step_failed",
  │                                     // "approved", "rejected", "transition", "timeout",
  │                                     // "cancelled", "workflow_completed", "workflow_failed"
  ├── ActorID               string      // Who triggered this event (user ID or "system")
  ├── Data                  map[string]any  // Event payload
  ├── Comment               string      // Optional human comment
  └── Timestamp             time.Time
```

Events are **append-only**. They provide a complete audit trail of the workflow.

---

## WorkflowEngine Interface

```
WorkflowEngine
  ├── Start(ctx, rctx, workflowId string, input map[string]any) → (WorkflowInstance, error)
  │     Creates a new workflow instance and enters the initial step.
  │     If the initial step is a system step, it is executed automatically.
  │
  ├── Advance(ctx, rctx, instanceId string, event string, input map[string]any) → (WorkflowInstance, error)
  │     Advances the workflow by processing the given event on the current step.
  │     Finds matching transition, executes system steps, updates state.
  │
  ├── Get(ctx, rctx, instanceId string) → (WorkflowDescriptor, error)
  │     Returns the workflow descriptor for the frontend (filtered by capabilities).
  │
  ├── Cancel(ctx, rctx, instanceId string, reason string) → error
  │     Cancels an active workflow.
  │
  ├── List(ctx, rctx, filters WorkflowFilters) → ([]WorkflowSummary, int, error)
  │     Lists workflow instances for the current user/tenant.
  │
  └── ProcessTimeouts(ctx) → error
        Background job: finds expired workflows and processes their timeouts.
```

---

## Workflow Lifecycle: Start

```
POST /ui/workflows/orders.approval/start
  { "order_id": "ord-123", "customer_email": "bob@example.com" }

Engine.Start(ctx, rctx, "orders.approval", input):

  1. Look up WorkflowDefinition "orders.approval" in registry.
     → 404 if not found.

  2. Evaluate workflow-level capabilities.
     → 403 if user lacks "orders:approve:execute".

  3. Check idempotency (if idempotency_key provided):
     → If duplicate: return existing instance.

  4. Create WorkflowInstance:
     {
       ID: uuid.New(),
       WorkflowID: "orders.approval",
       TenantID: rctx.TenantID,
       PartitionID: rctx.PartitionID,
       SubjectID: rctx.SubjectID,
       CurrentStep: "review",
       Status: "active",
       State: { "order_id": "ord-123", "customer_email": "bob@example.com" },
       Version: 1,
       CreatedAt: now,
       UpdatedAt: now,
       ExpiresAt: now + 72h,
     }

  5. Persist instance to WorkflowStore.

  6. Append event: { StepID: "review", Event: "step_entered", ActorID: rctx.SubjectID }

  7. Check initial step type:
     a. If "action" or "approval": stop here (wait for user).
     b. If "system" or "notification": execute immediately (see System Step Execution).

  8. Return WorkflowInstance.
```

---

## Workflow Lifecycle: Advance

```
POST /ui/workflows/wf-abc/advance
  { "event": "approved", "input": { "approval_notes": "Looks good" } }

Engine.Advance(ctx, rctx, "wf-abc", "approved", input):

  1. Load WorkflowInstance from store.
     → 404 if not found.

  2. Verify tenant isolation: instance.TenantID == rctx.TenantID.
     → 404 if mismatch (not 403, to prevent enumeration).

  3. Verify status is "active".
     → 409 if not active: { "code": "WORKFLOW_NOT_ACTIVE" }.

  4. Look up current StepDefinition.

  5. Evaluate step-level capabilities (if defined).
     → 403 if user lacks required capabilities.

  6. Find matching transition:
     - From: instance.CurrentStep ("review")
     - Event: "approved"
     - If transition has a condition: evaluate against workflow state.
     - If transition has a guard: verify user has the guard capability.
     → 422 if no valid transition found: { "code": "INVALID_TRANSITION" }.

  7. Merge input into workflow state:
     state["approval_notes"] = "Looks good"
     state["approved_by"] = rctx.SubjectID  (or from input if provided)

  8. Append event: { StepID: "review", Event: "step_completed", ActorID: rctx.SubjectID }
     Append event: { StepID: "review", Event: "approved", ActorID: rctx.SubjectID }

  9. Transition to next step:
     instance.CurrentStep = transition.To ("process")
     instance.UpdatedAt = now
     instance.Version++

  10. Append event: { StepID: "process", Event: "step_entered", ActorID: "system" }

  11. Execute step chain:
      "process" is type "system" → execute automatically.
      → See System Step Execution below.
      → If successful: auto-advance with "completed" event.
      → Continue chain until a human step or terminal state is reached.

  12. Persist updated instance (with optimistic locking on Version).
      → If version conflict: reload instance, retry from step 1 (another instance advanced it).

  13. Return updated WorkflowInstance.
```

---

## System Step Execution

When the workflow enters a `system` or `notification` step:

```
executeSystemStep(ctx, instance, step):

  1. Build InvocationInput from step.Input:
     - Resolve expressions against workflow state (workflow.order_id → "ord-123").
     - Resolve expressions against context (context.tenant_id → "acme-corp").

  2. Invoke backend via InvokerRegistry:
     - Use step.Operation binding.
     - For "openapi": dynamic HTTP call.
     - For "sdk": typed Go handler.

  3. On success (2xx):
     a. Apply step.Output mapping.
     b. Merge result into workflow state.
     c. Append event: { StepID: step.ID, Event: "step_completed" }
     d. Find transition with event "completed".
     e. If found: transition to next step.
        - If next step is also system → recursively execute (with depth limit).
        - If next step is terminal → set status = "completed".
        - If next step is human → stop, wait for user.
     f. If no transition: set status = "failed" (misconfigured workflow).

  4. On error:
     a. Record error in workflow state: state["_last_error"] = error details.
     b. Append event: { StepID: step.ID, Event: "step_failed", Data: error }
     c. Find transition with event "error".
        - If found: transition to error step.
        - If not found: set status = "suspended" (needs manual intervention).
     d. For "notification" type: treat errors as non-blocking.
        Generate "completed" event even on error (notifications are best-effort).
```

### System Step Chain Depth Limit

To prevent infinite loops, the engine limits the number of consecutive system
steps executed in a single request. Default limit: 10.

If the limit is reached, the workflow is suspended with an error:
```
{ "code": "WORKFLOW_CHAIN_LIMIT", "message": "Too many consecutive system steps" }
```

This typically indicates a misconfigured workflow with a cycle of system steps.

---

## Timeout Processing

### Workflow-Level Timeouts

```yaml
workflows:
  - id: "orders.approval"
    timeout: "72h"
    on_timeout: "expired"
```

When a workflow's `ExpiresAt` is in the past:

```
1. Background job calls Engine.ProcessTimeouts(ctx) periodically (every 1 minute).
2. WorkflowStore.FindExpired(ctx, now) → returns expired active instances.
3. For each expired instance:
   a. Check if current step has on_timeout defined:
      - If yes: treat as advance with event "timeout".
      - If no: check workflow-level on_timeout.
      - If neither: set status = "failed", append "timeout" event.
   b. Persist updated state.
   c. Log timeout with full context.
   d. Emit metric: workflow.timeout.
```

### Step-Level Timeouts

Individual steps can have their own timeouts:

```yaml
steps:
  - id: "review"
    timeout: "24h"
    on_timeout: "escalated"
```

When a step times out:
1. The timeout processor checks step entry time against step timeout.
2. If expired: advance with event "timeout" and transition to `on_timeout` step.
3. If no `on_timeout` for the step: fall back to workflow-level timeout handling.

---

## Workflow Cancellation

```
POST /ui/workflows/wf-abc/cancel
  { "reason": "Customer requested cancellation" }

Engine.Cancel(ctx, rctx, "wf-abc", "Customer requested cancellation"):

  1. Load instance from store.
  2. Verify tenant isolation.
  3. Verify status is "active" or "suspended".
     → 409 if already completed/cancelled/failed.
  4. Set status = "cancelled".
  5. Append event: { Event: "cancelled", ActorID: rctx.SubjectID, Comment: reason }
  6. Persist.
  7. Return success.
```

Cancellation does NOT undo any backend operations that were already executed.
If compensation is needed, it must be modeled as additional workflow steps
(saga pattern).

---

## WorkflowStore Interface

```
WorkflowStore
  ├── Create(ctx, instance WorkflowInstance) → error
  │     Inserts a new instance. Returns error if ID already exists.
  │
  ├── Get(ctx, tenantId string, instanceId string) → (WorkflowInstance, error)
  │     Returns the instance. Returns not-found error if wrong tenant.
  │
  ├── Update(ctx, instance WorkflowInstance) → error
  │     Updates the instance. Uses optimistic locking on Version.
  │     Returns conflict error if version mismatch.
  │
  ├── AppendEvent(ctx, event WorkflowEvent) → error
  │     Appends an event. Events are insert-only.
  │
  ├── GetEvents(ctx, tenantId string, instanceId string) → ([]WorkflowEvent, error)
  │     Returns all events for an instance, ordered by timestamp.
  │
  ├── FindActive(ctx, tenantId string, filters map[string]string) → ([]WorkflowInstance, error)
  │     Finds active instances matching filters.
  │     Filters: workflow_id, subject_id, status.
  │
  ├── FindExpired(ctx, cutoff time.Time) → ([]WorkflowInstance, error)
  │     Finds active instances where ExpiresAt < cutoff.
  │     Used by timeout processor. Scans across all tenants.
  │
  └── Delete(ctx, tenantId string, instanceId string) → error
        Hard deletes an instance and its events.
        Used for data retention cleanup only.
```

### PostgreSQL Schema

```sql
CREATE TABLE workflow_instances (
    id              UUID PRIMARY KEY,
    workflow_id     TEXT NOT NULL,
    tenant_id       TEXT NOT NULL,
    partition_id    TEXT NOT NULL,
    subject_id      TEXT NOT NULL,
    current_step    TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'active',
    state           JSONB NOT NULL DEFAULT '{}',
    version         INTEGER NOT NULL DEFAULT 1,
    idempotency_key TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ,

    CONSTRAINT unique_idempotency UNIQUE (tenant_id, idempotency_key)
);

CREATE INDEX idx_workflow_instances_tenant_status
    ON workflow_instances (tenant_id, status);

CREATE INDEX idx_workflow_instances_expires
    ON workflow_instances (expires_at)
    WHERE status = 'active';

CREATE TABLE workflow_events (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_instance_id  UUID NOT NULL REFERENCES workflow_instances(id),
    step_id               TEXT NOT NULL,
    event                 TEXT NOT NULL,
    actor_id              TEXT NOT NULL,
    data                  JSONB DEFAULT '{}',
    comment               TEXT DEFAULT '',
    timestamp             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflow_events_instance
    ON workflow_events (workflow_instance_id, timestamp);
```

### Optimistic Locking

```sql
UPDATE workflow_instances
SET current_step = $1, status = $2, state = $3, version = version + 1, updated_at = NOW()
WHERE id = $4 AND tenant_id = $5 AND version = $6;

-- If affected rows == 0: version conflict → retry
```

---

## Resumable Workflows

A key design property: workflows are always resumable.

### Scenario: User Leaves Mid-Workflow

```
1. User starts approval workflow → workflow enters "review" step.
2. User closes browser.
3. Later, user returns.
4. Frontend calls GET /ui/workflows?status=active → finds wf-abc at "review" step.
5. Frontend calls GET /ui/workflows/wf-abc → gets full descriptor with form.
6. User completes the review and advances the workflow.
```

### Scenario: BFF Restarts During System Step

```
1. Workflow is at "process" step (system step).
2. BFF crashes mid-execution.
3. BFF restarts.
4. Workflow is still at "process" step in the store (state was not updated yet).
5. Two options:
   a. User manually retries (advance with "retry" event if defined).
   b. Background job detects stuck workflow (active but not advanced for > expected duration).
      → Retries the system step.
```

### Scenario: Different User Advances Different Step

```
1. Alice starts approval workflow → enters "review" step (assignee: role "approver").
2. Bob (who has the "approver" role) calls GET /ui/workflows?status=active.
3. Bob sees the workflow in his task list.
4. Bob advances the "review" step with "approved".
5. Workflow proceeds. Audit trail shows Alice started, Bob approved.
```

---

## Testing Workflows

### Unit Testing the Engine

```go
func TestWorkflowEngine_StartAndAdvance(t *testing.T) {
    // Setup in-memory store, mock invoker, mock capability resolver
    engine := workflow.NewEngine(registry, store, invokerRegistry, capResolver)

    // Start workflow
    instance, err := engine.Start(ctx, rctx, "orders.approval", startInput)
    assert.NoError(t, err)
    assert.Equal(t, "review", instance.CurrentStep)
    assert.Equal(t, "active", instance.Status)

    // Advance with approval
    updated, err := engine.Advance(ctx, rctx, instance.ID, "approved", approvalInput)
    assert.NoError(t, err)
    // System steps auto-executed: process → notify → approved (terminal)
    assert.Equal(t, "approved", updated.CurrentStep)
    assert.Equal(t, "completed", updated.Status)

    // Verify events
    events, _ := store.GetEvents(ctx, rctx.TenantID, instance.ID)
    assert.Len(t, events, 7) // step_entered, step_completed, approved, step_entered(process), step_completed(process), step_entered(notify), step_completed(notify)...
}
```

### Testing Timeouts

```go
func TestWorkflowEngine_Timeout(t *testing.T) {
    // Start workflow
    instance := startWorkflow(t, engine, ctx, rctx)

    // Set expires_at to past
    instance.ExpiresAt = timeInPast()
    store.Update(ctx, instance)

    // Process timeouts
    engine.ProcessTimeouts(ctx)

    // Verify workflow transitioned to "expired" step
    updated, _ := store.Get(ctx, rctx.TenantID, instance.ID)
    assert.Equal(t, "expired", updated.CurrentStep)
}
```
