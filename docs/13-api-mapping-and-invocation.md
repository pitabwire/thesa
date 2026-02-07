# 13 — API Mapping and Invocation Engine

This document describes how Thesa maps between UI-level operations and backend
API calls — including the OpenAPI index, dynamic request building, SDK handlers,
and the standardization of pagination, filtering, and sorting.

---

## OpenAPI Index

The OpenAPI Index is the core data structure that enables dynamic invocation.
It parses OpenAPI specifications at startup and provides fast lookups by
`(serviceId, operationId)`.

### Structure

```
OpenAPIIndex
  ├── Load(specs []SpecSource) → error
  │     Loads and indexes all specs. Called at startup.
  │
  ├── GetOperation(serviceId, operationId string) → (IndexedOperation, bool)
  │     Returns the indexed operation metadata for invocation.
  │
  ├── ValidateRequest(serviceId, operationId string, body any) → []ValidationError
  │     Validates a request body against the operation's schema.
  │
  └── AllOperationIDs(serviceId string) → []string
        Lists all operation IDs for a service (for diagnostics).
```

### SpecSource

```
SpecSource
  ├── ServiceID    string    // Logical service name (e.g., "orders-svc")
  ├── SpecPath     string    // Path to the OpenAPI spec file
  ├── BaseURL      string    // Runtime base URL (e.g., "https://orders.internal:8443")
  └── Timeout      duration  // Per-request timeout for this service
```

### IndexedOperation

```
IndexedOperation
  ├── ServiceID       string
  ├── OperationID     string
  ├── Method          string                    // GET, POST, PUT, PATCH, DELETE
  ├── PathTemplate    string                    // e.g., "/api/v1/orders/{orderId}"
  ├── Parameters      []*openapi3.Parameter     // Path, query, header parameters (kin-openapi type)
  ├── RequestBody     *openapi3.RequestBody     // Request body schema (kin-openapi type)
  ├── Responses       *openapi3.Responses       // Response schemas (kin-openapi type)
  └── BaseURL         string                    // From SpecSource
```

### Loading Process

```
For each SpecSource:
  1. Read file (YAML or JSON).
  2. Parse using OpenAPI library (e.g., kin-openapi).
  3. Validate spec structure.
  4. Iterate all paths and methods:
     For each path (e.g., "/api/v1/orders/{orderId}"):
       For each method (e.g., GET, PUT):
         Extract operationId.
         → Silently skipped if operationId is missing (operation cannot be referenced without an ID).
         Build IndexedOperation.
         Store in map[(serviceId, operationId)] → IndexedOperation.
  5. Log: "Loaded N operations from service 'orders-svc'"
```

### Example

For this OpenAPI path:

```yaml
/api/v1/orders/{orderId}:
  patch:
    operationId: updateOrder
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
            properties:
              customerId: { type: string }
              shippingAddress: { type: string, maxLength: 500 }
              priority: { type: string, enum: [normal, high, urgent] }
    responses:
      200:
        content:
          application/json:
            schema:
              type: object
              properties:
                data:
                  type: object
                  properties:
                    id: { type: string }
                    orderNumber: { type: string }
```

The index stores:

```
Key: ("orders-svc", "updateOrder")
Value: IndexedOperation{
  Method: "PATCH",
  PathTemplate: "/api/v1/orders/{orderId}",
  Parameters: [{ Name: "orderId", In: "path", Required: true }],
  RequestBody: <schema for { customerId, shippingAddress, priority }>,
  Responses: { 200: <schema for { data: { id, orderNumber } }> },
  BaseURL: "https://orders.internal:8443",
}
```

---

## OpenAPI Invoker — Dynamic HTTP Request Building

### Request Construction

Given a definition's `OperationBinding` and `InputMapping`, the invoker builds
a complete HTTP request:

```
buildRequest(operation, input, requestContext):

  1. URL Construction:
     url = operation.BaseURL + operation.PathTemplate

     For each path parameter in operation.Parameters where In == "path":
       value = input.PathParams[param.Name]
       url = replace("{" + param.Name + "}", urlEncode(value), url)

     Example: "https://orders.internal:8443/api/v1/orders/{orderId}"
            → "https://orders.internal:8443/api/v1/orders/ord-123"

  2. Query Parameters:
     For each entry in input.QueryParams:
       Append as URL query parameter.

     Example: url += "?expand=items,customer"

  3. Headers:
     headers["Accept"] = "application/json"
     If method is POST, PUT, or PATCH:
       headers["Content-Type"] = "application/json"
     headers["Authorization"] = "Bearer " + requestContext.Token
     headers["X-Tenant-Id"] = requestContext.TenantID
     headers["X-Partition-Id"] = requestContext.PartitionID
     headers["X-Correlation-Id"] = requestContext.CorrelationID
     headers["X-Request-Subject"] = requestContext.SubjectID

     For each entry in input.Headers:
       headers[key] = value

  4. Body:
     If input.Body != nil:
       body = json.Marshal(input.Body)

  5. Return: method, url, headers, body
```

### Connection Pooling

Each service gets its own `http.Client` with a configured transport:

```
Per-service client:
  Transport:
    MaxIdleConns: 100
    MaxConnsPerHost: 50
    IdleConnTimeout: 90s
    TLSHandshakeTimeout: 10s
  Timeout: operation.Timeout
```

Clients are created once at startup and reused for all requests to that service.

### Circuit Breaker Integration

Before executing the HTTP request:

```
1. Check circuit breaker state for this service.
2. If OPEN: return error immediately (BACKEND_CIRCUIT_OPEN).
3. If CLOSED or HALF_OPEN: proceed with request.
4. After response:
   a. If success (2xx): record success.
   b. If server error (5xx) or connection error: record failure.
   c. If client error (4xx): do NOT count as circuit breaker failure
      (client errors are not infrastructure issues).
5. If failure threshold reached: open circuit.
```

### Retry Logic

```
For methods: GET, PUT, DELETE (idempotent by HTTP spec):
  If response is: 502, 503, 504, or connection error:
    Retry up to max_attempts times.
    Backoff: 100ms, 200ms, 400ms (exponential).

For methods: POST, PATCH (non-idempotent):
  NO retry unless the command has idempotency configured.
  If idempotency is configured: retry is safe (same key, same result).
```

---

## SDK Invoker

### When the SDK Invoker Is Used

The SDK invoker handles `type: "sdk"` operation bindings. These are Go
implementations registered at startup for specialized use cases.

### SDKHandler Interface

```
SDKHandler
  ├── Name() → string
  │     Returns the handler name (e.g., "ledger.PostEntry").
  │
  └── Invoke(ctx, rctx RequestContext, input InvocationInput) → (InvocationResult, error)
        Executes the operation using a typed client.
```

### Registration

```go
// In main.go or a setup function:
sdkRegistry := invoker.NewSDKHandlerRegistry()

// Register Connect RPC handler
ledgerClient := ledgerv1connect.NewLedgerServiceClient(httpClient, ledgerBaseURL)
sdkRegistry.Register("ledger.PostEntry", &LedgerPostEntryHandler{client: ledgerClient})

// Register gRPC handler
notifConn, _ := grpc.Dial(notifAddress, ...)
notifClient := notifpb.NewNotificationServiceClient(notifConn)
sdkRegistry.Register("notifications.Send", &NotificationSendHandler{client: notifClient})
```

### Example SDK Handler

```go
type LedgerPostEntryHandler struct {
    client ledgerv1connect.LedgerServiceClient
}

func (h *LedgerPostEntryHandler) Name() string { return "ledger.PostEntry" }

func (h *LedgerPostEntryHandler) Invoke(ctx context.Context, rctx model.RequestContext, input model.InvocationInput) (*model.InvocationResult, error) {
    // Build typed request from generic input
    req := connect.NewRequest(&ledgerv1.PostEntryRequest{
        TenantId:    rctx.TenantID,
        DebitAccount:  input.Body.(map[string]any)["debit_account"].(string),
        CreditAccount: input.Body.(map[string]any)["credit_account"].(string),
        Amount:        input.Body.(map[string]any)["amount"].(float64),
        Currency:      input.Body.(map[string]any)["currency"].(string),
        Reference:     input.Body.(map[string]any)["reference"].(string),
    })

    // Execute typed RPC
    resp, err := h.client.PostEntry(ctx, req)
    if err != nil {
        return nil, err
    }

    // Convert typed response to generic result
    return &model.InvocationResult{
        StatusCode: 200,
        Body: map[string]any{
            "entry_id":   resp.Msg.EntryId,
            "created_at": resp.Msg.CreatedAt.AsTime().Format(time.RFC3339),
        },
    }, nil
}
```

---

## Pagination Standardization

### The Problem

Different backend services use different pagination conventions:

| Service | Style | Parameters |
|---------|-------|------------|
| Orders | Offset/limit | `?offset=25&limit=25` |
| Customers | Page/per_page | `?page=2&per_page=25` |
| Inventory | Cursor | `?cursor=eyJpZCI6MTI1fQ&limit=25` |
| Ledger | Offset/limit | `?skip=25&take=25` |

### The Solution

> **Implementation status:** The `PaginationConfig` struct exists in the configuration
> model, but pagination style translation is not yet applied at runtime. Currently,
> `page` and `page_size` are passed directly to backends as query parameters. The
> translation logic and cursor caching described below are planned.

The BFF standardizes to page-based pagination on the frontend:

```
Frontend: ?page=2&page_size=25
```

The BFF translates to each backend's convention using service-level configuration:

```yaml
services:
  orders-svc:
    pagination:
      style: "offset"
      page_param: "offset"       # backend's offset parameter name
      size_param: "limit"        # backend's limit parameter name
      # Translation: page=2, page_size=25 → offset=25, limit=25

  customers-svc:
    pagination:
      style: "page"
      page_param: "page"
      size_param: "per_page"
      # Translation: page=2, page_size=25 → page=2, per_page=25

  inventory-svc:
    pagination:
      style: "cursor"
      cursor_param: "cursor"
      size_param: "limit"
      # Translation: BFF manages cursor state (see below)

  ledger-svc:
    pagination:
      style: "offset"
      page_param: "skip"
      size_param: "take"
      # Translation: page=2, page_size=25 → skip=25, take=25
```

### Cursor Pagination Handling

For cursor-based backends, the BFF maintains a short-lived cursor cache:

```
1. Page 1 request: no cursor → send request without cursor.
   Backend returns: { data: [...], next_cursor: "eyJpZCI6MTI1fQ" }
   BFF caches: (user, page, filters) → { page 2 cursor: "eyJpZCI6MTI1fQ" }

2. Page 2 request: page=2
   BFF looks up cursor for page 2 in cache.
   Sends: ?cursor=eyJpZCI6MTI1fQ&limit=25
   Backend returns: { data: [...], next_cursor: "eyJpZCI6MjUwfQ" }
   BFF caches: page 3 cursor.

3. Cache TTL: 5 minutes. If expired, page > 1 requests without cache hit
   return an error asking the frontend to restart from page 1.
```

---

## Filtering Standardization

### Frontend Filter Format

```
?status=pending                  → exact match
?amount_gte=100                  → greater than or equal
?amount_lte=1000                 → less than or equal
?date_from=2025-01-01            → date range start
?date_to=2025-01-31              → date range end
?status=pending,confirmed        → in (comma-separated)
```

### Backend Translation

> **Implementation status:** Currently, filter field names and sort parameters are
> passed through to backends without translation. The `field_map` is applied only
> to **response** field names (renaming backend fields to UI fields). Request-side
> filter and sort field translation is planned.

The BFF translates filter parameters using the filter definitions:

```yaml
# Definition:
filters:
  - field: "status"
    operator: "eq"
# Frontend: ?status=pending
# Backend (orders-svc): ?status=pending  (same name, passthrough)
# Backend (other-svc): ?order_status=pending  (renamed via field_map)
```

If the definition uses `field_map`, filter field names are translated:
```yaml
mapping:
  field_map:
    status: "orderStatus"
# Frontend: ?status=pending → Backend: ?orderStatus=pending
```

---

## Sort Field Translation

Sort fields are also translated via `field_map`:

```
Frontend: ?sort=created_at&sort_dir=desc

Definition field_map:
  created_at: "createdAt"

Service pagination config:
  sort_param: "sort_by"
  sort_dir_param: "order"

Backend: ?sort_by=createdAt&order=desc
```

---

## Update Masks

> **Implementation status:** The `_changed_fields` mechanism and `X-Update-Mask`
> header generation described below are not yet implemented. Currently, all mapped
> fields are sent in PATCH requests regardless of which fields changed.

For PATCH operations where only changed fields should be sent:

### Frontend Sends Changed Fields

```json
POST /ui/commands/orders.update
{
  "input": {
    "shipping_address": "456 New St",
    "priority": "high"
  },
  "route_params": { "id": "ord-123" },
  "_changed_fields": ["shipping_address", "priority"]
}
```

### BFF Applies Update Mask

1. If `_changed_fields` is present: only include those fields in the backend body.
2. Translate field names via field_projection.
3. Optionally set an update mask header:

```
PATCH /api/v1/orders/ord-123
X-Update-Mask: shippingAddress,priority

{
  "shippingAddress": "456 New St",
  "priority": "high"
}
```

### When Update Masks Are Not Used

If `_changed_fields` is not provided, the BFF sends all mapped fields. The
backend is responsible for handling partial updates (most PATCH implementations
ignore null/missing fields by convention).

---

## Schema Validation Details

### When Validation Occurs

```
1. At startup: definitions validated against OpenAPI schemas (structural check).
2. At runtime: request bodies validated before sending to backend.
```

> **Note:** Runtime request validation currently checks required fields only. Full
> constraint validation (max length, enum, pattern, etc.) is not yet implemented.
> Response body validation is not currently available.

### Validation Error Format

When a request body fails OpenAPI schema validation:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "details": [
      {
        "field": "shipping_address",
        "code": "MAX_LENGTH",
        "message": "shipping_address must be at most 500 characters"
      },
      {
        "field": "priority",
        "code": "ENUM",
        "message": "priority must be one of: normal, high, urgent"
      }
    ]
  }
}
```

Note: field names in validation errors are translated back to UI field names
using the reverse field_map, so the frontend can highlight the correct form fields.
