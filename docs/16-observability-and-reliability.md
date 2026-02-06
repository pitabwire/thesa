# 16 — Observability and Reliability

This document describes logging, tracing, metrics, circuit breakers, retries, and
timeouts — everything needed to operate Thesa in production.

---

## Structured Logging

### Format

All logs are JSON-structured, written to stdout:

```json
{
  "level": "info",
  "msg": "command executed",
  "ts": "2025-01-15T10:30:00.123Z",
  "caller": "command/executor.go:142",
  "command_id": "orders.update",
  "tenant_id": "acme-corp",
  "partition_id": "us-production",
  "subject_id": "user-123",
  "correlation_id": "corr-456",
  "trace_id": "trace-789",
  "duration_ms": 145,
  "backend_service": "orders-svc",
  "backend_status": 200
}
```

### Standard Fields (every log entry in request scope)

| Field | Source |
|-------|--------|
| `tenant_id` | RequestContext |
| `subject_id` | RequestContext |
| `partition_id` | RequestContext |
| `correlation_id` | RequestContext |
| `trace_id` | OpenTelemetry span |

### Log Levels

| Level | Usage |
|-------|-------|
| `error` | Infrastructure failures (DB down, unhandled panics), 5xx responses |
| `warn` | Client errors (4xx), degraded operation (circuit breaker open), slow queries |
| `info` | Request start/end, command execution, workflow transitions, definition reload |
| `debug` | Cache operations, detailed input/output mapping, schema validation details |

### Sensitive Data Policy

- **Never log:** JWT tokens, passwords, API keys, credit card numbers.
- **Never log at INFO:** Request/response bodies (may contain PII).
- **Log at DEBUG only:** Request/response bodies with PII fields redacted.
- **Always log:** Correlation IDs, trace IDs, operation identifiers, durations, status codes.

---

## Distributed Tracing

### Integration

Thesa uses OpenTelemetry for distributed tracing. Every request creates a root span,
and nested spans are created for each significant operation.

### Span Hierarchy

```
Span: HTTP POST /ui/commands/orders.update
  │
  ├── Span: capability.resolve
  │     Attributes: cache_hit=true
  │
  ├── Span: command.execute (orders.update)
  │     Attributes: command_id=orders.update
  │     │
  │     ├── Span: input.mapping
  │     │     Attributes: body_mapping=projection
  │     │
  │     ├── Span: schema.validate
  │     │     Attributes: valid=true
  │     │
  │     ├── Span: backend.invoke (orders-svc/updateOrder)
  │     │     Attributes: service=orders-svc, operation=updateOrder, method=PATCH
  │     │     │
  │     │     └── Span: http.client.request
  │     │           Attributes: http.method=PATCH, http.url=https://orders.internal/...,
  │     │                       http.status_code=200, http.response_content_length=256
  │     │
  │     └── Span: output.mapping
  │           Attributes: strategy=project
  │
  └── (span ends with status OK)
```

### Context Propagation

Trace context is propagated to backend services via the W3C `traceparent` header:

```
traceparent: 00-{traceId}-{spanId}-01
```

Backend services that support OpenTelemetry will continue the trace, creating a
complete distributed trace from frontend → BFF → backend.

### Sampling

| Environment | Sampling Rate | Rationale |
|-------------|--------------|-----------|
| Development | 100% | Full visibility |
| Staging | 100% | Full visibility for testing |
| Production | 10-20% (adaptive) | Balance visibility vs. cost |
| On error | 100% (forced) | Always trace errors |

---

## Metrics

### Metric Types and Naming

Metrics follow the Prometheus naming convention: `thesa_{subsystem}_{name}_{unit}`.

### HTTP Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `thesa_http_requests_total` | Counter | method, path_pattern, status | Total HTTP requests |
| `thesa_http_request_duration_seconds` | Histogram | method, path_pattern | Request duration distribution |
| `thesa_http_request_size_bytes` | Histogram | method, path_pattern | Request body size |
| `thesa_http_response_size_bytes` | Histogram | method, path_pattern | Response body size |

`path_pattern` uses the route template (e.g., `/ui/commands/{commandId}`) not the
actual path, to prevent high cardinality.

### Command Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `thesa_command_executions_total` | Counter | command_id, status | Command execution count |
| `thesa_command_duration_seconds` | Histogram | command_id | Command execution duration |
| `thesa_command_validation_failures_total` | Counter | command_id | Schema validation failures |

### Workflow Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `thesa_workflow_starts_total` | Counter | workflow_id | Workflow starts |
| `thesa_workflow_advances_total` | Counter | workflow_id, step_id, event | Step advances |
| `thesa_workflow_completions_total` | Counter | workflow_id, final_status | Workflow completions |
| `thesa_workflow_active_instances` | Gauge | workflow_id | Currently active instances |
| `thesa_workflow_step_duration_seconds` | Histogram | workflow_id, step_id | Time spent in each step |
| `thesa_workflow_timeouts_total` | Counter | workflow_id | Timeout occurrences |

### Backend Invocation Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `thesa_backend_requests_total` | Counter | service_id, operation_id, status | Backend call count |
| `thesa_backend_request_duration_seconds` | Histogram | service_id | Backend call duration |
| `thesa_backend_circuit_breaker_state` | Gauge | service_id | 0=closed, 1=half-open, 2=open |
| `thesa_backend_retries_total` | Counter | service_id | Retry count |

### Cache Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `thesa_capability_cache_hits_total` | Counter | — | Capability cache hits |
| `thesa_capability_cache_misses_total` | Counter | — | Capability cache misses |
| `thesa_lookup_cache_hits_total` | Counter | lookup_id | Lookup cache hits |
| `thesa_lookup_cache_misses_total` | Counter | lookup_id | Lookup cache misses |

### System Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `thesa_definition_reload_total` | Counter | status | Definition reload count |
| `thesa_definitions_loaded` | Gauge | — | Number of loaded domain definitions |
| `thesa_openapi_operations_indexed` | Gauge | service_id | Indexed operations per service |
| `thesa_search_duration_seconds` | Histogram | — | Global search duration |
| `thesa_search_providers_responded` | Histogram | — | Number of providers that responded |

---

## Circuit Breakers

### Per-Service Circuit Breakers

Each backend service has an independent circuit breaker.

### State Machine

```
     success rate above threshold
  ┌─────────────────────────────────┐
  │                                 │
  ▼                                 │
┌──────┐  failure threshold    ┌────┴───┐   timeout     ┌───────────┐
│CLOSED│──────────────────────▶│ OPEN   │──────────────▶│HALF-OPEN  │
│      │                       │        │               │           │
│ Pass │                       │  Fail  │               │ Probe     │
│ all  │                       │  fast  │               │ single    │
│      │◀──────────────────────│        │◀──────────────│ request   │
└──────┘   probe succeeds      └────────┘  probe fails  └───────────┘
```

### Configuration

```yaml
services:
  orders-svc:
    circuit_breaker:
      failure_threshold: 5        # Consecutive failures to open
      success_threshold: 2        # Successes in half-open to close
      timeout: 30s                # How long to stay open before half-open
      # Alternative: error rate based
      error_rate_threshold: 0.5   # Open when error rate > 50%
      error_rate_window: 60s      # Window for error rate calculation
```

### What Counts as Failure

- Connection refused
- Connection timeout
- DNS resolution failure
- HTTP 500, 502, 503, 504 responses
- Response timeout

### What Does NOT Count as Failure

- HTTP 400, 401, 403, 404, 409, 422 responses (client errors)
- Successful responses (any 2xx)
- HTTP 429 (rate limited — not a service failure)

---

## Retries

### Retry Policy

```yaml
services:
  orders-svc:
    retry:
      max_attempts: 3             # Total attempts (including first)
      backoff_initial: 100ms      # First retry delay
      backoff_multiplier: 2       # Exponential multiplier
      backoff_max: 2s             # Maximum delay cap
      idempotent_only: true       # Only retry idempotent methods
```

### Retry Schedule

```
Attempt 1: immediate
  → failure
Attempt 2: wait 100ms
  → failure
Attempt 3: wait 200ms
  → failure
Give up, return error.
```

### Retryable Conditions

| Condition | Retryable? |
|-----------|-----------|
| Connection refused | Yes |
| Connection timeout | Yes |
| HTTP 502 | Yes |
| HTTP 503 | Yes |
| HTTP 504 | Yes |
| HTTP 500 | Yes (cautiously — may indicate deterministic failure) |
| HTTP 4xx | No |
| HTTP 2xx | No (success!) |
| Circuit breaker open | No (fail immediately) |

### POST Retry Safety

POST requests are NOT retried unless:
1. The command has idempotency configured AND an idempotency key is provided.
2. The error is clearly a network failure (connection reset before response received).

---

## Timeouts

### Three-Layer Timeout Model

```
┌──────────────────────────────────────────────┐
│ Layer 1: Client Timeout (30s)                 │
│  Flutter HTTP client waits up to 30s          │
│                                               │
│  ┌──────────────────────────────────────────┐ │
│  │ Layer 2: BFF Handler Timeout (25s)        │ │
│  │  Go context with deadline                 │ │
│  │                                           │ │
│  │  ┌──────────────────────────────────────┐ │ │
│  │  │ Layer 3: Backend Call Timeout (10s)   │ │ │
│  │  │  Per-service HTTP client timeout     │ │ │
│  │  └──────────────────────────────────────┘ │ │
│  └──────────────────────────────────────────┘ │
└──────────────────────────────────────────────┘
```

**Layer 1 > Layer 2 > Layer 3** to ensure each layer can timeout before the outer
layer, providing meaningful error messages instead of abrupt connection resets.

### Configuration

```yaml
server:
  read_timeout: 30s               # HTTP server read timeout
  write_timeout: 30s              # HTTP server write timeout
  handler_timeout: 25s            # Per-handler context deadline

services:
  orders-svc:
    timeout: 10s                  # Backend call timeout
  ledger-svc:
    timeout: 5s                   # Faster timeout for ledger
```

---

## Health Checks

### Liveness: GET /ui/health

Returns 200 if the process is running. Does NOT check dependencies.

```json
{ "status": "ok" }
```

Used by Kubernetes liveness probe. If this fails, the container is restarted.

### Readiness: GET /ui/ready

Returns 200 if the BFF can serve traffic. Checks all critical dependencies.

```json
{
  "status": "ready",
  "checks": {
    "definitions": "ok",
    "openapi_index": "ok",
    "workflow_store": "ok",
    "policy_engine": "ok"
  }
}
```

Used by Kubernetes readiness probe. If this fails, the pod is removed from the
service endpoints (no traffic routed to it).

### Startup: GET /ui/ready (with longer timeout)

Kubernetes startup probe uses the readiness endpoint with a longer timeout,
allowing time for initial definition loading and spec indexing.
