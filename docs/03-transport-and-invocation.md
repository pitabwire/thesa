# 03 — Transport and Invocation Model

This document describes both sides of Thesa's HTTP communication: the inbound
API that the frontend consumes, and the outbound invocation mechanisms used to
call backend services.

---

## Part I: Inbound — BFF HTTP API

### Design Philosophy

The BFF exposes a **small, fixed set of endpoints** that do NOT change when new
domains, services, or definitions are added. The frontend team learns these
endpoints once and builds their metadata-driven rendering engine around them.

Adding a new "orders" domain does not add new BFF endpoints. The orders domain's
pages, forms, and commands become accessible through the existing generic endpoints
(`/ui/pages/{pageId}`, `/ui/commands/{commandId}`, etc.).

### Complete Endpoint Catalog

| Method | Path | Purpose | Auth | Section |
|--------|------|---------|------|---------|
| GET | `/ui/navigation` | Menu tree for current user | Yes | [09](09-server-driven-ui-apis.md) |
| GET | `/ui/pages/{pageId}` | Page descriptor (metadata) | Yes | [09](09-server-driven-ui-apis.md) |
| GET | `/ui/pages/{pageId}/data` | Table/section data | Yes | [09](09-server-driven-ui-apis.md) |
| GET | `/ui/forms/{formId}` | Form descriptor (metadata) | Yes | [09](09-server-driven-ui-apis.md) |
| GET | `/ui/forms/{formId}/data` | Pre-populated form data | Yes | [09](09-server-driven-ui-apis.md) |
| POST | `/ui/commands/{commandId}` | Execute a command | Yes | [10](10-command-and-action-model.md) |
| POST | `/ui/workflows/{workflowId}/start` | Start a workflow | Yes | [11](11-workflow-engine.md) |
| POST | `/ui/workflows/{instanceId}/advance` | Advance a workflow | Yes | [11](11-workflow-engine.md) |
| GET | `/ui/workflows/{instanceId}` | Workflow instance state | Yes | [11](11-workflow-engine.md) |
| GET | `/ui/workflows` | List user's workflow instances | Yes | [11](11-workflow-engine.md) |
| POST | `/ui/workflows/{instanceId}/cancel` | Cancel a workflow | Yes | [11](11-workflow-engine.md) |
| GET | `/ui/search` | Global search | Yes | [12](12-global-search.md) |
| GET | `/ui/lookups/{lookupId}` | Reference data lookup | Yes | [09](09-server-driven-ui-apis.md) |
| GET | `/ui/health` | Health check | No | [22](22-deployment-and-operations.md) |
| GET | `/ui/ready` | Readiness check | No | [22](22-deployment-and-operations.md) |

### URL Design Rationale

- All endpoints are under `/ui/` to clearly namespace them as frontend-facing.
- Resource IDs (`pageId`, `formId`, `commandId`) are the IDs from definition files.
  They are domain-namespaced (e.g., `orders.list`, `orders.update`).
- Workflow endpoints use two ID types:
  - `workflowId` (e.g., `orders.approval`) — the definition ID, for starting new instances.
  - `instanceId` (e.g., `wf-a1b2c3`) — the unique instance ID, for advancing/querying.

### Request/Response Envelope

All responses (except health checks) use a consistent envelope:

**Success Response:**
```json
{
  "data": { ... },
  "meta": {
    "trace_id": "abc123",
    "timestamp": "2025-01-15T10:30:00Z"
  }
}
```

**Error Response:**
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "One or more fields are invalid",
    "details": [ ... ],
    "trace_id": "abc123"
  }
}
```

**Paginated Response (for data endpoints):**
```json
{
  "data": {
    "items": [ ... ],
    "total_count": 142,
    "page": 2,
    "page_size": 25
  },
  "meta": {
    "trace_id": "abc123"
  }
}
```

### Standard Query Parameters (Data Endpoints)

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `page` | int | Page number (1-based) | `?page=2` |
| `page_size` | int | Items per page (max 100, default from definition) | `?page_size=50` |
| `sort` | string | Field to sort by | `?sort=created_at` |
| `sort_dir` | string | Sort direction: `asc` or `desc` | `?sort_dir=desc` |
| `q` | string | Free-text search query | `?q=ORD-2024` |
| `{field}` | string | Filter by field value | `?status=pending` |
| `{field}_gte` | string | Filter: field >= value | `?amount_gte=100` |
| `{field}_lte` | string | Filter: field <= value | `?amount_lte=1000` |
| `{field}_from` | string | Date range start | `?date_from=2025-01-01` |
| `{field}_to` | string | Date range end | `?date_to=2025-01-31` |

### Standard Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | Yes (except health) | `Bearer {jwt_token}` |
| `X-Partition-Id` | Yes (except health) | Selected workspace/partition |
| `X-Correlation-Id` | No | Client-generated correlation ID; auto-generated if absent |
| `X-Device-Id` | No | Device fingerprint for session tracking |
| `X-Timezone` | No | Client timezone (e.g., "America/New_York") |
| `Accept-Language` | No | Locale preference (e.g., "en-US") |
| `Content-Type` | Yes (POST) | `application/json` |
| `Accept` | No | `application/json` or `application/vnd.thesa.v1+json` |
| `Idempotency-Key` | No | For command idempotency (see [10](10-command-and-action-model.md)) |

### CORS Configuration

The BFF configures CORS to allow only the known frontend origins:

```yaml
cors:
  allowed_origins:
    - "https://app.example.com"
    - "https://staging.app.example.com"
  allowed_methods: ["GET", "POST", "OPTIONS"]
  allowed_headers:
    - "Authorization"
    - "Content-Type"
    - "X-Partition-Id"
    - "X-Correlation-Id"
    - "X-Device-Id"
    - "X-Timezone"
    - "Idempotency-Key"
  expose_headers:
    - "X-Correlation-Id"
    - "X-Trace-Id"
  max_age: 3600
```

In production, wildcard (`*`) origins are never allowed.

---

## Part II: Outbound — Invocation Models

Thesa uses two invocation models to call backend services. Both satisfy the
same `OperationInvoker` interface, allowing definitions to reference either
model interchangeably.

### Model 1: OpenAPI-Driven Dynamic Invocation (Primary)

This is the default and most common invocation model. The BFF builds HTTP
requests at runtime from indexed OpenAPI specifications, without any code
generation or compilation.

#### When to Use

- For standard CRUD operations (list, get, create, update, delete).
- For any synchronous request/response API documented with OpenAPI.
- For the vast majority of backend interactions.

#### When NOT to Use

- For streaming APIs (WebSockets, Server-Sent Events, bidirectional gRPC streams).
- For operations requiring complex multi-step orchestration within a single invocation.
- For high-integrity paths where compile-time type safety of the request/response
  is required (e.g., ledger entries where a wrong field could cause financial errors).

#### How It Works

**Startup:**

```
1. For each service in configuration:
   a. Read the OpenAPI spec file (YAML or JSON).
   b. Parse the spec using the OpenAPI library (e.g., kin-openapi).
   c. Extract all operations (all paths × all methods).
   d. For each operation:
      - Extract operationId (required — specs without operationId are rejected).
      - Extract method, path template, parameters, request body schema, response schemas.
      - Store in the OpenAPIIndex keyed by (serviceId, operationId).
   e. Log: "Loaded {N} operations from {service} spec"

2. Validation phase:
   - For each definition that references an OpenAPI operation:
     a. Verify (service_id, operation_id) exists in the index.
     b. Verify input mapping references valid parameters.
     c. Log warnings if response mapping paths don't match response schema.
```

**Runtime invocation (per request):**

```
1. Look up operation in OpenAPIIndex by (serviceId, operationId).
2. Get service base URL from configuration.
3. Build the URL:
   a. Substitute path parameters: "/orders/{orderId}" → "/orders/ord-123"
   b. Append query parameters: "?limit=25&offset=0&sort_by=createdAt"
4. Build headers:
   a. Content-Type: application/json (if body present)
   b. Accept: application/json
   c. Authorization: Bearer {forwarded or exchanged token}
   d. X-Tenant-Id: {from RequestContext}
   e. X-Partition-Id: {from RequestContext}
   f. X-Correlation-Id: {from RequestContext}
   g. traceparent: {from current OpenTelemetry span}
   h. X-Request-Subject: {from RequestContext}
5. Serialize request body to JSON (if present).
6. Validate request body against OpenAPI schema (optional, recommended).
7. Send HTTP request via connection pool (per service):
   a. Apply circuit breaker check.
   b. Apply timeout.
   c. If failure and method is idempotent: retry with exponential backoff.
8. Parse response:
   a. Read HTTP status code.
   b. Parse JSON response body.
   c. Package as InvocationResult { StatusCode, Body, Headers }.
9. Return to caller (CommandExecutor, PageProvider, WorkflowEngine, etc.).
```

#### Connection Management

Each backend service gets its own HTTP connection pool:

```yaml
# Per-service HTTP client configuration
services:
  orders-svc:
    http_client:
      max_idle_conns: 100
      max_conns_per_host: 50
      idle_conn_timeout: 90s
      tls_handshake_timeout: 10s
      response_header_timeout: 10s
```

Connection pools are reused across requests for efficiency.

#### Error Handling in the Invoker

The OpenAPI invoker returns the raw status code and body. Error interpretation
is the responsibility of the caller (CommandExecutor, PageProvider, etc.).

However, the invoker handles transport-level errors:
- **Connection refused:** Returns error with code `BACKEND_UNREACHABLE`.
- **Timeout:** Returns error with code `BACKEND_TIMEOUT`.
- **Circuit breaker open:** Returns error with code `BACKEND_CIRCUIT_OPEN`.
- **DNS failure:** Returns error with code `BACKEND_UNREACHABLE`.

### Model 2: SDK / Typed Client Invocation (Secondary)

For cases where dynamic HTTP invocation is insufficient, Thesa supports
pluggable, typed Go clients.

#### When to Use

- **Streaming APIs:** gRPC server-streaming, bidirectional streaming, SSE.
- **Complex multi-call orchestration:** A single "operation" that requires calling
  multiple backend endpoints in a specific order with transactional semantics.
- **High-integrity paths:** Operations where the request/response types must be
  verified at compile time (e.g., ledger entries where a wrong field name would
  cause financial errors).
- **Non-HTTP protocols:** gRPC, AMQP, Kafka producers, or any other protocol.

#### How It Works

**Registration (at startup in main.go):**

SDK handlers are Go types that implement the `SDKHandler` interface. They are
registered in the `SDKInvokerRegistry` at application startup:

```
SDKInvokerRegistry.Register("ledger.PostEntry", ledgerPostEntryHandler)
SDKInvokerRegistry.Register("notifications.Send", notificationSendHandler)
SDKInvokerRegistry.Register("streaming.OrderUpdates", orderStreamHandler)
```

Each handler contains the typed client (e.g., a gRPC client stub, a Connect RPC
client) and implements the invocation logic.

**Runtime invocation:**

When a definition references an SDK handler:

```yaml
operation:
  type: "sdk"
  handler: "ledger.PostEntry"
  service_id: "ledger-svc"
```

The invocation flow is:
1. The `SDKOperationInvoker` receives the invocation request.
2. It looks up "ledger.PostEntry" in the SDKInvokerRegistry.
3. It calls the handler's `Invoke()` method with the RequestContext and input.
4. The handler constructs a typed request, calls the backend using its typed client,
   and returns a typed response packaged as an `InvocationResult`.

#### Trade-offs

| Aspect | OpenAPI Invoker | SDK Invoker |
|--------|----------------|-------------|
| Adding new operations | YAML only, no recompilation | Go code, requires recompilation |
| Type safety | Runtime schema validation | Compile-time type checking |
| Protocol support | HTTP only | Any protocol |
| Streaming | Not supported | Supported |
| Complexity | Low (declarative) | Higher (imperative) |
| Testing | Requires running backend or mock server | Can unit test with interface mocks |

### Unified Invocation Abstraction

Both models satisfy the same interface:

```
OperationInvoker
  ├── Invoke(ctx, requestContext, binding, input) → (InvocationResult, error)
  └── Supports(binding) → bool
```

The `InvokerRegistry` holds all registered invokers and dispatches:

```
InvokerRegistry.Invoke(ctx, requestContext, binding, input):
  1. Iterate registered invokers.
  2. Find the first invoker where Supports(binding) returns true.
  3. Call Invoke() on that invoker.
  4. Return result.
```

In practice:
- `OpenAPIOperationInvoker.Supports()` returns true when `binding.Type == "openapi"`.
- `SDKOperationInvoker.Supports()` returns true when `binding.Type == "sdk"`.

This unified abstraction means that definition authors don't need to think about
the invocation mechanism. They specify `type: "openapi"` or `type: "sdk"`, and
the rest is handled by the registry.

---

## Part III: Middleware Pipeline

The BFF processes every inbound request through a middleware chain before
reaching the handler:

```
Request
  │
  ├── 1. Recovery Middleware
  │      Catches panics, returns 500, logs stack trace.
  │
  ├── 2. Request ID Middleware
  │      Generates or accepts X-Correlation-Id.
  │
  ├── 3. Logging Middleware
  │      Logs request start and completion with duration.
  │
  ├── 4. Metrics Middleware
  │      Records request count and duration histogram.
  │
  ├── 5. Tracing Middleware
  │      Creates OpenTelemetry span, propagates context.
  │
  ├── 6. CORS Middleware
  │      Handles preflight OPTIONS requests, sets CORS headers.
  │
  ├── 7. Authentication Middleware
  │      Verifies JWT, extracts claims. Returns 401 if invalid.
  │
  ├── 8. Context Construction Middleware
  │      Builds RequestContext from JWT claims and headers.
  │      Validates partition belongs to tenant.
  │
  ├── 9. Capability Resolution Middleware (optional — can be lazy)
  │      Calls CapabilityResolver, attaches CapabilitySet to context.
  │
  └── 10. Handler
         Business logic: page resolution, command execution, etc.
```

### Lazy vs. Eager Capability Resolution

Capabilities can be resolved eagerly (in middleware, for every request) or lazily
(on demand, when a handler needs them).

**Eager (middleware-based):**
- Simpler handler code — capabilities are always available.
- Every request pays the resolution cost, even health checks or cache hits.
- Best for systems where most requests need capabilities.

**Lazy (on-demand):**
- Capabilities resolved only when needed.
- Handlers explicitly call `CapabilityResolver.Resolve()` when needed.
- Better for read-heavy workloads where some requests (data fetching with
  pre-validated page) don't need fresh capability evaluation.

**Recommended approach:** Eager resolution with caching. The cache (TTL: 60s)
makes the per-request cost minimal, and eager resolution simplifies handler code.

---

## Part IV: Request/Response Types

### InvocationInput (BFF → Backend)

```
InvocationInput
  ├── PathParams    map[string]string    // Substituted into URL path template
  ├── QueryParams   map[string]string    // Appended as URL query parameters
  ├── Headers       map[string]string    // Additional HTTP headers
  └── Body          any                  // Request body (serialized to JSON)
```

### InvocationResult (Backend → BFF)

```
InvocationResult
  ├── StatusCode    int                  // HTTP status code
  ├── Body          any                  // Parsed JSON response body
  └── Headers       map[string]string    // Response headers of interest
```

### CommandInput (Frontend → BFF)

```
POST /ui/commands/{commandId}

{
  "input": {                           // User-provided payload
    "shipping_address": "123 Main St",
    "priority": "high"
  },
  "route_params": {                    // Current route parameters
    "id": "ord-123"
  },
  "idempotency_key": "abc-123"        // Optional
}
```

### CommandResponse (BFF → Frontend)

```
{
  "data": {
    "success": true,
    "message": "Order updated successfully",
    "result": {
      "id": "ord-123",
      "order_number": "ORD-2024-001"
    }
  },
  "meta": {
    "trace_id": "trace-xyz"
  }
}
```

---

## Part V: Connection to Backend Services

### HTTP Client Configuration

The BFF maintains a configurable HTTP client per backend service. This allows
per-service tuning of timeouts, connection limits, and TLS settings.

| Setting | Default | Description |
|---------|---------|-------------|
| `timeout` | 10s | Total request timeout (includes connection, TLS, response) |
| `max_idle_conns` | 100 | Maximum idle connections in pool |
| `max_conns_per_host` | 50 | Maximum total connections to this service |
| `idle_conn_timeout` | 90s | How long to keep idle connections open |
| `tls_handshake_timeout` | 10s | TLS handshake timeout |
| `disable_keepalive` | false | Whether to disable HTTP keep-alive |

### DNS Resolution

For Kubernetes deployments, backend services are resolved via DNS (e.g.,
`orders-svc.namespace.svc.cluster.local`). DNS TTL should be respected;
the Go HTTP client's default resolver caches for the DNS TTL.

For services behind a load balancer, a single DNS entry resolves to the LB IP.
For services using client-side load balancing, the DNS entry resolves to
multiple IPs and the HTTP client distributes connections.
