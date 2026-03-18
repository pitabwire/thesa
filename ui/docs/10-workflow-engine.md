# 10. Workflow Engine

## What Is a Workflow?

A workflow is a multi-step process that guides a user through a sequence of actions to achieve an outcome. Think of it like a wizard or a guided checkout:

- **Step 1**: Select which order to refund
- **Step 2**: Enter the reason for the refund
- **Step 3**: Review the details
- **Step 4**: Confirm and submit

Each step collects data, validates it, and determines what the next step should be. The workflow might branch: if the refund is over $1,000, an extra approval step is inserted.

In enterprise systems, workflows are everywhere:
- Employee onboarding (10+ steps)
- Purchase approvals (varies by amount)
- Insurance claims (varies by claim type)
- Shipping and fulfillment (varies by carrier)

---

## Why a Workflow Engine?

Without a workflow engine, developers would build each multi-step flow as a custom screen sequence. With 50 different workflows, that is 50 custom implementations — each with its own bugs, inconsistencies, and maintenance burden.

The Workflow Engine is generic. The BFF describes the workflow (steps, transitions, conditions), and the engine renders it. Adding a new workflow requires zero frontend code changes.

---

## How Workflows Are Defined

The BFF defines workflows and exposes them via:

- `GET /ui/workflows/{workflowId}` — returns the workflow definition
- `POST /ui/workflows/{workflowId}/step` — submits a step and gets the next one

### Example Workflow Definition

```json
{
  "workflowId": "refund-process",
  "title": "Process Refund",
  "steps": [
    {
      "stepId": "select-order",
      "title": "Select Order",
      "renderAs": "selection",
      "resource": "orders",
      "filters": { "status": ["delivered", "shipped"] },
      "selectionType": "single",
      "description": "Choose the order to refund"
    },
    {
      "stepId": "enter-reason",
      "title": "Refund Reason",
      "renderAs": "form",
      "schemaRef": "schemas/refund-reason",
      "description": "Explain why this refund is being issued"
    },
    {
      "stepId": "manager-approval",
      "title": "Manager Approval",
      "renderAs": "info",
      "content": "This refund exceeds $1,000 and requires manager approval. You will be notified when approved.",
      "condition": {
        "field": "select-order.total",
        "greaterThan": 1000
      },
      "waitFor": "external_approval",
      "pollingIntervalSeconds": 10
    },
    {
      "stepId": "review",
      "title": "Review",
      "renderAs": "review",
      "description": "Verify all details before submitting"
    },
    {
      "stepId": "confirm",
      "title": "Confirm",
      "renderAs": "confirmation",
      "confirmLabel": "Submit Refund",
      "cancelLabel": "Cancel",
      "destructive": false
    }
  ],
  "transitions": {
    "select-order": ["enter-reason"],
    "enter-reason": ["manager-approval", "review"],
    "manager-approval": ["review"],
    "review": ["confirm"],
    "confirm": []
  }
}
```

### What This Defines

- **Steps**: Five steps, each with a type and configuration
- **Transitions**: After each step, which step(s) could come next
- **Conditions**: The "manager-approval" step only appears if the order total exceeds $1,000
- **External wait**: The "manager-approval" step pauses the workflow until an external approval happens

---

## The Workflow State Machine

### What Is a State Machine?

A state machine is a programming pattern that tracks "what state am I in?" and "what states can I move to from here?" It prevents invalid transitions — you cannot jump from step 1 to step 5 without going through steps 2, 3, and 4.

### How It Works in Thesa UI

The `WorkflowStateMachine` is a class that:

1. **Holds the current state**:
   - Which step the user is on
   - All data collected from previous steps
   - The history of visited steps (for "Back" navigation)

2. **Evaluates transitions**:
   - When the user completes a step, the state machine checks the `transitions` map
   - If there are multiple possible next steps, it evaluates `condition` rules to determine which one applies
   - It then moves to the determined next step

3. **Persists to the database**:
   - After every step, the state machine writes its state to the `workflow_state` table in Drift
   - This means if the user closes the app and reopens it, they resume exactly where they left off

### Step-by-Step Execution Flow

```
User starts the "refund-process" workflow
        │
        ▼
State Machine initializes:
  currentStep = "select-order"
  stepData = {}
  history = []
        │
        ▼
WorkflowRenderer renders "select-order" as a selection page
  → User sees a list of orders, picks ORD-1234 (total: $1,500)
  → Clicks "Continue"
        │
        ▼
State Machine processes:
  1. Save step data: stepData["select-order"] = { orderId: "ORD-1234", total: 1500 }
  2. Push to history: history = ["select-order"]
  3. Check transitions: "select-order" → ["enter-reason"]
  4. Only one option → move to "enter-reason"
  5. currentStep = "enter-reason"
  6. Persist to Drift
        │
        ▼
WorkflowRenderer renders "enter-reason" as a form
  → User fills in reason: "Product defective"
  → Clicks "Continue"
        │
        ▼
State Machine processes:
  1. POST /ui/workflows/refund-process/step
     body: { stepId: "enter-reason", data: { reason: "Product defective" } }
  2. BFF validates the data → success
  3. Save step data: stepData["enter-reason"] = { reason: "Product defective" }
  4. Push to history: history = ["select-order", "enter-reason"]
  5. Check transitions: "enter-reason" → ["manager-approval", "review"]
  6. Evaluate conditions:
     - "manager-approval" has condition: select-order.total > 1000
     - 1500 > 1000 → TRUE → include "manager-approval"
  7. Move to "manager-approval"
  8. currentStep = "manager-approval"
  9. Persist to Drift
        │
        ▼
WorkflowRenderer renders "manager-approval" as an info page
  → "This refund exceeds $1,000 and requires manager approval."
  → Workflow begins polling: GET /ui/workflows/refund-process every 10 seconds
  → (User can close the app and come back later)
        │
        ▼
Eventually, manager approves (externally)
  → Polling detects: workflow status changed to "approved"
  → State Machine advances to "review"
        │
        ▼
WorkflowRenderer renders "review" as a read-only summary
  → Shows: Order ORD-1234, $1,500, Reason: "Product defective", Approved by: Manager Jane
  → User clicks "Continue"
        │
        ▼
WorkflowRenderer renders "confirm" as a confirmation dialog
  → "Submit this refund for $1,500?"
  → [Cancel] [Submit Refund]
  → User clicks "Submit Refund"
        │
        ▼
State Machine:
  1. POST /ui/workflows/refund-process/step
     body: { stepId: "confirm", data: { confirmed: true } }
  2. BFF responds: { completed: true, result: { refundId: "REF-567" } }
  3. State Machine marks workflow as completed
  4. WorkflowRenderer shows success: "Refund REF-567 has been submitted"
  5. Clean up: remove workflow state from Drift
```

---

## Step Rendering Types

Each workflow step declares a `renderAs` type that tells the Workflow Renderer how to display it:

### `form` — Data Collection Step

Renders a dynamic form (using the Form Engine from Section 8).

```
┌──────────────────────────────────────────────────┐
│ Step 2 of 5: Refund Reason                        │
├──────────────────────────────────────────────────┤
│                                                    │
│ Reason Category *                                  │
│ ┌──────────────────────────────────────────┐      │
│ │ Product Defective                      ▾ │      │
│ └──────────────────────────────────────────┘      │
│                                                    │
│ Description *                                      │
│ ┌──────────────────────────────────────────┐      │
│ │ The product arrived with a cracked       │      │
│ │ screen and does not power on.            │      │
│ │                                          │      │
│ └──────────────────────────────────────────┘      │
│ 87 / 1000 characters                               │
│                                                    │
│ Attach Photo (optional)                            │
│ ┌──────────────────────────────────────────┐      │
│ │ [📎 Choose file]                         │      │
│ └──────────────────────────────────────────┘      │
│                                                    │
├──────────────────────────────────────────────────┤
│                         [← Back]  [Continue →]    │
└──────────────────────────────────────────────────┘
```

The schema for the form comes from `schemaRef` in the step definition. Validation rules apply — the user cannot advance until the form is valid.

### `review` — Read-Only Summary

Renders all data collected so far in a read-only card layout. The user can go back to edit previous steps.

```
┌──────────────────────────────────────────────────┐
│ Step 4 of 5: Review                               │
├──────────────────────────────────────────────────┤
│                                                    │
│ Order Details                          [✏ Edit]   │
│ ┌──────────────────────────────────────────────┐  │
│ │ Order:     ORD-1234                          │  │
│ │ Customer:  Alice Smith                       │  │
│ │ Total:     $1,500.00                         │  │
│ │ Status:    Delivered                         │  │
│ └──────────────────────────────────────────────┘  │
│                                                    │
│ Refund Details                         [✏ Edit]   │
│ ┌──────────────────────────────────────────────┐  │
│ │ Category:  Product Defective                 │  │
│ │ Reason:    The product arrived with a        │  │
│ │            cracked screen and does not       │  │
│ │            power on.                         │  │
│ │ Photo:     cracked_screen.jpg                │  │
│ └──────────────────────────────────────────────┘  │
│                                                    │
│ Approval                                           │
│ ┌──────────────────────────────────────────────┐  │
│ │ Approved by: Manager Jane                    │  │
│ │ Approved at: Feb 6, 2026, 3:15 PM           │  │
│ └──────────────────────────────────────────────┘  │
│                                                    │
├──────────────────────────────────────────────────┤
│                         [← Back]  [Continue →]    │
└──────────────────────────────────────────────────┘
```

The [Edit] buttons navigate back to the corresponding step, allowing the user to change their input without restarting the entire workflow.

### `selection` — Item Selection Step

Renders a list or table of items from a BFF resource. The user selects one (or multiple, depending on `selectionType`).

```
┌──────────────────────────────────────────────────┐
│ Step 1 of 5: Select Order                         │
├──────────────────────────────────────────────────┤
│ 🔍 Search orders...                               │
├──────────────────────────────────────────────────┤
│                                                    │
│ ○ │ ORD-1234 │ Alice Smith │ $1,500.00 │ Delivered│
│ ● │ ORD-1235 │ Bob Jones   │ $89.00    │ Shipped  │  ← selected
│ ○ │ ORD-1236 │ Carol White │ $213.75   │ Delivered│
│                                                    │
├──────────────────────────────────────────────────┤
│                                    [Continue →]   │
└──────────────────────────────────────────────────┘
```

Uses radio buttons for `selectionType: "single"` and checkboxes for `selectionType: "multi"`.

### `confirmation` — Final Confirmation Step

A summary with prominent confirm/cancel buttons. For destructive actions, the confirm button is styled red.

```
┌──────────────────────────────────────────────────┐
│ Step 5 of 5: Confirm Refund                       │
├──────────────────────────────────────────────────┤
│                                                    │
│  ┌────────────────────────────────────────────┐   │
│  │  You are about to submit a refund of       │   │
│  │  $1,500.00 for order ORD-1234.             │   │
│  │                                            │   │
│  │  This will credit the customer's account   │   │
│  │  within 5-7 business days.                 │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
├──────────────────────────────────────────────────┤
│                  [Cancel]  [Submit Refund]         │
└──────────────────────────────────────────────────┘
```

### `info` — Informational / Waiting Step

Displays text content. May include a waiting state for external events (like approvals).

```
┌──────────────────────────────────────────────────┐
│ Step 3 of 5: Manager Approval                     │
├──────────────────────────────────────────────────┤
│                                                    │
│  ┌────────────────────────────────────────────┐   │
│  │  ⏳ Waiting for Approval                   │   │
│  │                                            │   │
│  │  This refund of $1,500.00 exceeds the      │   │
│  │  $1,000 threshold and requires manager     │   │
│  │  approval.                                 │   │
│  │                                            │   │
│  │  An approval request has been sent to      │   │
│  │  your manager. You will be notified when   │   │
│  │  it is approved.                           │   │
│  │                                            │   │
│  │  You can safely close this page. Your      │   │
│  │  progress is saved.                        │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
│  Last checked: 30 seconds ago                      │
│                                                    │
├──────────────────────────────────────────────────┤
│  [← Back]                                         │
└──────────────────────────────────────────────────┘
```

---

## Branching Workflows

### What Is Branching?

Branching means different users may follow different paths through the same workflow based on conditions. Think of it like a "Choose Your Own Adventure" book.

### How It Works

The `transitions` map can list multiple possible next steps:

```json
"transitions": {
  "enter-details": ["standard-review", "compliance-review", "express-approval"]
}
```

The state machine evaluates conditions on each possible next step:

```json
{
  "stepId": "compliance-review",
  "condition": { "field": "enter-details.amount", "greaterThan": 50000 }
},
{
  "stepId": "express-approval",
  "condition": { "field": "enter-details.type", "equals": "internal_transfer" }
},
{
  "stepId": "standard-review",
  "condition": null
}
```

Evaluation order:
1. Check "compliance-review" condition → if true, go there
2. Check "express-approval" condition → if true, go there
3. Fall through to "standard-review" (no condition = default path)

The first matching condition wins. The BFF controls the ordering.

### Visual Representation

```
                  enter-details
                      │
           ┌──────────┼──────────┐
           │          │          │
     amount > 50k   internal   default
           │        transfer    │
           ▼          │         ▼
    compliance-    ───▶   standard-
      review       express   review
           │       approval    │
           │          │        │
           └──────────┼────────┘
                      │
                   confirm
```

---

## Workflow Persistence and Resumability

### Why Persistence Matters

Enterprise workflows can take hours, days, or weeks to complete:
- An employee might start an expense report on Monday and finish it on Tuesday
- An approval workflow might wait days for a manager to approve
- A user might start a complex form, get interrupted, and return later

If the workflow state were only in memory, all progress would be lost when the app is closed.

### How Persistence Works

The `workflow_state` table in Drift stores:

| Column | What It Stores |
|---|---|
| `workflow_id` | Unique identifier for this workflow instance (e.g., "refund-process-abc123") |
| `current_step` | Which step the user is on (e.g., "enter-reason") |
| `step_data` | JSON containing all data from all completed steps |
| `started_at` | When the workflow was started |
| `updated_at` | When the workflow was last modified |
| `resumable` | Whether this workflow can be resumed (true until completed or cancelled) |

### Resume Flow

```
User opens the app
        │
        ▼
workflowProvider checks Drift for active workflows
        │
        ├── Found: workflow "refund-process-abc123"
        │   currentStep: "manager-approval"
        │   stepData: { selectOrder: {...}, enterReason: {...} }
        │
        ▼
User navigates to the workflow (or sees a notification: "You have an active refund in progress")
        │
        ▼
WorkflowRenderer resumes at "manager-approval" step
  → All previous data is intact
  → Polling resumes for external approval
```

### Abandoning a Workflow

If a user wants to cancel a workflow in progress:
1. They click "Cancel Workflow"
2. A confirmation dialog appears: "Cancel this refund? All progress will be lost."
3. On confirm: the state machine sends a cancellation to the BFF, and the `workflow_state` row is deleted from Drift
4. The BFF handles any cleanup (releasing held resources, etc.)

---

## Background Polling for External Events

### The Problem

Some workflow steps depend on actions taken by other people or systems:
- A manager must approve a request
- A payment must be processed
- An external system must complete a verification

The user cannot advance until the external event happens. But the UI must detect when it happens.

### The Solution: Polling

When a step has `"waitFor": "external_approval"`, the workflow engine starts a periodic poll:

```
Every N seconds (defined by pollingIntervalSeconds):
  1. Send: GET /ui/workflows/{workflowId}
  2. Check response: has the status changed?
     - If YES → advance to next step
     - If NO → keep polling
  3. Use ETag to minimize bandwidth
```

### Polling Behavior

| Scenario | Behavior |
|---|---|
| App is in foreground | Poll at the defined interval (e.g., every 10 seconds) |
| App is in background | Reduce poll frequency (e.g., every 60 seconds) or stop |
| App is closed | No polling. On next open, resume polling |
| Device is offline | Pause polling. Resume when online |
| Step is completed externally | BFF responds with new status → engine advances |

### User Notification

When the external event happens:
- If the user is viewing the workflow → the step automatically advances with a smooth transition
- If the user is on a different page → a notification/snackbar appears: "Your refund has been approved. [Continue]"
- If the user is not in the app → on next open, the workflow resumes at the new step

---

## Workflow Step Indicator (Stepper)

At the top of every workflow page, a visual stepper shows progress:

```
●─────────●─────────●─────────○─────────○
Select    Reason    Approval  Review    Confirm
Order                (current)
```

- **● (filled)**: Completed step — user can click to go back and review/edit
- **● (current, highlighted)**: The step the user is currently on
- **○ (empty)**: Future step — not yet reachable

On mobile, the stepper condenses to show just the current step with a count:

```
Step 3 of 5: Manager Approval
[══════════░░░░░░░░░] 60%
```

### Stepper and Branching

When the workflow has branches, the stepper adapts. Steps that were skipped (because their condition was not met) do not appear in the stepper. The user only sees the steps on their actual path.

---

## Error Handling in Workflows

### Step Submission Failure

If `POST /ui/workflows/{workflowId}/step` returns an error:

```json
{
  "error": "validation_failed",
  "fields": {
    "reason": "Reason must be at least 20 characters"
  }
}
```

The workflow stays on the current step and shows the errors inline (using the Form Engine's error display).

### Server Error During Transition

If the BFF is unreachable when the user clicks "Continue":

1. The state machine saves the step data locally (Drift)
2. Shows an error: "Could not submit. Your progress is saved."
3. Offers a "Retry" button
4. When the user retries (or the network returns), the submission is attempted again

### Workflow Expired on Server

If the user resumes a workflow that the server has already cancelled (e.g., the order was refunded by someone else):

1. The poll or step submission returns: `{ "error": "workflow_expired", "message": "This order has already been refunded." }`
2. The workflow engine shows the message and offers to close the workflow
3. The local workflow state is cleaned up

---

## Workflow Security

- The BFF validates every step transition — the client cannot skip steps
- The BFF validates all data at every step — client-side validation is a convenience, not a security measure
- Step `allowed` flags determine which steps the current user can interact with
- The client never computes workflow logic — it only renders what the BFF tells it
