# 15 — Error Handling and Validation

This document describes the error handling strategy: how errors are categorized,
how backend errors are translated, how validation works at each layer, and the
standard error envelope format.

---

## Error Envelope

All error responses from the BFF use a consistent envelope:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "One or more fields are invalid",
    "details": [
      {
        "field": "shipping_address",
        "code": "REQUIRED",
        "message": "Shipping address is required"
      }
    ],
    "trace_id": "abc-123-def"
  }
}
```

### Fields

| Field | Type | Always Present | Description |
|-------|------|----------------|-------------|
| `code` | string | Yes | Machine-readable error code (uppercase, underscored) |
| `message` | string | Yes | Human-readable error description |
| `details` | FieldError[] | No | Field-level errors (for validation errors) |
| `trace_id` | string | Yes | Distributed trace ID for debugging |

### FieldError

| Field | Type | Description |
|-------|------|-------------|
| `field` | string | Field path (e.g., "shipping_address", "items[0].quantity") |
| `code` | string | Error code (REQUIRED, MIN_LENGTH, MAX_LENGTH, INVALID_VALUE, PATTERN, etc.) |
| `message` | string | Human-readable message for this field error |

---

## Error Code Catalog

### Client Errors (4xx)

| HTTP | Code | When | Details |
|------|------|------|---------|
| 400 | `BAD_REQUEST` | Malformed JSON, missing required envelope fields | — |
| 401 | `UNAUTHORIZED` | Missing or invalid JWT token | — |
| 403 | `FORBIDDEN` | Valid token but insufficient capabilities | — |
| 404 | `NOT_FOUND` | Page, form, command, workflow, or resource not found | — |
| 409 | `CONFLICT` | Idempotency key conflict or optimistic lock conflict | — |
| 409 | `WORKFLOW_NOT_ACTIVE` | Attempting to advance a completed/cancelled workflow | — |
| 422 | `VALIDATION_ERROR` | Input validation failed | Field-level details |
| 422 | `INVALID_TRANSITION` | Workflow event not valid for current step | — |
| 429 | `RATE_LIMITED` | Rate limit exceeded | Retry-After header |

### Server Errors (5xx)

| HTTP | Code | When | Details |
|------|------|------|---------|
| 500 | `INTERNAL_ERROR` | Unexpected server error | Never includes backend details |
| 502 | `BACKEND_UNAVAILABLE` | Backend service unreachable or circuit breaker open | — |
| 504 | `BACKEND_TIMEOUT` | Backend service timed out | — |

### Design Rules for Error Responses

1. **Never leak backend details in 5xx errors.** The error message for 500 is always
   generic. Backend error details are logged server-side with the trace_id.

2. **404 for wrong tenant, not 403.** If a user requests a resource belonging to a
   different tenant, return 404 (not 403). This prevents tenant ID enumeration.

3. **Field errors use UI field names.** Backend field names are reverse-mapped to UI
   field names before returning validation errors.

4. **Always include trace_id.** Every error response includes the distributed trace
   ID so support teams can correlate errors with logs.

---

## Validation Layers

The BFF validates requests at five layers. Each layer short-circuits: if an early
layer fails, later layers are skipped.

### Layer 1: Transport Validation

**When:** Before any business logic.
**What:** HTTP-level checks.

- Request body is valid JSON → 400 if not.
- Content-Type is application/json (for POST requests) → 400 if not.
- Required envelope fields present (e.g., `input` field in command request) → 400 if not.

### Layer 2: Definition Resolution

**When:** After parsing the request.
**What:** The referenced resource exists.

- Page ID exists in registry → 404 if not.
- Form ID exists → 404 if not.
- Command ID exists → 404 if not.
- Workflow ID exists → 404 if not.

### Layer 3: Capability Validation

**When:** After definition resolution.
**What:** The user is authorized.

- User has required capabilities → 403 if not.

### Layer 4: Schema Validation

**When:** After capability check, before backend invocation.
**What:** Input data matches the expected schema.

For commands:
1. Apply InputMapping to construct the backend request body.
2. Validate constructed body against the OpenAPI request schema.
3. If validation fails → 422 with field-level errors.

Validation checks include:
- Required fields present.
- String lengths within bounds.
- Numbers within range.
- Enum values valid.
- Pattern matching (regex).
- Type correctness (string, number, boolean).

### Layer 5: Backend Validation

**When:** After backend invocation returns.
**What:** The backend rejected the request for business reasons.

If the backend returns a 4xx error:
1. Parse the backend error response.
2. Map error codes using the command's `error_map`.
3. Extract field-level errors if present.
4. Translate field names to UI field names.
5. Return the translated error envelope.

---

## Backend Error Translation

### How It Works

Each command definition can include an `error_map`:

```yaml
output:
  error_map:
    ORDER_NOT_FOUND: "This order no longer exists. It may have been deleted."
    INVALID_STATUS: "This order cannot be modified in its current status."
    INSUFFICIENT_STOCK: "Insufficient stock for one or more items."
    DUPLICATE_ORDER: "An order with this reference already exists."
```

### Translation Process

```
1. Backend returns 4xx response:
   { "error": { "code": "ORDER_NOT_FOUND", "message": "Order xyz not found" } }

2. BFF extracts error code: "ORDER_NOT_FOUND"

3. Look up in error_map:
   Found: "This order no longer exists. It may have been deleted."

4. Return translated error:
   { "error": { "code": "ORDER_NOT_FOUND", "message": "This order no longer exists. It may have been deleted.", "trace_id": "..." } }
```

### Unmapped Errors

If the backend error code is NOT in the error_map:

```
1. Log the backend error with full context (for debugging).
2. Return a generic error:
   { "error": { "code": "COMMAND_FAILED", "message": "The operation could not be completed. Please try again.", "trace_id": "..." } }
```

This prevents leaking internal error messages to the frontend.

### Backend Error Detection

The BFF attempts to extract error information from backend responses using
configurable strategies:

```yaml
# Service-level error extraction configuration
services:
  orders-svc:
    error_extraction:
      code_path: "error.code"           # JSON path to error code
      message_path: "error.message"     # JSON path to error message
      details_path: "error.details"     # JSON path to field errors
      field_name_key: "field"           # Key for field name in details
      field_message_key: "message"      # Key for error message in details
```

Default extraction tries common patterns:
1. `{ "error": { "code": "...", "message": "..." } }`
2. `{ "code": "...", "message": "..." }`
3. `{ "error": "...", "error_description": "..." }`

---

## Workflow-Specific Errors

| Code | HTTP | When |
|------|------|------|
| `WORKFLOW_NOT_FOUND` | 404 | Instance ID doesn't exist or belongs to different tenant |
| `WORKFLOW_NOT_ACTIVE` | 409 | Attempting to advance a completed/cancelled/failed workflow |
| `INVALID_TRANSITION` | 422 | Event is not valid for the current step |
| `STEP_UNAUTHORIZED` | 403 | User lacks capability for the current step |
| `WORKFLOW_EXPIRED` | 409 | Workflow or step has timed out |
| `WORKFLOW_CHAIN_LIMIT` | 500 | Too many consecutive system steps (misconfiguration) |

---

## Error Logging

### What Gets Logged

Every error is logged with structured fields:

```json
{
  "level": "error",
  "msg": "command execution failed",
  "error_code": "VALIDATION_ERROR",
  "command_id": "orders.update",
  "tenant_id": "acme-corp",
  "subject_id": "user-123",
  "correlation_id": "corr-456",
  "trace_id": "trace-789",
  "backend_status": 422,
  "backend_error": "INVALID_STATUS",
  "duration_ms": 145
}
```

### What Does NOT Get Logged

- User input values (PII risk).
- JWT token contents.
- Full request/response bodies (except at DEBUG level, redacted).

### Log Levels

| Level | When |
|-------|------|
| ERROR | 5xx responses, infrastructure failures, configuration errors |
| WARN | 4xx responses (client errors), degraded functionality |
| INFO | Request completion (success), definition reload, startup events |
| DEBUG | Full request/response bodies (redacted), cache hits/misses |

---

## Frontend Error Handling

### Recommended Frontend Strategy

```
1. Parse error envelope.
2. Check error.code:
   a. UNAUTHORIZED (401) → redirect to login
   b. FORBIDDEN (403) → show "access denied" message
   c. NOT_FOUND (404) → show "not found" page
   d. VALIDATION_ERROR (422) → highlight fields in error.details
   e. RATE_LIMITED (429) → show retry message with Retry-After value
   f. CONFLICT (409) → show conflict message, suggest refresh
   g. BACKEND_UNAVAILABLE (502) → show "service temporarily unavailable"
   h. BACKEND_TIMEOUT (504) → show "request timed out, please try again"
   i. INTERNAL_ERROR (500) → show "unexpected error" with trace_id
3. For unknown codes: show generic error with trace_id for support.
```

### Field Error Highlighting

For VALIDATION_ERROR with details:

```json
{
  "details": [
    { "field": "shipping_address", "code": "REQUIRED", "message": "Required" },
    { "field": "priority", "code": "INVALID_VALUE", "message": "Invalid value" }
  ]
}
```

The frontend matches `field` to form field names and displays the error message
next to the corresponding input.
