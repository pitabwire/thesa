# 22 — Deployment and Operations

This document covers everything needed to build, deploy, configure, and operate
Thesa in production. It describes the startup sequence, graceful shutdown,
configuration management, health checks, scaling considerations, CI/CD
integration, and operational runbooks.

---

## Build and Packaging

### Binary Build

Thesa compiles to a single statically-linked Go binary:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
  -o thesa-bff ./cmd/bff
```

Build flags:
- `CGO_ENABLED=0`: Pure Go binary, no C dependencies. Required for scratch/distroless containers.
- `-ldflags="-s -w"`: Strip debug info and DWARF tables (smaller binary).
- `-X main.version/commit`: Embed version info at build time for `/ui/health` and logging.

### Container Image

```dockerfile
# Stage 1: Build
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /thesa-bff ./cmd/bff

# Stage 2: Runtime
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /thesa-bff /thesa-bff
COPY definitions/ /definitions/
COPY specs/ /specs/
COPY config/ /config/
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/thesa-bff"]
```

Image characteristics:
- **Base image:** `distroless/static` — no shell, no package manager, minimal attack surface.
- **User:** `nonroot` — never runs as root.
- **Read-only filesystem:** Kubernetes can mount the filesystem as read-only (`readOnlyRootFilesystem: true`).
- **Size:** Typically 15-25 MB (Go binary + distroless base).

### Artifact Layout

```
Container filesystem:
  /thesa-bff                    # Binary
  /definitions/                 # Definition YAML files
    orders/definition.yaml
    inventory/definition.yaml
    customers/definition.yaml
  /specs/                       # OpenAPI specification files
    orders-svc.yaml
    inventory-svc.yaml
    customers-svc.yaml
  /config/                      # Configuration files
    config.yaml
    config.production.yaml
```

In production, definitions and specs may be mounted from separate sources
(ConfigMaps, shared volumes, or artifact stores) rather than baked into
the image, depending on the deployment strategy.

---

## Startup Sequence

The startup sequence is ordered to ensure fail-fast behavior. If any critical
step fails, the process exits immediately with a non-zero status code. This
prevents a misconfigured BFF from ever serving traffic.

```
┌─────────────────────────────────────────────────────────┐
│  1. Load Configuration                                   │
│     Parse config.yaml + environment overrides             │
│     Validate required fields                              │
│     FAIL → exit 1 (invalid configuration)                │
├─────────────────────────────────────────────────────────┤
│  2. Initialize Telemetry                                  │
│     Set up structured logging (zap)                       │
│     Initialize OpenTelemetry tracer provider              │
│     Register Prometheus metrics                           │
│     FAIL → exit 1 (telemetry is required)                │
├─────────────────────────────────────────────────────────┤
│  3. Load OpenAPI Specifications                           │
│     Parse all spec files in specs/ directory              │
│     Build OpenAPIIndex                                    │
│     FAIL → exit 1 (specs are required for validation)    │
├─────────────────────────────────────────────────────────┤
│  4. Load UI Definitions                                   │
│     Parse all definition YAML files                       │
│     Validate structural integrity                         │
│     Validate against OpenAPIIndex (operation IDs exist)   │
│     Compute SHA-256 checksums                             │
│     Build DefinitionRegistry                              │
│     FAIL → exit 1 (invalid definitions)                  │
├─────────────────────────────────────────────────────────┤
│  5. Initialize Capability System                          │
│     Connect to policy engine (OPA) or load static policy  │
│     Verify connectivity (health check on OPA)             │
│     Initialize capability cache                           │
│     FAIL → exit 1 (auth system unreachable)              │
├─────────────────────────────────────────────────────────┤
│  6. Initialize Workflow Store                             │
│     Connect to PostgreSQL                                 │
│     Run connection health check (ping)                    │
│     Run migrations (if auto-migrate enabled)              │
│     FAIL → exit 1 (database unreachable)                 │
│     SKIP if workflow feature disabled                     │
├─────────────────────────────────────────────────────────┤
│  7. Initialize Idempotency Store                          │
│     Connect to Redis (or PostgreSQL)                      │
│     FAIL → exit 1 (idempotency requires shared state)    │
│     SKIP if idempotency feature disabled                  │
├─────────────────────────────────────────────────────────┤
│  8. Register SDK Handlers                                 │
│     Instantiate typed client handlers                     │
│     Register in SDKInvokerRegistry                        │
│     WARN if expected handler is missing (non-fatal)       │
├─────────────────────────────────────────────────────────┤
│  9. Build Service Layer                                   │
│     Instantiate InvokerRegistry (OpenAPI + SDK invokers)  │
│     Instantiate MenuProvider, PageProvider, FormProvider   │
│     Instantiate CommandExecutor                           │
│     Instantiate WorkflowEngine                            │
│     Instantiate SearchProvider                            │
│     Wire all dependencies                                 │
├─────────────────────────────────────────────────────────┤
│  10. Build HTTP Router                                    │
│      Register all routes with chi router                  │
│      Attach middleware chain:                             │
│        Recovery → CORS → RequestID → Auth → Context →    │
│        Timeout → Logging → Metrics → Tracing             │
├─────────────────────────────────────────────────────────┤
│  11. Start Background Tasks                               │
│      Start workflow timeout processor (goroutine)         │
│      Start definition file watcher (if hot-reload on)     │
│      Start capability cache cleanup (goroutine)           │
├─────────────────────────────────────────────────────────┤
│  12. Start HTTP Server                                    │
│      Listen on configured port (default: 8080)            │
│      Log: "server started" with port, version, commit     │
├─────────────────────────────────────────────────────────┤
│  13. Report Healthy                                       │
│      /ui/health returns 200                               │
│      /ui/ready returns 200 (all checks pass)              │
│      Kubernetes readiness probe succeeds                  │
│      Pod receives traffic                                 │
└─────────────────────────────────────────────────────────┘
```

### Startup Timing

| Step | Typical Duration | Notes |
|------|-----------------|-------|
| Configuration | < 10ms | File read + parse |
| Telemetry | 50-100ms | OTLP exporter connection |
| OpenAPI specs | 100-500ms | Depends on spec count and size |
| Definitions | 50-200ms | Depends on definition count |
| Policy engine | 100-500ms | Network round-trip to OPA |
| Workflow store | 100-300ms | PostgreSQL connection + ping |
| Idempotency store | 50-100ms | Redis connection + ping |
| Router + middleware | < 10ms | In-memory setup |
| **Total cold start** | **~500ms - 2s** | Dominated by network I/O |

---

## Graceful Shutdown

When the process receives `SIGTERM` or `SIGINT`, it initiates a graceful
shutdown to avoid dropping in-flight requests.

```
┌─────────────────────────────────────────────────────────┐
│  Signal Received: SIGTERM                                │
├─────────────────────────────────────────────────────────┤
│  1. Log: "shutdown initiated"                            │
│                                                          │
│  2. Stop accepting new connections                       │
│     HTTP server stops listening on port                   │
│     Kubernetes removes pod from endpoints (in parallel)   │
│                                                          │
│  3. Wait for in-flight requests to complete               │
│     Deadline: 30s (configurable via shutdown_timeout)     │
│     If deadline exceeded: force-close remaining conns     │
│                                                          │
│  4. Stop background tasks                                │
│     Cancel workflow timeout processor context              │
│     Stop definition file watcher                          │
│     Wait for goroutines to exit (5s max)                  │
│                                                          │
│  5. Close external connections                            │
│     Close WorkflowStore (PostgreSQL pool)                 │
│     Close idempotency store (Redis)                       │
│     Close HTTP client pools (backend connections)         │
│                                                          │
│  6. Flush telemetry                                      │
│     Flush OpenTelemetry span exporter (5s timeout)        │
│     Flush log buffers                                     │
│                                                          │
│  7. Log: "shutdown complete"                              │
│     Exit 0                                                │
└─────────────────────────────────────────────────────────┘
```

### Kubernetes Integration

```yaml
# Pod spec
spec:
  terminationGracePeriodSeconds: 45  # Must be > shutdown_timeout (30s)
  containers:
    - name: thesa-bff
      lifecycle:
        preStop:
          exec:
            command: ["sleep", "5"]   # Allow endpoint removal to propagate
```

The `preStop` sleep ensures that Kubernetes has time to remove the pod from
service endpoints before the server stops accepting connections. Without this,
clients may still route to the pod after it begins shutdown.

Timeline:

```
T+0:    SIGTERM received
T+0-5s: preStop sleep (pod still accepts traffic, but k8s is removing endpoints)
T+5s:   Server stops accepting new connections
T+5-35s: Drain in-flight requests
T+35s:  Force close, flush, exit
T+45s:  Kubernetes forcefully kills the pod (if still running)
```

---

## Configuration

### Configuration File Structure

```yaml
# config/config.yaml

server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  handler_timeout: 25s
  shutdown_timeout: 30s
  cors:
    allowed_origins:
      - "https://app.example.com"
      - "https://staging.example.com"
    allowed_methods: ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"]
    allowed_headers: ["Authorization", "Content-Type", "X-Partition-Id",
                       "X-Correlation-Id", "X-Idempotency-Key"]
    max_age: 86400

identity:
  issuer: "https://auth.example.com"
  audience: "thesa-bff"
  jwks_url: "https://auth.example.com/.well-known/jwks.json"
  jwks_cache_ttl: 1h
  algorithms: ["RS256", "ES256"]
  claim_paths:
    subject_id: "sub"
    tenant_id: "custom:tenant_id"
    email: "email"
    roles: "custom:roles"
    partition_id: "custom:default_partition"

definitions:
  directories:
    - "/definitions"
  hot_reload: false           # Set to true in development
  strict_checksums: true      # Verify against manifest in production

specs:
  directory: "/specs"
  sources:
    - service_id: "orders-svc"
      spec_file: "orders-svc.yaml"
    - service_id: "inventory-svc"
      spec_file: "inventory-svc.yaml"
    - service_id: "customers-svc"
      spec_file: "customers-svc.yaml"

services:
  orders-svc:
    base_url: "https://orders.internal"
    timeout: 10s
    auth:
      strategy: "forward_token"
    pagination:
      style: "offset"
      page_param: "offset"
      size_param: "limit"
      sort_param: "sort_by"
      sort_dir_param: "order"
    circuit_breaker:
      failure_threshold: 5
      success_threshold: 2
      timeout: 30s
    retry:
      max_attempts: 3
      backoff_initial: 100ms
      backoff_multiplier: 2
      backoff_max: 2s
      idempotent_only: true

  inventory-svc:
    base_url: "https://inventory.internal"
    timeout: 8s
    auth:
      strategy: "forward_token"
    pagination:
      style: "page"
      page_param: "page"
      size_param: "per_page"
    circuit_breaker:
      failure_threshold: 5
      timeout: 30s
    retry:
      max_attempts: 3
      backoff_initial: 100ms

  customers-svc:
    base_url: "https://customers.internal"
    timeout: 10s
    auth:
      strategy: "service_token"
      client_id: "thesa-bff"
      token_endpoint: "https://auth.internal/oauth/token"
    pagination:
      style: "cursor"
      cursor_param: "after"
      size_param: "first"

capability:
  evaluator: "static"         # "static" | "opa"
  static_policy_file: "/config/policies.yaml"
  cache:
    ttl: 5m
    max_entries: 10000
  # OPA configuration (when evaluator: "opa"):
  # opa:
  #   url: "https://opa.internal:8181"
  #   policy_path: "/v1/data/thesa/capabilities"
  #   timeout: 2s

workflow:
  enabled: true
  store:
    driver: "postgres"
    dsn_env: "THESA_WORKFLOW_DSN"    # Read DSN from environment variable
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 5m
  timeout_check_interval: 60s

idempotency:
  enabled: true
  store:
    driver: "redis"
    addr_env: "THESA_REDIS_ADDR"
    db: 0
    default_ttl: 24h

search:
  timeout_per_provider: 3s
  max_results_per_provider: 50

lookup:
  cache:
    ttl: 300s
    max_entries: 1000

observability:
  log_level: "info"           # debug | info | warn | error
  tracing:
    enabled: true
    exporter: "otlp"          # "otlp" | "jaeger" | "stdout"
    endpoint: "otel-collector.internal:4317"
    sampling_rate: 0.1        # 10% in production
    force_sample_errors: true
  metrics:
    enabled: true
    path: "/metrics"
```

### Environment Variable Overrides

Every configuration value can be overridden by an environment variable using
the `THESA_` prefix with underscore-separated path:

| Config Path | Environment Variable |
|------------|---------------------|
| `server.port` | `THESA_SERVER_PORT` |
| `server.handler_timeout` | `THESA_SERVER_HANDLER_TIMEOUT` |
| `identity.issuer` | `THESA_IDENTITY_ISSUER` |
| `identity.jwks_url` | `THESA_IDENTITY_JWKS_URL` |
| `services.orders-svc.base_url` | `THESA_SERVICES_ORDERS_SVC_BASE_URL` |
| `workflow.store.dsn_env` | Points to another env var containing the DSN |
| `observability.log_level` | `THESA_OBSERVABILITY_LOG_LEVEL` |

**Priority (highest wins):**

```
1. Environment variables (THESA_*)
2. Config file specified by --config flag
3. config/config.yaml (default path)
4. Compiled defaults
```

### Secrets Management

Secrets are NEVER stored in configuration files. They are injected via:

| Secret | Injection Method |
|--------|-----------------|
| PostgreSQL DSN | Environment variable: `THESA_WORKFLOW_DSN` |
| Redis address | Environment variable: `THESA_REDIS_ADDR` |
| Service token client secret | Environment variable: `THESA_SERVICE_TOKEN_SECRET` |
| mTLS certificates | Mounted files: `/certs/client.crt`, `/certs/client.key` |

In Kubernetes:

```yaml
envFrom:
  - secretRef:
      name: thesa-secrets
env:
  - name: THESA_WORKFLOW_DSN
    valueFrom:
      secretKeyRef:
        name: thesa-db-credentials
        key: dsn
```

---

## Health Checks

### Liveness Probe: `GET /ui/health`

Returns 200 if the process is running and can handle HTTP requests. This check
has no dependencies — it tests only that the HTTP server is responsive.

```json
{
  "status": "ok",
  "version": "1.2.3",
  "commit": "abc1234"
}
```

**When it fails:**
- Process is deadlocked
- HTTP server crashed
- Memory exhaustion

**Kubernetes action on failure:** Restart the container.

```yaml
livenessProbe:
  httpGet:
    path: /ui/health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 3
  failureThreshold: 3
```

### Readiness Probe: `GET /ui/ready`

Returns 200 if the BFF can serve traffic. Checks all critical dependencies.

```json
{
  "status": "ready",
  "checks": {
    "definitions": { "status": "ok", "count": 3 },
    "openapi_index": { "status": "ok", "operations": 21 },
    "workflow_store": { "status": "ok", "latency_ms": 2 },
    "policy_engine": { "status": "ok", "latency_ms": 5 },
    "idempotency_store": { "status": "ok", "latency_ms": 1 }
  }
}
```

**Degraded response (still serving but with issues):**

```json
{
  "status": "degraded",
  "checks": {
    "definitions": { "status": "ok", "count": 3 },
    "openapi_index": { "status": "ok", "operations": 21 },
    "workflow_store": { "status": "error", "error": "connection refused" },
    "policy_engine": { "status": "ok", "latency_ms": 5 },
    "idempotency_store": { "status": "ok", "latency_ms": 1 }
  }
}
```

Whether "degraded" returns 200 or 503 depends on policy. If workflows are not
critical for all traffic, a degraded state may still serve 200 for readiness.

**Kubernetes action on failure (503):** Remove pod from service endpoints (no traffic).

```yaml
readinessProbe:
  httpGet:
    path: /ui/ready
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 15
  timeoutSeconds: 5
  failureThreshold: 2

startupProbe:
  httpGet:
    path: /ui/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 10
  failureThreshold: 30    # Allow up to 150s for startup
```

### Metrics Endpoint: `GET /metrics`

Prometheus metrics in standard exposition format. This endpoint does NOT
require authentication (it is typically exposed only to the internal
monitoring network, not through the public ingress).

---

## Kubernetes Deployment

### Deployment Manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: thesa-bff
  labels:
    app: thesa-bff
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0      # Zero-downtime deployments
  selector:
    matchLabels:
      app: thesa-bff
  template:
    metadata:
      labels:
        app: thesa-bff
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      terminationGracePeriodSeconds: 45
      serviceAccountName: thesa-bff
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
        fsGroup: 65534
      containers:
        - name: thesa-bff
          image: registry.example.com/thesa-bff:1.2.3
          ports:
            - containerPort: 8080
              name: http
              protocol: TCP
          envFrom:
            - configMapRef:
                name: thesa-config
            - secretRef:
                name: thesa-secrets
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 256Mi
          livenessProbe:
            httpGet:
              path: /ui/health
              port: http
            initialDelaySeconds: 5
            periodSeconds: 10
            timeoutSeconds: 3
          readinessProbe:
            httpGet:
              path: /ui/ready
              port: http
            initialDelaySeconds: 10
            periodSeconds: 15
            timeoutSeconds: 5
          startupProbe:
            httpGet:
              path: /ui/ready
              port: http
            periodSeconds: 5
            failureThreshold: 30
          securityContext:
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
            capabilities:
              drop: ["ALL"]
          volumeMounts:
            - name: definitions
              mountPath: /definitions
              readOnly: true
            - name: specs
              mountPath: /specs
              readOnly: true
            - name: tmp
              mountPath: /tmp
          lifecycle:
            preStop:
              exec:
                command: ["sleep", "5"]
      volumes:
        - name: definitions
          configMap:
            name: thesa-definitions
        - name: specs
          configMap:
            name: thesa-specs
        - name: tmp
          emptyDir:
            sizeLimit: 10Mi
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: thesa-bff
spec:
  selector:
    app: thesa-bff
  ports:
    - port: 80
      targetPort: http
      protocol: TCP
  type: ClusterIP
```

### Horizontal Pod Autoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: thesa-bff
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: thesa-bff
  minReplicas: 3
  maxReplicas: 20
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Pods
      pods:
        metric:
          name: thesa_http_request_duration_seconds
        target:
          type: AverageValue
          averageValue: "500m"    # Scale up if p50 latency exceeds 500ms
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
        - type: Pods
          value: 2
          periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Pods
          value: 1
          periodSeconds: 120
```

### Pod Disruption Budget

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: thesa-bff
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: thesa-bff
```

---

## Multi-Instance Considerations

### Shared State

When running multiple BFF instances behind a load balancer:

| Component | Sharing Strategy | Notes |
|-----------|-----------------|-------|
| Definition registry | Each instance loads independently | Must load same definitions; coordinate via artifact version |
| OpenAPI index | Each instance loads independently | Same specs = same index |
| Capability cache | Per-instance (in-memory) | TTL ensures eventual consistency |
| Workflow store | Shared PostgreSQL | Optimistic locking prevents conflicts |
| Idempotency store | Shared Redis/PostgreSQL | MUST be shared for correctness |
| Circuit breaker state | Per-instance | Intentional: each instance tracks its own view of backend health |

### Definition Consistency

All instances MUST serve the same definitions at any point in time (eventual
consistency within seconds is acceptable).

**Strategies:**

1. **Baked into image** (recommended for production):
   - Definitions are part of the container image.
   - All instances of the same image version have the same definitions.
   - Rolling updates ensure gradual transition.

2. **ConfigMap mount:**
   - Definitions are in a Kubernetes ConfigMap.
   - ConfigMap updates propagate to all pods (with a delay of up to kubelet sync period).
   - If hot-reload is enabled, all instances pick up changes within the sync period + debounce time.

3. **Shared filesystem (NFS/EFS):**
   - All instances mount the same filesystem.
   - File watcher detects changes on all instances simultaneously.
   - Risk: filesystem latency and availability becomes a dependency.

### Session Affinity

Thesa is **stateless** from the HTTP perspective. No session affinity is required.
Any instance can serve any request. The only state is in the shared stores
(PostgreSQL for workflows, Redis for idempotency).

---

## CI/CD Integration

### Definition Validation Pipeline

Domain teams maintain definitions in their own repositories. A CI pipeline
validates definitions before they can be deployed:

```
┌─────────────────────────────────────────────────┐
│  Domain Team CI (e.g., Orders team)              │
│                                                   │
│  1. Author/modify definitions/orders/*.yaml       │
│  2. CI step: validate-definitions                 │
│     a. Parse YAML (structural validation)         │
│     b. Load OpenAPI specs (from artifact store)    │
│     c. Validate operation_id references            │
│     d. Validate capability namespaces              │
│     e. Check for breaking changes (diff-based)     │
│  3. CI step: compute checksum                     │
│     a. SHA-256 of each definition file             │
│     b. Sign manifest with CI signing key           │
│  4. Publish definition artifact                    │
│     a. Upload to artifact store (versioned)        │
│     b. Include signed manifest                     │
│                                                   │
│  FAIL at step 2 → block merge                     │
└─────────────────────────────────────────────────┘
```

### BFF Deployment Pipeline

```
┌─────────────────────────────────────────────────┐
│  BFF CI/CD                                        │
│                                                   │
│  1. Build binary                                  │
│  2. Run unit tests                                │
│  3. Run integration tests                         │
│     a. Load test definitions                      │
│     b. Start BFF with in-memory stores            │
│     c. Exercise all endpoints                     │
│     d. Verify capability filtering                │
│     e. Verify command pipeline                    │
│  4. Pull latest definition artifacts              │
│  5. Build container image                         │
│  6. Scan image (Trivy, Snyk)                      │
│  7. Sign image (cosign)                           │
│  8. Push to registry                              │
│  9. Deploy to staging                             │
│  10. Run smoke tests against staging              │
│  11. Deploy to production (canary → full)         │
│                                                   │
│  FAIL at any step → stop pipeline                 │
└─────────────────────────────────────────────────┘
```

### Canary Deployment Strategy

```
Phase 1: 5% traffic (1 canary pod, rest on old version)
  Monitor for 10 minutes:
    - Error rate < 1%
    - p99 latency < 2s
    - No 5xx spike
  → Proceed or rollback

Phase 2: 25% traffic
  Monitor for 10 minutes
  → Proceed or rollback

Phase 3: 100% traffic (full rollout)
  Monitor for 30 minutes
  → Mark deployment as successful
```

---

## Operational Runbooks

### Runbook: High Error Rate

**Alert:** `thesa_http_requests_total` with status 5xx exceeds threshold.

```
1. Check which endpoints are failing:
     Query: rate(thesa_http_requests_total{status=~"5.."}[5m]) by (path_pattern)

2. If a specific backend is failing:
     Check: thesa_backend_circuit_breaker_state{service_id="..."}
     If circuit breaker is OPEN:
       → Backend service is down
       → Check backend service health
       → Circuit breaker will auto-recover when backend is healthy

3. If errors are across all endpoints:
     Check BFF logs for panic/crash:
       kubectl logs -l app=thesa-bff --since=10m | grep '"level":"error"'
     Check pod restarts:
       kubectl get pods -l app=thesa-bff
     Check resource limits:
       kubectl top pods -l app=thesa-bff

4. If errors are on definition-related endpoints:
     Check recent definition changes:
       thesa_definition_reload_total{status="failure"}
     Verify definition integrity:
       kubectl exec ... -- /thesa-bff --validate-definitions
```

### Runbook: High Latency

**Alert:** `thesa_http_request_duration_seconds` p99 exceeds threshold.

```
1. Check backend latency:
     Query: histogram_quantile(0.99, rate(thesa_backend_request_duration_seconds_bucket[5m])) by (service_id)
     → If a specific backend is slow, the issue is upstream

2. Check capability cache:
     Query: rate(thesa_capability_cache_misses_total[5m])
     → High miss rate means every request calls the policy engine
     → Check cache TTL configuration
     → Check if cache was recently invalidated

3. Check for resource contention:
     kubectl top pods -l app=thesa-bff
     → CPU throttling → increase CPU limits or add replicas
     → Memory pressure → check for memory leaks, increase limits

4. Check for hot path:
     Query: topk(10, rate(thesa_http_requests_total[5m])) by (path_pattern)
     → High request rate on specific endpoint → add caching or optimize
```

### Runbook: Workflow Store Unavailable

**Alert:** Readiness check shows `workflow_store: error`.

```
1. Check PostgreSQL connectivity:
     kubectl exec ... -- pg_isready -h <db-host> -p 5432

2. Check connection pool:
     Query: thesa_workflow_store_connections_active (if instrumented)
     → Pool exhausted → increase max_open_conns or investigate connection leaks

3. Check for long-running queries:
     SELECT pid, now() - pg_stat_activity.query_start AS duration, query
     FROM pg_stat_activity
     WHERE state = 'active' AND query NOT LIKE '%pg_stat_activity%'
     ORDER BY duration DESC;

4. Mitigation:
     Workflows will return 502 errors
     All other BFF endpoints (navigation, pages, forms, commands) continue to work
     → Prioritize database recovery
```

### Runbook: Definition Reload Failure

**Alert:** `thesa_definition_reload_total{status="failure"}` incremented.

```
1. Check BFF logs for the specific error:
     kubectl logs -l app=thesa-bff | grep "definition reload failed"
     → YAML parse error → fix the YAML syntax
     → Validation error → definition references a non-existent operation_id
     → Checksum mismatch → definition was modified outside the CI pipeline

2. Current state:
     BFF is still serving traffic using the PREVIOUS definitions
     No user impact unless the reload was for a critical fix

3. Resolution:
     Fix the definition file
     If using ConfigMap: update the ConfigMap, wait for propagation
     If using hot-reload: BFF will automatically retry on next file change
     If baked into image: build and deploy a new image
```

### Runbook: Pod Scaling Issues

**Alert:** HPA unable to scale or pods in CrashLoopBackOff.

```
1. Check HPA status:
     kubectl describe hpa thesa-bff
     → Insufficient resources → add cluster capacity
     → Metrics unavailable → check metrics-server / Prometheus adapter

2. If CrashLoopBackOff:
     kubectl logs -l app=thesa-bff --previous
     → Startup failure → check configuration, database connectivity, spec files
     → OOM killed → increase memory limits
     → Check events: kubectl describe pod <pod-name>

3. If pods are healthy but not receiving traffic:
     kubectl get endpoints thesa-bff
     → Readiness probe failing → check /ui/ready response
     → Check service selector matches pod labels
```

---

## Resource Sizing Guide

### Small Deployment (< 100 concurrent users)

```yaml
replicas: 2
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 256Mi
```

### Medium Deployment (100-1000 concurrent users)

```yaml
replicas: 3-5
resources:
  requests:
    cpu: 250m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 512Mi
```

### Large Deployment (1000+ concurrent users)

```yaml
replicas: 5-20 (HPA managed)
resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2000m
    memory: 1Gi
```

### Resource Drivers

| Factor | Impact | Mitigation |
|--------|--------|------------|
| Number of definitions | Memory (registry size) | Typically negligible (< 10MB) |
| Number of OpenAPI specs | Memory (index size) | Typically 50-200MB for large specs |
| Concurrent requests | CPU + goroutines | Scale horizontally (more pods) |
| Backend response sizes | Memory (buffering) | Stream large responses, set max body size |
| Workflow count | Database connections | Tune connection pool, scale PostgreSQL |
| Capability cache entries | Memory | Set max_entries limit, tune TTL |

---

## Monitoring Dashboard

Recommended Grafana dashboard panels:

### Overview Row
- Request rate (total, by status code)
- Error rate (5xx percentage)
- p50 / p95 / p99 latency

### Commands Row
- Command execution rate by command_id
- Command error rate by command_id
- Command duration by command_id

### Workflows Row
- Active workflow instances by workflow_id
- Workflow starts / completions / timeouts
- Step duration heatmap

### Backend Services Row
- Backend request rate by service_id
- Backend latency by service_id
- Circuit breaker state by service_id
- Retry rate by service_id

### System Row
- Definition reload count (success / failure)
- Capability cache hit ratio
- Pod CPU / memory usage
- Connection pool utilization

---

## Related Documents

- [16 — Observability and Reliability](16-observability-and-reliability.md) — metrics, tracing, circuit breakers
- [17 — Security Model](17-security-model.md) — security controls for deployment
- [19 — Go Package Structure](19-go-package-structure.md) — build targets and module layout
- [21 — Example End-to-End Flows](21-example-end-to-end-flows.md) — runtime behavior reference
