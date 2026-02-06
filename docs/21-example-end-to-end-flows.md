# 21 — Example End-to-End Flows

This document walks through complete request flows for multiple scenarios, showing
every HTTP call, every BFF internal step, and every backend interaction. Use these
flows as reference implementations when building new domains or debugging
production issues.

---

## How to Read These Flows

Each flow shows:
1. **The actor** — who or what initiates the request.
2. **The HTTP request** — exact method, URL, headers, body.
3. **BFF internal steps** — numbered, in execution order.
4. **Backend calls** — what the BFF sends to backend services.
5. **The HTTP response** — what the frontend receives.

Notation:
- `→` = outbound call (frontend→BFF or BFF→backend)
- `←` = response
- `✓` = check passes
- `✗` = check fails (error path)

---

## Flow 1: Page Load with Data (Orders List)

**Actor:** Alice (role: `order_manager`, tenant: `acme-corp`, partition: `us-west`)

### Step 1: Navigation Request

```
→ GET /ui/navigation
  Headers:
    Authorization: Bearer eyJhbGci...  (Alice's JWT)
    Accept: application/json
    X-Partition-Id: us-west
    X-Correlation-Id: corr-001
```

**BFF Internal:**

```
1.  Middleware: Parse JWT
      Extract subject_id: "user-alice"
      Extract tenant_id: "acme-corp"
      Extract roles: ["order_manager"]
      Extract email: "alice@acme-corp.com"
      Verify: signature ✓, expiry ✓, issuer ✓, audience ✓

2.  Middleware: Build RequestContext
      SubjectID: "user-alice"
      TenantID: "acme-corp"
      PartitionID: "us-west" (from X-Partition-Id, validated against tenant)
      Roles: ["order_manager"]
      CorrelationID: "corr-001"
      SessionID: (from JWT claim or generated)
      Locale: "en-US" (from Accept-Language or JWT claim)

3.  Middleware: Inject into context.Context

4.  Handler: NavigationHandler.GetNavigation()

5.  Resolve capabilities:
      CapabilityResolver.Resolve(RequestContext)
        → Cache MISS (first request)
        → PolicyEvaluator.ResolveCapabilities(RequestContext)
          → Static policy lookup: role "order_manager" →
            {
              "orders:nav:view",
              "orders:list:view",
              "orders:detail:view",
              "orders:detail:edit",
              "orders:edit:execute",
              "orders:cancel:execute",
              "orders:line_items:view",
              "orders:notes:view",
              "orders:notes:edit",
              "orders:search:execute",
              "inventory:nav:view",
              "inventory:list:view"
            }
        → Cache SET (key: "user-alice:acme-corp:us-west", TTL: 5m)

6.  MenuProvider.GetMenu(ctx, rctx, caps)
      Iterate all DomainDefinitions:

      Domain "orders":
        Check capability "orders:nav:view" → ✓
        Navigation entry: { id: "orders", label: "Orders", icon: "shopping_cart", order: 1 }
        Children:
          "orders.list": check "orders:list:view" → ✓ → include
          "orders.create": check "orders:create:view" → ✗ (not in cap set) → OMIT

      Domain "inventory":
        Check capability "inventory:nav:view" → ✓
        Navigation entry: { id: "inventory", label: "Inventory", icon: "inventory", order: 2 }
        Children:
          "inventory.list": check "inventory:list:view" → ✓ → include

      Domain "customers":
        Check capability "customers:nav:view" → ✗ → OMIT entire domain

7.  Badge resolution (optional, if configured):
      For "orders.list" badge:
        Invoke backend: GET /api/v1/orders/count?status=pending
        → Badge: { count: 12, style: "warning" }
      (Failures here are non-fatal; badge is omitted if backend is unreachable)

8.  Sort navigation items by order field
```

```
← 200 OK
  Content-Type: application/json
  Cache-Control: no-store

  {
    "items": [
      {
        "id": "orders",
        "label": "Orders",
        "icon": "shopping_cart",
        "children": [
          {
            "id": "orders.list",
            "label": "All Orders",
            "icon": "list",
            "route": "/orders",
            "badge": { "count": 12, "style": "warning" }
          }
        ]
      },
      {
        "id": "inventory",
        "label": "Inventory",
        "icon": "inventory",
        "children": [
          {
            "id": "inventory.list",
            "label": "Stock Levels",
            "icon": "list",
            "route": "/inventory"
          }
        ]
      }
    ]
  }
```

### Step 2: Page Descriptor Request

```
→ GET /ui/pages/orders.list
  Headers:
    Authorization: Bearer eyJhbGci...
    X-Partition-Id: us-west
    X-Correlation-Id: corr-002
```

**BFF Internal:**

```
1.  Middleware: JWT + RequestContext (as above, reuses cached capabilities)

2.  Handler: PageHandler.GetPage("orders.list")

3.  DefinitionRegistry.GetPage("orders.list") → PageDefinition ✓

4.  Check page-level capability: "orders:list:view" → ✓

5.  Resolve table descriptor:
      Columns (iterate definition columns):
        "order_number": check cap "orders:list:view" → ✓ → include
        "customer_name": check cap "orders:list:view" → ✓ → include
        "status": check cap "orders:list:view" → ✓ → include
        "total_amount": check cap "orders:list:view" → ✓ → include
        "created_at": check cap "orders:list:view" → ✓ → include

      Filters:
        "status": type "select" → include
        "date_range": type "date_range" → include
        "priority": type "select" → include

      Row actions:
        "Edit": check cap "orders:detail:edit" → ✓ → include
        "Cancel": check cap "orders:cancel:execute" → ✓ → include
        "Approve": check cap "orders:approve:execute" → ✗ → OMIT
            (order_manager doesn't have approve capability in this example)

      Bulk actions:
        "Export": check cap "orders:export:execute" → ✗ → OMIT

6.  Resolve page-level actions:
      "Create Order": check cap "orders:create:view" → ✗ → OMIT

7.  Build data endpoint URL:
      data_endpoint: "/ui/pages/orders.list/data"

8.  Strip internal metadata (operation IDs, backend field names, capability strings)
```

```
← 200 OK
  {
    "id": "orders.list",
    "title": "Orders",
    "layout": "table",
    "table": {
      "columns": [
        { "id": "order_number", "label": "Order #", "type": "text", "sortable": true },
        { "id": "customer_name", "label": "Customer", "type": "text", "sortable": true },
        { "id": "status", "label": "Status", "type": "badge", "sortable": true },
        { "id": "total_amount", "label": "Total", "type": "currency", "sortable": true },
        { "id": "created_at", "label": "Date", "type": "datetime", "sortable": true }
      ],
      "filters": [
        { "id": "status", "label": "Status", "type": "select",
          "options_endpoint": "/ui/lookups/orders.statuses" },
        { "id": "date_range", "label": "Date Range", "type": "date_range" },
        { "id": "priority", "label": "Priority", "type": "select",
          "options": [
            { "value": "normal", "label": "Normal" },
            { "value": "high", "label": "High" },
            { "value": "urgent", "label": "Urgent" }
          ]
        }
      ],
      "row_actions": [
        { "id": "edit", "label": "Edit", "icon": "edit", "type": "navigate",
          "navigate_to": "/orders/{id}/edit" },
        { "id": "cancel", "label": "Cancel", "icon": "cancel", "type": "command",
          "command_id": "orders.cancel", "style": "danger",
          "confirmation": { "title": "Cancel Order?",
                            "message": "This action cannot be undone." } }
      ],
      "data_endpoint": "/ui/pages/orders.list/data",
      "page_size": 25,
      "default_sort": "created_at",
      "default_sort_dir": "desc"
    },
    "actions": []
  }
```

### Step 3: Data Request

```
→ GET /ui/pages/orders.list/data?page=1&page_size=25&sort=created_at&sort_dir=desc&status=pending
  Headers:
    Authorization: Bearer eyJhbGci...
    X-Partition-Id: us-west
    X-Correlation-Id: corr-003
```

**BFF Internal:**

```
1.  Middleware: JWT + RequestContext + Capabilities (cached)

2.  Handler: PageHandler.GetPageData("orders.list", params)

3.  DefinitionRegistry.GetPage("orders.list") → PageDefinition
    Extract DataSourceDefinition:
      operation_id: "listOrders"
      service_id: "orders-svc"
      mapping:
        items_path: "data.orders"
        total_path: "meta.total"
        field_map: { order_number: "orderNumber", customer_name: "customerName", ... }

4.  Check capability "orders:list:view" → ✓

5.  Build backend request:
      a. Resolve operation: OpenAPIIndex.GetOperation("orders-svc", "listOrders")
         → GET /api/v1/orders

      b. Map pagination (service config: style "offset"):
         page=1, page_size=25 → offset=0, limit=25

      c. Map sort:
         sort=created_at → sort_by=createdAt (via field_map)
         sort_dir=desc → order=desc

      d. Map filters:
         status=pending → status=pending (direct mapping)

      e. Build InvocationInput:
         PathParams: {}
         QueryParams: { "offset": "0", "limit": "25",
                        "sort_by": "createdAt", "order": "desc",
                        "status": "pending" }
         Headers: {}
         Body: nil

6.  Invoke backend:
      → GET https://orders.internal/api/v1/orders?offset=0&limit=25&sort_by=createdAt&order=desc&status=pending
        Headers:
          Authorization: Bearer eyJhbGci... (forwarded)
          X-Tenant-Id: acme-corp
          X-Partition-Id: us-west
          X-Correlation-Id: corr-003
          traceparent: 00-{traceId}-{spanId}-01
          Accept: application/json

      Circuit breaker: orders-svc → CLOSED (healthy) ✓
      Timeout: 10s

      ← 200 OK
        {
          "data": {
            "orders": [
              {
                "id": "ord-123",
                "orderNumber": "ORD-2024-001",
                "customerName": "Bob Smith",
                "status": "pending",
                "totalAmount": 299.99,
                "currency": "USD",
                "createdAt": "2025-01-14T08:30:00Z"
              },
              ...
            ]
          },
          "meta": {
            "total": 47,
            "offset": 0,
            "limit": 25
          }
        }

7.  Apply response mapping:
      a. Extract items: JSONPath "data.orders" → array of 25 items
      b. Extract total: JSONPath "meta.total" → 47
      c. Apply field_map to each item:
         "orderNumber" → "order_number"
         "customerName" → "customer_name"
         "totalAmount" → "total_amount"
         "createdAt" → "created_at"
      d. Compute pagination metadata:
         total_items: 47, page: 1, page_size: 25, total_pages: 2
```

```
← 200 OK
  {
    "items": [
      {
        "id": "ord-123",
        "order_number": "ORD-2024-001",
        "customer_name": "Bob Smith",
        "status": "pending",
        "total_amount": 299.99,
        "currency": "USD",
        "created_at": "2025-01-14T08:30:00Z"
      },
      ...
    ],
    "pagination": {
      "page": 1,
      "page_size": 25,
      "total_items": 47,
      "total_pages": 2
    }
  }
```

---

## Flow 2: Command Execution (Cancel Order)

**Actor:** Alice (role: `order_manager`, tenant: `acme-corp`)

Alice clicks "Cancel" on order ORD-2024-001 and confirms the dialog.

```
→ POST /ui/commands/orders.cancel
  Headers:
    Authorization: Bearer eyJhbGci...
    Content-Type: application/json
    X-Partition-Id: us-west
    X-Correlation-Id: corr-010
    X-Idempotency-Key: idem-abc-123

  Body:
  {
    "input": {
      "reason": "Customer requested cancellation",
      "refund_type": "full"
    },
    "route_params": {
      "id": "ord-123"
    },
    "idempotency_key": "idem-abc-123"
  }
```

**BFF Internal:**

```
1.  Middleware: JWT + RequestContext + Capabilities (cached)

2.  Handler: CommandHandler.Execute("orders.cancel", commandInput)

3.  DefinitionRegistry.GetCommand("orders.cancel") → CommandDefinition
      operation:
        type: "openapi"
        service_id: "orders-svc"
        operation_id: "cancelOrder"
      capabilities: ["orders:cancel:execute"]
      input_mapping:
        path_params:
          orderId: "route.id"
        body_mapping: "projection"
        field_projection:
          cancellationReason: "input.reason"
          refundType: "input.refund_type"
      idempotency:
        enabled: true
        ttl: 24h
      error_map:
        ORDER_ALREADY_CANCELLED: "This order has already been cancelled."
        ORDER_SHIPPED: "Cannot cancel an order that has already shipped."

4.  Check capability "orders:cancel:execute" → ✓

5.  Check idempotency:
      Key: "idem-abc-123"
      Store lookup → NOT FOUND (first attempt) → proceed

6.  Apply input mapping:
      a. Path params:
         orderId = route_params["id"] = "ord-123"

      b. Body (projection strategy):
         {
           "cancellationReason": input["reason"] → "Customer requested cancellation",
           "refundType": input["refund_type"] → "full"
         }

      c. Headers: none additional

      Constructed InvocationInput:
        PathParams: { "orderId": "ord-123" }
        QueryParams: {}
        Headers: {}
        Body: { "cancellationReason": "Customer requested cancellation", "refundType": "full" }

7.  Schema validation:
      OpenAPIIndex.ValidateRequest("orders-svc", "cancelOrder", body)
      → Schema says: cancellationReason (string, required), refundType (enum: "full"|"partial")
      → Validation: ✓ (all fields valid)

8.  Invoke backend:
      Resolve operation: POST /api/v1/orders/{orderId}/cancel
      URL: https://orders.internal/api/v1/orders/ord-123/cancel

      → POST https://orders.internal/api/v1/orders/ord-123/cancel
        Headers:
          Authorization: Bearer eyJhbGci... (forwarded)
          Content-Type: application/json
          X-Tenant-Id: acme-corp
          X-Partition-Id: us-west
          X-Correlation-Id: corr-010
          traceparent: 00-{traceId}-{spanId}-01

        Body: { "cancellationReason": "Customer requested cancellation", "refundType": "full" }

      Circuit breaker: orders-svc → CLOSED ✓
      Timeout: 10s

      ← 200 OK
        {
          "id": "ord-123",
          "status": "cancelled",
          "cancelledAt": "2025-01-15T10:30:00Z",
          "cancelledBy": "user-alice",
          "refundAmount": 299.99,
          "refundStatus": "pending"
        }

9.  Handle success (2xx):
      a. Apply output_mapping (if defined):
         Project fields: { "refund_amount": "refundAmount", "refund_status": "refundStatus" }

      b. Store idempotency:
         Key: "idem-abc-123"
         Value: CommandResponse (serialized)
         TTL: 24h

      c. Build CommandResponse:
         {
           "success": true,
           "message": "Order cancelled successfully",
           "result": {
             "refund_amount": 299.99,
             "refund_status": "pending"
           }
         }

10. Emit metrics:
      thesa_command_executions_total{command_id="orders.cancel", status="success"} +1
      thesa_command_duration_seconds{command_id="orders.cancel"} = 0.145

11. Audit log:
      {
        "type": "audit",
        "event": "command.executed",
        "command_id": "orders.cancel",
        "subject_id": "user-alice",
        "tenant_id": "acme-corp",
        "resource_id": "ord-123",
        "success": true,
        "correlation_id": "corr-010"
      }
```

```
← 200 OK
  {
    "success": true,
    "message": "Order cancelled successfully",
    "result": {
      "refund_amount": 299.99,
      "refund_status": "pending"
    }
  }
```

### Idempotency Replay (Duplicate Request)

If the frontend retries the same request (network retry, user double-click):

```
→ POST /ui/commands/orders.cancel
  X-Idempotency-Key: idem-abc-123
  Body: (same as above)

BFF Internal:
  1. Check idempotency:
     Key: "idem-abc-123"
     Store lookup → FOUND
     Compare input hash → matches ✓
     Return cached result (no backend call)

← 200 OK
  (same response as original)
```

If the key matches but the input differs:

```
→ POST /ui/commands/orders.cancel
  X-Idempotency-Key: idem-abc-123
  Body: { "input": { "reason": "DIFFERENT REASON", ... } }

BFF Internal:
  1. Check idempotency:
     Key: "idem-abc-123"
     Store lookup → FOUND
     Compare input hash → MISMATCH ✗

← 409 Conflict
  {
    "error": {
      "code": "CONFLICT",
      "message": "Idempotency key already used with different input"
    }
  }
```

---

## Flow 3: Command Execution — Validation Failure

**Actor:** Alice submits an order update with invalid data.

```
→ POST /ui/commands/orders.update
  Body:
  {
    "input": {
      "priority": "super-urgent",
      "shipping_address": ""
    },
    "route_params": { "id": "ord-123" }
  }
```

**BFF Internal:**

```
1.  Middleware: JWT + RequestContext + Capabilities

2.  Resolve CommandDefinition "orders.update"

3.  Check capability "orders:edit:execute" → ✓

4.  Apply input mapping → construct body:
      {
        "priority": "super-urgent",
        "shippingAddress": ""
      }

5.  Schema validation against OpenAPI:
      OpenAPIIndex.ValidateRequest("orders-svc", "updateOrder", body)
      → Failures:
        - "priority": value "super-urgent" not in enum ["normal", "high", "urgent"]
        - "shippingAddress": minLength 1 violated (empty string)

6.  Map validation errors back to UI field names (reverse field_map):
      "priority" → "priority" (same name)
      "shippingAddress" → "shipping_address"

7.  Short-circuit — do NOT invoke backend
```

```
← 422 Unprocessable Entity
  {
    "error": {
      "code": "VALIDATION_ERROR",
      "message": "One or more fields are invalid",
      "details": [
        {
          "field": "priority",
          "code": "INVALID_VALUE",
          "message": "Priority must be one of: normal, high, urgent"
        },
        {
          "field": "shipping_address",
          "code": "REQUIRED",
          "message": "Shipping address is required"
        }
      ],
      "trace_id": "trace-xyz"
    }
  }
```

---

## Flow 4: Command Execution — Backend Error Translation

**Actor:** Alice tries to cancel an already-shipped order.

```
→ POST /ui/commands/orders.cancel
  Body:
  {
    "input": { "reason": "Changed my mind", "refund_type": "full" },
    "route_params": { "id": "ord-456" }
  }
```

**BFF Internal:**

```
1.  Middleware → RequestContext → Capabilities → all checks pass

2.  Apply input mapping → construct body

3.  Schema validation → ✓

4.  Invoke backend:
      → POST https://orders.internal/api/v1/orders/ord-456/cancel
      ← 422 Unprocessable Entity
        {
          "error": {
            "code": "ORDER_SHIPPED",
            "message": "Cannot cancel order in 'shipped' state"
          }
        }

5.  Handle client error (4xx):
      a. Extract error code: "ORDER_SHIPPED"
      b. Look up in command's error_map:
         "ORDER_SHIPPED" → "Cannot cancel an order that has already shipped."
      c. Build error response with translated message

6.  Do NOT store in idempotency cache (failure)
```

```
← 422 Unprocessable Entity
  {
    "error": {
      "code": "ORDER_SHIPPED",
      "message": "Cannot cancel an order that has already shipped.",
      "trace_id": "trace-abc"
    }
  }
```

If the backend returns an unmapped error code:

```
Backend: 422 { "error": { "code": "INTERNAL_ORDER_STATE_MACHINE_ERROR", "message": "..." } }

BFF: error code not in error_map → use generic message
  {
    "error": {
      "code": "COMMAND_FAILED",
      "message": "The operation could not be completed. Please try again or contact support.",
      "trace_id": "trace-def"
    }
  }
```

---

## Flow 5: Workflow — Full Order Approval Lifecycle

**Actors:**
- Bob (role: `order_clerk`, tenant: `acme-corp`) — starts the workflow
- Alice (role: `order_approver`, tenant: `acme-corp`) — approves
- System — processes the approval and sends notification

### Phase 1: Bob Starts the Workflow

```
→ POST /ui/workflows/orders.approval/start
  Authorization: Bearer (Bob's token)
  Body:
  {
    "order_id": "ord-789",
    "order_total": 15000.00,
    "customer_email": "charlie@example.com"
  }
```

**BFF Internal:**

```
1.  Resolve capabilities for Bob:
      PolicyEvaluator → roles ["order_clerk"] →
        { "orders:nav:view", "orders:list:view", "orders:detail:view",
          "orders:approve:start", ... }

2.  DefinitionRegistry.GetWorkflow("orders.approval") → WorkflowDefinition
      initial_step: "review"
      capability: "orders:approve:start"
      timeout: 72h

3.  Check capability "orders:approve:start" → ✓

4.  Create WorkflowInstance:
      ID: "wf-uuid-001" (UUID v4)
      WorkflowID: "orders.approval"
      TenantID: "acme-corp"
      PartitionID: "us-west"
      SubjectID: "user-bob"
      CurrentStep: "review"
      Status: "active"
      State: {
        "order_id": "ord-789",
        "order_total": 15000.00,
        "customer_email": "charlie@example.com"
      }
      CreatedAt: now
      UpdatedAt: now
      ExpiresAt: now + 72h

5.  WorkflowStore.Create(ctx, instance) → ✓

6.  Append event:
      WorkflowStore.AppendEvent(ctx, {
        ID: "evt-001",
        WorkflowInstanceID: "wf-uuid-001",
        StepID: "review",
        Event: "step_entered",
        ActorID: "user-bob",
        Data: { "order_id": "ord-789" },
        Timestamp: now
      })

7.  Step "review" is type "action" (requires user input) → do NOT auto-advance

8.  Build WorkflowDescriptor:
      Resolve current step "review":
        form_id: "orders.approval_form" → resolve FormDescriptor
        actions: [
          { id: "approve", label: "Approve", event: "approved",
            capability: "orders:approve:execute" },
          { id: "reject", label: "Reject", event: "rejected",
            capability: "orders:approve:execute" }
        ]
      Filter actions by Bob's capabilities:
        "orders:approve:execute" → ✗ (Bob is clerk, not approver)
        → Both approve and reject actions are OMITTED for Bob

9.  Emit: thesa_workflow_starts_total{workflow_id="orders.approval"} +1
         thesa_workflow_active_instances{workflow_id="orders.approval"} = +1
```

```
← 201 Created
  {
    "id": "wf-uuid-001",
    "workflow_id": "orders.approval",
    "name": "Order Approval",
    "status": "active",
    "current_step": {
      "id": "review",
      "name": "Review Order",
      "type": "action",
      "status": "pending",
      "form": { ... },
      "actions": []
    },
    "steps": [
      { "id": "review", "name": "Review Order", "status": "pending" },
      { "id": "process", "name": "Process Order", "status": "future" },
      { "id": "notify", "name": "Send Notification", "status": "future" },
      { "id": "approved", "name": "Approved", "status": "future" },
      { "id": "rejected", "name": "Rejected", "status": "future" }
    ],
    "history": [
      {
        "step_id": "review",
        "event": "step_entered",
        "actor": "user-bob",
        "timestamp": "2025-01-15T09:00:00Z"
      }
    ]
  }
```

**Note:** Bob sees the workflow was created and is at "review" step, but cannot
advance it because he lacks `orders:approve:execute`. The UI will show the
workflow in a "waiting for approval" state.

### Phase 2: Alice Views and Approves

Alice navigates to the workflow list or is notified of a pending approval.

```
→ GET /ui/workflows/wf-uuid-001
  Authorization: Bearer (Alice's token)
```

**BFF Internal:**

```
1.  Resolve capabilities for Alice:
      roles ["order_approver"] →
        { "orders:approve:execute", "orders:detail:view", ... }

2.  WorkflowStore.Get(ctx, "acme-corp", "wf-uuid-001") → instance ✓
      Verify tenant: instance.TenantID ("acme-corp") == rctx.TenantID ("acme-corp") ✓

3.  Build WorkflowDescriptor:
      Current step "review" actions filtered by Alice's capabilities:
        "orders:approve:execute" → ✓
        → Approve and Reject actions ARE included for Alice
```

```
← 200 OK
  {
    "id": "wf-uuid-001",
    ...
    "current_step": {
      "id": "review",
      "name": "Review Order",
      "type": "action",
      "status": "pending",
      "form": {
        "sections": [
          {
            "title": "Approval",
            "fields": [
              { "id": "approval_notes", "label": "Notes", "type": "textarea", "required": false }
            ]
          }
        ]
      },
      "actions": [
        { "id": "approve", "label": "Approve", "icon": "check", "style": "primary",
          "event": "approved" },
        { "id": "reject", "label": "Reject", "icon": "close", "style": "danger",
          "event": "rejected" }
      ]
    },
    ...
  }
```

Alice clicks "Approve":

```
→ POST /ui/workflows/wf-uuid-001/advance
  Authorization: Bearer (Alice's token)
  Body:
  {
    "event": "approved",
    "input": {
      "approval_notes": "Verified with warehouse. Stock available."
    }
  }
```

**BFF Internal:**

```
1.   Capabilities for Alice → includes "orders:approve:execute" ✓

2.   WorkflowStore.Get(ctx, "acme-corp", "wf-uuid-001") → instance
       CurrentStep: "review", Status: "active" ✓

3.   Tenant check: ✓

4.   Status check: "active" ✓

5.   Step definition for "review":
       type: "action"
       capability: "orders:approve:execute"
       transitions:
         - event: "approved" → target: "process"
         - event: "rejected" → target: "rejected"
         - on_timeout → target: "expired"

6.   Step capability check: "orders:approve:execute" → ✓

7.   Transition validation: event "approved" from step "review"
       → Found: target = "process" ✓

8.   Merge input into state:
       State += { "approval_notes": "Verified with warehouse. Stock available.",
                  "approved_by": "user-alice" }

9.   Append event: "step_completed" for "review"
       { StepID: "review", Event: "step_completed", ActorID: "user-alice",
         Data: { approval_notes: "..." } }

10.  Transition to "process":
       Update CurrentStep = "process"
       Append event: "step_entered" for "process"

11.  Step "process" is type "system" → auto-execute:

       a. Resolve operation binding:
            service_id: "orders-svc"
            operation_id: "confirmOrder"

       b. Apply input mapping from workflow state:
            path_params: { orderId: state["order_id"] } → { orderId: "ord-789" }
            body: {
              approvedBy: state["approved_by"] → "user-alice",
              approvalNotes: state["approval_notes"] → "Verified with warehouse..."
            }

       c. Invoke backend:
            → POST https://orders.internal/api/v1/orders/ord-789/confirm
              Body: { "approvedBy": "user-alice", "approvalNotes": "Verified with warehouse..." }
            ← 200 OK { "status": "confirmed", "confirmedAt": "2025-01-15T10:31:00Z" }

       d. Merge result into state:
            State += { "confirmed_at": "2025-01-15T10:31:00Z" }

       e. Append "step_completed" for "process"

12.  Auto-advance from "process" → "notify":
       Transition: event "completed" → target "notify"
       Update CurrentStep = "notify"
       Append "step_entered" for "notify"

13.  Step "notify" is type "notification" → auto-execute:

       a. Resolve SDK handler: "notifications.SendOrderApproved"

       b. Invoke:
            SDKOperationInvoker.Invoke(ctx, rctx, binding, {
              Body: {
                orderId: "ord-789",
                customerEmail: "charlie@example.com",
                orderTotal: 15000.00
              }
            })
            ← OK (fire-and-forget; failure is non-blocking)

       c. Append "step_completed" for "notify"

14.  Auto-advance from "notify" → "approved":
       Transition: event "completed" → target "approved"
       Update CurrentStep = "approved"
       Append "step_entered" for "approved"

15.  Step "approved" is type "terminal":
       Set Status = "completed"
       Append "workflow_completed" event

16.  Persist final state (optimistic locking):
       WorkflowStore.Update(ctx, instance)
         → UPDATE workflow_instances SET ... WHERE id = 'wf-uuid-001' AND version = 1
         → version now = 2

17.  Metrics:
       thesa_workflow_advances_total{workflow_id="orders.approval", step_id="review", event="approved"} +1
       thesa_workflow_completions_total{workflow_id="orders.approval", final_status="completed"} +1
       thesa_workflow_active_instances{workflow_id="orders.approval"} = -1
```

```
← 200 OK
  {
    "id": "wf-uuid-001",
    "workflow_id": "orders.approval",
    "name": "Order Approval",
    "status": "completed",
    "current_step": {
      "id": "approved",
      "name": "Approved",
      "type": "terminal",
      "status": "completed"
    },
    "steps": [
      { "id": "review", "name": "Review Order", "status": "completed" },
      { "id": "process", "name": "Process Order", "status": "completed" },
      { "id": "notify", "name": "Send Notification", "status": "completed" },
      { "id": "approved", "name": "Approved", "status": "completed" }
    ],
    "history": [
      { "step_id": "review", "event": "step_entered", "actor": "user-bob",
        "timestamp": "2025-01-15T09:00:00Z" },
      { "step_id": "review", "event": "step_completed", "actor": "user-alice",
        "timestamp": "2025-01-15T10:30:00Z",
        "data": { "approval_notes": "Verified with warehouse. Stock available." } },
      { "step_id": "process", "event": "step_completed", "actor": "system",
        "timestamp": "2025-01-15T10:31:00Z" },
      { "step_id": "notify", "event": "step_completed", "actor": "system",
        "timestamp": "2025-01-15T10:31:01Z" },
      { "event": "workflow_completed", "actor": "system",
        "timestamp": "2025-01-15T10:31:01Z" }
    ]
  }
```

---

## Flow 6: Workflow — Timeout Handling

**Scenario:** A workflow at the "review" step is not acted on within 72 hours.

### Background Timeout Processor

The BFF runs a background goroutine (or cron-triggered handler) that periodically
checks for expired workflows.

```
Background task runs every 60 seconds:

1.  WorkflowStore.FindExpired(ctx, time.Now())
      → SQL: SELECT * FROM workflow_instances
             WHERE status = 'active' AND expires_at < NOW()
      → Returns: [ instance wf-uuid-002 ]

2.  For each expired instance:

    a. Load WorkflowDefinition

    b. Find current step's timeout transition:
       Step "review" has: on_timeout → target: "expired"

    c. Auto-advance with "timeout" event:
       - Append "step_completed" event with event: "timeout"
       - Transition to "expired" step

    d. Step "expired" is type "terminal":
       - Set Status = "completed" (or "failed" depending on definition)
       - Append "workflow_completed" event

    e. Persist

    f. Metrics:
       thesa_workflow_timeouts_total{workflow_id="orders.approval"} +1

    g. Audit log:
       { "event": "workflow.timeout", "instance_id": "wf-uuid-002",
         "step_id": "review", "tenant_id": "acme-corp" }
```

If a step has NO timeout transition defined:

```
    b. No on_timeout transition found
    c. Set Status = "failed"
    d. Append event: { event: "timeout", data: { reason: "No timeout handler defined" } }
    e. Persist
```

---

## Flow 7: Global Search

**Actor:** Alice searches for "ORD-2024"

```
→ GET /ui/search?q=ORD-2024&page=1&page_size=20
  Authorization: Bearer (Alice's token)
  X-Correlation-Id: corr-020
```

**BFF Internal:**

```
1.  Capabilities for Alice:
      { "orders:search:execute", "inventory:search:execute" }

2.  All SearchDefinitions:
      "orders.search" → requires "orders:search:execute" → ✓
      "customers.search" → requires "customers:search:execute" → ✗ → SKIP
      "inventory.search" → requires "inventory:search:execute" → ✓

3.  Execute eligible providers IN PARALLEL (goroutines + WaitGroup):

    Provider: orders.search
    ─────────────────────────
      a. Input mapping: q → searchQuery
      b. Invoke: GET https://orders.internal/api/v1/orders/search?searchQuery=ORD-2024&limit=10
         Headers: Authorization, X-Tenant-Id, X-Correlation-Id, traceparent
      c. Response:
         {
           "results": [
             { "id": "ord-789", "orderNumber": "ORD-2024-001", "customerName": "Bob Smith",
               "status": "confirmed", "score": 0.95 },
             { "id": "ord-790", "orderNumber": "ORD-2024-002", "customerName": "Jane Doe",
               "status": "pending", "score": 0.90 }
           ]
         }
      d. Apply result mapping:
         title: orderNumber → "ORD-2024-001"
         subtitle: customerName → "Bob Smith"
         badge: status → "confirmed"
         route: "/orders/{id}" → "/orders/ord-789"
         domain_weight: 1.0
      e. Score: backend_score * domain_weight = 0.95 * 1.0 = 0.95

    Provider: inventory.search
    ──────────────────────────
      a. Input mapping: q → query
      b. Invoke: GET https://inventory.internal/api/v1/items/search?query=ORD-2024&limit=10
      c. Response: { "results": [] }  (no matches)
      d. Result: empty

    Provider timeout: 3s per provider
    If a provider times out, its results are omitted. The response includes
    a "providers_failed" count.

4.  Merge results from all providers:
      [
        { title: "ORD-2024-001", subtitle: "Bob Smith", badge: "confirmed",
          route: "/orders/ord-789", domain: "orders", score: 0.95 },
        { title: "ORD-2024-002", subtitle: "Jane Doe", badge: "pending",
          route: "/orders/ord-790", domain: "orders", score: 0.90 }
      ]

5.  Deduplicate (by route + domain):
      No duplicates in this case.

6.  Sort by score descending:
      (already sorted)

7.  Apply pagination (page=1, page_size=20):
      Page 1 of 1, 2 results total
```

```
← 200 OK
  {
    "results": [
      {
        "title": "ORD-2024-001",
        "subtitle": "Bob Smith",
        "badge": { "text": "confirmed", "style": "success" },
        "route": "/orders/ord-789",
        "domain": "orders",
        "domain_label": "Orders",
        "icon": "shopping_cart"
      },
      {
        "title": "ORD-2024-002",
        "subtitle": "Jane Doe",
        "badge": { "text": "pending", "style": "warning" },
        "route": "/orders/ord-790",
        "domain": "orders",
        "domain_label": "Orders",
        "icon": "shopping_cart"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total_items": 2,
      "total_pages": 1
    },
    "metadata": {
      "providers_queried": 2,
      "providers_responded": 2,
      "providers_failed": 0,
      "query_time_ms": 87
    }
  }
```

---

## Flow 8: Form Load with Pre-populated Data

**Actor:** Alice clicks "Edit" on order ORD-2024-001.

### Step 1: Form Descriptor

```
→ GET /ui/forms/orders.edit_form
  Authorization: Bearer (Alice's token)
```

**BFF Internal:**

```
1.  DefinitionRegistry.GetForm("orders.edit_form") → FormDefinition

2.  Capability check: "orders:detail:edit" → ✓

3.  Resolve form sections and fields:
      Section "Order Details":
        shipping_address: type "text" → ✓
        priority: type "select" → ✓
        notes: type "textarea", capability "orders:notes:edit" → ✓
        internal_code: type "text", capability "orders:internal:edit" → ✗ → OMIT

4.  Resolve lookups:
      priority → options from definition (static: normal, high, urgent)
      status → options_endpoint: "/ui/lookups/orders.statuses"

5.  Resolve actions:
      "Submit": type "command", command_id: "orders.update"
      "Cancel": type "navigate", navigate_to: "/orders/{id}"
```

```
← 200 OK
  {
    "id": "orders.edit_form",
    "title": "Edit Order",
    "sections": [
      {
        "title": "Order Details",
        "fields": [
          { "id": "shipping_address", "label": "Shipping Address", "type": "text",
            "required": true },
          { "id": "priority", "label": "Priority", "type": "select",
            "options": [
              { "value": "normal", "label": "Normal" },
              { "value": "high", "label": "High" },
              { "value": "urgent", "label": "Urgent" }
            ] },
          { "id": "notes", "label": "Notes", "type": "textarea", "required": false }
        ]
      }
    ],
    "actions": [
      { "id": "submit", "label": "Save Changes", "type": "command",
        "command_id": "orders.update", "style": "primary" },
      { "id": "cancel", "label": "Cancel", "type": "navigate",
        "navigate_to": "/orders/{id}" }
    ],
    "data_endpoint": "/ui/forms/orders.edit_form/data"
  }
```

### Step 2: Pre-populated Data

```
→ GET /ui/forms/orders.edit_form/data?id=ord-123
  Authorization: Bearer (Alice's token)
```

**BFF Internal:**

```
1.  DefinitionRegistry.GetForm("orders.edit_form") → FormDefinition
      data_source:
        operation_id: "getOrder"
        service_id: "orders-svc"
        field_map:
          shipping_address: "shippingAddress"
          priority: "priority"
          notes: "internalNotes"

2.  Capability check ✓

3.  Invoke backend:
      → GET https://orders.internal/api/v1/orders/ord-123
      ← 200 OK
        {
          "id": "ord-123",
          "orderNumber": "ORD-2024-001",
          "shippingAddress": "123 Main St, Anytown, USA",
          "priority": "normal",
          "internalNotes": "Customer prefers morning delivery",
          "internalCode": "IC-SECRET-001",
          ...
        }

4.  Apply field_map (extract only mapped fields):
      "shippingAddress" → shipping_address
      "priority" → priority
      "internalNotes" → notes

    Note: "internalCode" is NOT extracted because Alice's form descriptor
    doesn't include that field (capability filtered out earlier)

5.  Return mapped data
```

```
← 200 OK
  {
    "shipping_address": "123 Main St, Anytown, USA",
    "priority": "normal",
    "notes": "Customer prefers morning delivery"
  }
```

The frontend fills the form fields with these values. Alice edits what she needs
and submits via `POST /ui/commands/orders.update`.

---

## Flow 9: Lookup Resolution

**Actor:** Frontend needs options for a select dropdown.

```
→ GET /ui/lookups/orders.statuses?q=
  Authorization: Bearer (Alice's token)
```

**BFF Internal:**

```
1.  DefinitionRegistry.GetLookup("orders.statuses") → LookupDefinition
      type: "dynamic"
      operation:
        service_id: "orders-svc"
        operation_id: "getOrderStatuses"
      mapping:
        value_field: "code"
        label_field: "displayName"
      cache:
        ttl: 300s
        scope: "tenant"

2.  Check cache:
      Key: "lookup:orders.statuses:acme-corp"
      → Cache HIT (loaded 2 minutes ago)
      → Return cached result

    (If cache MISS):
      Invoke: GET https://orders.internal/api/v1/orders/statuses
      ← 200 OK
        {
          "statuses": [
            { "code": "draft", "displayName": "Draft" },
            { "code": "pending", "displayName": "Pending" },
            { "code": "confirmed", "displayName": "Confirmed" },
            { "code": "shipped", "displayName": "Shipped" },
            { "code": "delivered", "displayName": "Delivered" },
            { "code": "cancelled", "displayName": "Cancelled" }
          ]
        }

      Apply mapping: value_field="code", label_field="displayName"
      Cache SET (TTL: 300s, scope: tenant)
```

```
← 200 OK
  {
    "options": [
      { "value": "draft", "label": "Draft" },
      { "value": "pending", "label": "Pending" },
      { "value": "confirmed", "label": "Confirmed" },
      { "value": "shipped", "label": "Shipped" },
      { "value": "delivered", "label": "Delivered" },
      { "value": "cancelled", "label": "Cancelled" }
    ]
  }
```

With a search term:

```
→ GET /ui/lookups/customers.by_name?q=bob
  → BFF filters results where label contains "bob" (case-insensitive)
  → If the backend supports search, "q" is forwarded as a query param
```

---

## Flow 10: Cross-Tenant Isolation — Denied Access

**Actor:** Eve (tenant: `evil-corp`) tries to access Alice's workflow.

Eve has somehow obtained the workflow instance ID `wf-uuid-001` (which belongs
to `acme-corp`).

```
→ GET /ui/workflows/wf-uuid-001
  Authorization: Bearer (Eve's token, tenant: "evil-corp")
```

**BFF Internal:**

```
1.  JWT verification:
      tenant_id: "evil-corp" (from Eve's token)
      subject_id: "user-eve"

2.  WorkflowStore.Get(ctx, "evil-corp", "wf-uuid-001")
      SQL: SELECT * FROM workflow_instances
           WHERE tenant_id = 'evil-corp' AND id = 'wf-uuid-001'
      → NO ROWS (workflow belongs to acme-corp, not evil-corp)

3.  Return 404 NOT_FOUND (not 403, to prevent ID enumeration)
```

```
← 404 Not Found
  {
    "error": {
      "code": "NOT_FOUND",
      "message": "Workflow not found"
    }
  }
```

Eve learns nothing — she doesn't know if the ID exists in another tenant,
if it ever existed, or if the format is wrong.

---

## Flow 11: Circuit Breaker — Backend Service Down

**Actor:** Alice requests the orders list, but the orders service is down.

```
→ GET /ui/pages/orders.list/data?page=1&page_size=25
  Authorization: Bearer (Alice's token)
```

**BFF Internal:**

```
1.  Middleware + capabilities ✓

2.  Build backend request for listOrders

3.  Check circuit breaker for orders-svc:
      State: OPEN (5 consecutive failures in last 30s)
      → Request REJECTED immediately (no network call made)

4.  Error: circuit breaker open
```

```
← 502 Bad Gateway
  {
    "error": {
      "code": "BACKEND_UNAVAILABLE",
      "message": "The orders service is temporarily unavailable. Please try again later.",
      "trace_id": "trace-789"
    }
  }
```

Internally, the BFF logs:

```json
{
  "level": "warn",
  "msg": "circuit breaker open for backend service",
  "service_id": "orders-svc",
  "circuit_state": "open",
  "tenant_id": "acme-corp",
  "subject_id": "user-alice",
  "correlation_id": "corr-025"
}
```

### Circuit Breaker Recovery

After the configured timeout (30s), the circuit breaker transitions to HALF-OPEN:

```
Next request to orders-svc:
  Circuit state: HALF-OPEN → allow one probe request
  → GET https://orders.internal/api/v1/orders?offset=0&limit=25...
  ← 200 OK (service recovered)
  → Circuit state: CLOSED (healthy)

  All subsequent requests flow normally.
```

---

## Flow 12: Definition Hot-Reload

**Actor:** DevOps deploys a new definition file that adds a "priority" column
to the orders list page.

### Hot-Reload Sequence

```
1.  File watcher detects change:
      definitions/orders/definition.yaml modified
      Debounce: wait 2s for batch changes

2.  DefinitionLoader.LoadFile("definitions/orders/definition.yaml")
      Parse YAML → new DomainDefinition
      Compute SHA-256: "abc123..." (new) vs "def456..." (old)

3.  Validate new definition:
      a. Structural validation (required fields, valid enums) → ✓
      b. Cross-reference validation (form_ids, command_ids exist) → ✓
      c. OpenAPI validation (all operation_ids exist in specs) → ✓

4.  DefinitionRegistry.Replace(newDefinitions)
      Atomic pointer swap:
        old = registry.snapshot.Load()
        new = buildSnapshot(newDefinitions)
        registry.snapshot.Store(new)
      (Lock-free: in-flight reads on old snapshot complete safely)

5.  Log:
      {
        "level": "info",
        "msg": "definitions reloaded",
        "old_checksum": "def456...",
        "new_checksum": "abc123...",
        "domains_affected": ["orders"],
        "definitions_count": 3
      }

6.  Metric:
      thesa_definition_reload_total{status="success"} +1
```

All subsequent requests see the new definitions immediately. In-flight requests
that obtained a reference to the old snapshot before the swap complete without
interruption (they use the old snapshot's data, which remains valid until
garbage collected).

### Validation Failure During Reload

```
1.  File watcher detects change

2.  Parse new definition → YAML parse error

3.  Log:
      {
        "level": "error",
        "msg": "definition reload failed",
        "error": "yaml: line 42: mapping values are not allowed in this context",
        "file": "definitions/orders/definition.yaml"
      }

4.  Registry is NOT updated — old definitions remain active

5.  Metric:
      thesa_definition_reload_total{status="failure"} +1

6.  Health check remains "ready" — serving with old definitions
```

---

## Flow Summary Table

| Flow | Scenario | Key Steps | Error Paths |
|------|----------|-----------|-------------|
| 1 | Page load with data | Navigation → Page descriptor → Data fetch | — |
| 2 | Command execution | Input mapping → Schema validation → Backend invoke → Output mapping | Idempotency replay, input conflict |
| 3 | Validation failure | Schema validation catches invalid input | 422 with field errors |
| 4 | Backend error translation | Backend 4xx → error_map → translated message | Unknown error → generic message |
| 5 | Full workflow lifecycle | Start → Review (user) → Process (system) → Notify (system) → Terminal | — |
| 6 | Workflow timeout | Background processor → auto-advance or fail | No timeout handler → status "failed" |
| 7 | Global search | Parallel provider invocation → merge → score → paginate | Provider timeout → omit results |
| 8 | Form with pre-populated data | Form descriptor → Data fetch → field mapping | — |
| 9 | Lookup resolution | Cache check → Backend invoke → mapping → cache store | — |
| 10 | Cross-tenant isolation | Tenant-scoped query → 404 (not 403) | — |
| 11 | Circuit breaker | State check → immediate rejection → 502 | Recovery via half-open probe |
| 12 | Definition hot-reload | File watch → parse → validate → atomic swap | Validation failure → keep old defs |

---

## Related Documents

- [09 — Server-Driven UI APIs](09-server-driven-ui-apis.md) — endpoint specifications
- [10 — Command and Action Model](10-command-and-action-model.md) — command pipeline details
- [11 — Workflow Engine](11-workflow-engine.md) — workflow mechanics
- [12 — Global Search](12-global-search.md) — search aggregation
- [13 — API Mapping and Invocation](13-api-mapping-and-invocation.md) — backend invocation details
- [16 — Observability and Reliability](16-observability-and-reliability.md) — circuit breakers, retries
- [17 — Security Model](17-security-model.md) — tenant isolation controls
- [20 — Example Domain: Orders](20-example-domain-orders.md) — complete orders definition
