# 10 — Command and Action Model

This document describes the command execution pipeline — the sole path through which
the frontend triggers mutations on backend services. It covers the full lifecycle of a
command from frontend invocation to backend execution and response translation.

---

## Design Rationale

The frontend NEVER constructs direct API calls to backend services. Instead, every
mutation flows through a single endpoint:

```
POST /ui/commands/{commandId}
```

This design provides:

| Benefit | How |
|---------|-----|
| **Uniform authorization** | Every command is capability-checked through the same code path |
| **Uniform validation** | Every command's payload is validated against the OpenAPI schema |
| **Uniform observability** | Every command is logged, traced, and metered with the same structure |
| **Uniform error handling** | Backend errors are translated through the same mapping layer |
| **Idempotency** | Available for every command via a standard header |
| **Rate limiting** | Configurable per command |
| **Auditability** | Every command is tied to a subject, tenant, and correlation ID |
| **Decoupling** | Backend API shape can change without frontend changes |

---

## Command Execution Pipeline

### Overview

```
POST /ui/commands/orders.update
  │
  ├── 1. Parse request body
  ├── 2. Lookup command definition
  ├── 3. Evaluate capabilities
  ├── 4. Check idempotency
  ├── 5. Check rate limit
  ├── 6. Apply input mapping
  ├── 7. Validate against OpenAPI schema
  ├── 8. Invoke backend
  ├── 9. Process response
  └── 10. Return result
```

### Step 1: Parse Request Body

> **Note:** This step occurs in the **transport layer** (HTTP handler), not in the
> CommandExecutor itself. The executor receives a pre-parsed `CommandInput` struct
> and a pre-resolved `CapabilitySet`.

Extract the command input from the HTTP request:

```json
{
  "input": { "shipping_address": "123 Main St", "priority": "high" },
  "route_params": { "id": "ord-123" },
  "idempotency_key": "key-abc"
}
```

Validation:
- `input` must be a JSON object (not null, not array) → 400 if invalid.
- `route_params` is optional; defaults to empty map.
- `idempotency_key` may also come from the `Idempotency-Key` header.

### Step 2: Lookup Command Definition

```
registry.GetCommand("orders.update") → (CommandDefinition, true)
```

If not found → 404 with error:
```json
{ "error": { "code": "NOT_FOUND", "message": "Command 'orders.update' not found" } }
```

### Step 3: Evaluate Capabilities

```
For each capability in command.capabilities:
  If !capabilitySet.Has(capability) → 403
```

Error response:
```json
{ "error": { "code": "FORBIDDEN", "message": "Insufficient permissions to execute this command" } }
```

Note: The error does NOT reveal which specific capability is missing. This prevents
information leakage that could aid privilege escalation attempts.

### Step 4: Check Idempotency

If the command has an `idempotency` configuration and a key is provided:

```
1. Look up idempotencyKey in the idempotency store.
2. If found:
   a. Compare stored input hash with current input hash.
   b. If match: return the cached response (replay).
   c. If mismatch: return 409 Conflict:
      { "error": { "code": "CONFLICT", "message": "Idempotency key already used with different input" } }
3. If not found: continue (will store after execution).
```

### Step 5: Check Rate Limit

If the command has a `rate_limit` configuration:

```
1. Compute rate limit key:
   - scope "user": hash(subjectId + commandId)
   - scope "tenant": hash(tenantId + commandId)
   - scope "global": hash(commandId)
2. Check rate limit counter.
3. If exceeded → 429:
   { "error": { "code": "RATE_LIMITED", "message": "Too many requests. Try again later." } }
   Headers: Retry-After: 30
```

### Step 6: Apply Input Mapping

This is where the user's input is transformed into a backend request.

**Path parameters:**
```yaml
path_params:
  orderId: "route.id"       # → extract "id" from route_params → "ord-123"
  tenantId: "context.tenant_id"  # → extract from RequestContext → "acme-corp"
```

**Query parameters:**
```yaml
query_params:
  expand: "'items,customer'"  # literal value (single-quoted)
  format: "input.format"      # from user input
```

**Header parameters:**
```yaml
header_params:
  X-Correlation-ID: "context.correlation_id"
  X-Tenant-ID: "context.tenant_id"
```

Headers are resolved with the same expression syntax as path and query parameters.

**Body mapping — "passthrough" (default):**
```
backend_body = input    # send user input as-is
```

> If `body_mapping` is empty or unspecified, it defaults to "passthrough".

**Body mapping — "template":**
```yaml
body_template:
  customerId: "input.customer_id"      # → "cust-002"
  shippingAddress: "input.shipping_address"  # → "456 Oak Ave"
  updatedBy: "context.subject_id"      # → "user-123"
  source: "'bff'"                      # literal → "bff"
  correlationId: "context.correlation_id"  # → from RequestContext
```

Each key-value pair in the template is resolved against the expression resolver.
Values that match a source expression pattern (e.g., `input.field`, `context.tenant_id`,
`'literal'`) are resolved. The template is a flat `map[string]string` — nested structures
are not supported.

**Body mapping — "projection":**
```yaml
field_projection:
  customerId: "input.customer_id"
  shippingAddress: "input.shipping_address"
  priority: "input.priority"
```

Only the projected fields are included in the body. Other input fields are dropped.
This prevents the frontend from injecting unexpected fields.

### Source Expression Reference

The expression resolver supports these prefixes:

| Prefix | Source | Example |
|--------|--------|---------|
| `input.` | User-provided input fields | `input.customer_id`, `input.address.city` (nested) |
| `route.` | Route parameters from the URL | `route.id` |
| `context.` | RequestContext (from JWT) | `context.subject_id`, `context.tenant_id`, `context.partition_id`, `context.email` |
| `workflow.` | Workflow state (when invoked from a workflow step) | `workflow.order_id` |
| `'literal'` | Single-quoted literal string | `'bff'`, `'items,customer'` |
| Numeric | Numeric literal (int64 or float64) | `123`, `99.99` |

The `input.` prefix supports nested field access via dot notation (e.g., `input.address.city`
navigates into a nested `address` object to find the `city` field).

### Step 7: Validate Against OpenAPI Schema

If the operation binding is `type: "openapi"`:

```
1. Look up the operation in the OpenAPI index.
2. Get the request body schema.
3. Validate the constructed body against the schema.
4. If validation fails → 422 with field-level errors:
```

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "details": [
      { "field": "customerId", "code": "REQUIRED", "message": "customerId is required" },
      { "field": "priority", "code": "INVALID_VALUE", "message": "priority must be one of: normal, high, urgent" }
    ]
  }
}
```

Note: The field names in validation errors use **backend field names** (since validation
is against the backend schema). The BFF translates these back to **UI field names** using
the reverse field_map before returning to the frontend.

### Step 8: Invoke Backend

```
invokerRegistry.Invoke(ctx, requestContext, command.Operation, invocationInput)
```

The invoker dispatches to the appropriate implementation:
- `type: "openapi"` → OpenAPIOperationInvoker
- `type: "sdk"` → SDKOperationInvoker

The invocation includes:
- Circuit breaker check
- Timeout enforcement
- Retry logic (for idempotent operations only)
- Distributed tracing span

### Step 9: Process Response

**On success (2xx):**

```
1. Parse JSON response body.
2. Apply OutputMapping:
   a. If output.fields is empty: return the full backend response body as the result.
   b. If output.fields is present: extract and rename only the listed fields (projection).
3. Build CommandResponse { success: true, message: output.success_message, result: {...} }.
4. Store idempotency key → response (if configured, and only on success).
5. Return CommandResponse.
```

> **Note:** The `output.type` field is defined in the schema but the executor does not
> branch on it. The actual behavior is determined by whether `output.fields` is populated.

**On client error (4xx):**

```
1. Parse error response body.
2. Attempt to match error code against output.error_map:
   - If matched: use the mapped UI-friendly message.
   - If not matched: use a generic message ("An error occurred").
3. If backend returns field-level errors:
   - Map field names from backend → UI using reverse field_map.
   - Return as details[] in error envelope.
4. Return error envelope with appropriate HTTP status.
```

**On server error (5xx):**

```
1. Log the full error with backend response body, status, and headers.
2. Do NOT return backend error details to the frontend.
3. Return generic error:
   { "error": { "code": "INTERNAL_ERROR", "message": "An unexpected error occurred", "trace_id": "..." } }
```

### Step 10: Return Result

The CommandResponse is serialized to JSON and returned to the frontend.

Metrics emitted:
- `thesa_command_executions_total{command_id="orders.update", status="success"}`
- `thesa_command_duration_seconds{command_id="orders.update"}`

---

## CommandExecutor Interface

```
CommandExecutor
  ├── Execute(ctx, rctx, caps CapabilitySet, commandId string, input CommandInput) → (CommandResponse, error)
  │     Full pipeline execution. Capabilities are pre-resolved by the transport layer.
  │     Returns a CommandResponse on success.
  │     Returns an error only for infrastructure failures (not business errors).
  │     Business errors (validation, authorization) are encoded in CommandResponse.
  │
  └── Validate(rctx RequestContext, caps CapabilitySet, commandId string, input CommandInput) → []FieldError
        Validates input against the command's schema without executing.
        Used for dry-run / pre-flight validation.
        Requires rctx for expression resolution and caps for capability checks.
```

---

## Idempotency

### Why Idempotency Matters

Network failures, retries, and double-clicks can cause duplicate command executions.
For commands that create resources or trigger irreversible actions, idempotency ensures
that retrying the same command produces the same result without side effects.

### How It Works

```
Request 1: POST /ui/commands/orders.create
  Idempotency-Key: key-abc
  Body: { input: { customer_id: "cust-001", items: [...] } }

  → Backend creates order ord-456
  → BFF stores: key-abc → { status: 200, body: { id: "ord-456" } }
  → Response: { success: true, result: { id: "ord-456" } }

Request 2: POST /ui/commands/orders.create  (retry, same key)
  Idempotency-Key: key-abc
  Body: { input: { customer_id: "cust-001", items: [...] } }

  → BFF finds key-abc in store
  → Input hash matches → return cached response
  → Response: { success: true, result: { id: "ord-456" } }
  → No second backend call made

Request 3: POST /ui/commands/orders.create  (different input, same key)
  Idempotency-Key: key-abc
  Body: { input: { customer_id: "cust-002", items: [...] } }

  → BFF finds key-abc in store
  → Input hash does NOT match → 409 Conflict
```

### Key Source Configuration

```yaml
idempotency:
  key_source: "header"                    # Read from Idempotency-Key HTTP header
  # OR
  key_source: "input"                     # Read from idempotency_key in request body
  # OR
  key_source: "auto"                      # BFF generates from hash of input + route_params
  ttl: "24h"                              # How long to remember keys (Go duration format)
```

> **Note:** The transport layer is responsible for extracting the idempotency key
> based on `key_source` and placing it into `CommandInput.IdempotencyKey` before
> calling the executor. The executor receives the key already extracted.

### Store Requirements

The idempotency store must be shared across BFF instances (Redis or PostgreSQL).
Per-instance stores would not catch retries hitting different instances.

---

## Rate Limiting

### Configuration

```yaml
rate_limit:
  max_requests: 10        # Max requests allowed in the window
  window: "1m"            # Time window
  scope: "user"           # "user" (per-user), "tenant" (per-tenant), "global"
```

### Implementation

Uses a sliding window counter (or token bucket) in Redis:

```
Key: "ratelimit:{scope_key}:{commandId}"
Increment counter, set TTL to window duration.
If counter > max_requests → reject.
```

### Response Headers

When rate limited:
```
HTTP/1.1 429 Too Many Requests
Retry-After: 30
X-RateLimit-Limit: 10
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1705312800
```

---

## Error Translation

### Error Map

The `error_map` in a CommandDefinition translates backend error codes to
user-friendly messages:

```yaml
output:
  error_map:
    ORDER_NOT_FOUND: "This order no longer exists. It may have been deleted."
    INVALID_STATUS: "This order cannot be modified in its current status."
    INSUFFICIENT_STOCK: "Insufficient stock for one or more items."
    DUPLICATE_ORDER: "An order with this reference number already exists."
```

### Backend Error Detection

The BFF extracts error codes and messages from backend responses by looking for
well-known field paths in the response body:

1. **Error code:** First tries `error.code`, then falls back to `code`.
2. **Error message:** First tries `error.message`, then falls back to `message`.
3. **Field errors:** Looks for `error.details` or `details` as an array of
   `{ field, code, message }` objects.

### Field Error Translation

When the backend returns field-level validation errors, the BFF:

1. Extracts field names from the error response.
2. Reverses the field_projection/field_map to convert backend field names to UI field names.
3. Returns the errors with UI field names so the frontend can highlight the correct fields.

```
Backend error: { "errors": [{ "field": "shippingAddress", "message": "too long" }] }
Reverse map: "shippingAddress" → "shipping_address"
BFF error: { "details": [{ "field": "shipping_address", "code": "INVALID", "message": "too long" }] }
```

---

## Actions and Their Relationship to Commands

Actions (defined in pages and tables) are the UI elements that trigger commands.
The relationship is:

```
ActionDefinition (type: "command")
  └── command_id: "orders.update"
       └── CommandDefinition
            └── operation_id: "updateOrder"
                 └── OpenAPI Operation
```

When the frontend renders an ActionDescriptor with `type: "command"`, it shows a
button. When clicked:
1. Frontend sends `POST /ui/commands/{commandId}` with the input and route params.
2. BFF executes the full pipeline described above.
3. Frontend displays the result (success message or errors).

### Actions with Confirmation

When an action has a `confirmation` definition, the frontend must:
1. Show the confirmation dialog first.
2. Only execute the command if the user confirms.
3. The confirmation dialog content is in the ActionDescriptor — the BFF provides
   the title, message, confirm button text, and cancel button text.

### Actions with Conditions

Actions can be conditionally visible based on resource data:

```json
{
  "id": "orders.cancel_action",
  "conditions": [
    { "field": "status", "operator": "in", "value": "pending,confirmed", "effect": "show" }
  ]
}
```

> For `in`/`not_in` operators, the value is a comma-separated string of allowed values.

The frontend evaluates these conditions against the current resource data and
shows/hides/enables/disables the action accordingly. This is client-side logic
based on BFF-provided rules.

---

## Testing Commands

### Unit Testing the Executor

```go
func TestCommandExecutor_Execute(t *testing.T) {
    // Setup mock registry with a test command definition
    // Setup mock invoker that returns a canned response
    // Setup mock capability set that grants required capabilities

    executor := command.NewCommandExecutor(registry, invokerRegistry, openAPIIndex)

    resp, err := executor.Execute(ctx, rctx, caps, "orders.update", input)

    assert.NoError(t, err)
    assert.True(t, resp.Success)
    assert.Equal(t, "Order updated", resp.Message)
}
```

### Integration Testing

```
Test: "Execute command with valid input"
  1. Load definitions and OpenAPI specs
  2. Start mock backend server
  3. POST /ui/commands/orders.update with valid input
  4. Assert: 200, success: true

Test: "Execute command with missing capability"
  1. Authenticate as user without orders:edit:execute
  2. POST /ui/commands/orders.update
  3. Assert: 403

Test: "Execute command with invalid input"
  1. POST /ui/commands/orders.update with invalid field values
  2. Assert: 422, field-level errors

Test: "Execute command with idempotency"
  1. POST /ui/commands/orders.create with Idempotency-Key: test-key
  2. Assert: 200, success: true
  3. POST /ui/commands/orders.create with same key and input
  4. Assert: 200, same response (from cache)
  5. POST /ui/commands/orders.create with same key, different input
  6. Assert: 409 Conflict
```
