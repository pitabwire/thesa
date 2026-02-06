# 02 — System Topology

This document describes the physical and logical topology of a Thesa deployment:
who talks to whom, how tenancy works, and how the system is organized at the
network and service level.

---

## Network Topology

```
                           ┌──────────────────────────┐
                           │      Load Balancer        │
                           │  (TLS termination, WAF)   │
                           └────────────┬─────────────┘
                                        │
                           ┌────────────▼─────────────┐
                           │     Thesa BFF Cluster     │
                           │  ┌────────┐ ┌────────┐   │
                           │  │ BFF-1  │ │ BFF-2  │   │
                           │  │        │ │        │   │
                           │  └────┬───┘ └───┬────┘   │
                           │       │         │        │
                           └───────┼─────────┼────────┘
                                   │         │
                    ┌──────────────┼─────────┼──────────────┐
                    │         Internal Network               │
                    │                                        │
          ┌────────▼──┐  ┌──────────┐  ┌──────────┐  ┌────▼─────┐
          │ Identity   │  │ Policy   │  │Workflow  │  │Idempotency│
          │ Provider   │  │ Engine   │  │ Store    │  │ Store     │
          │ (OIDC)     │  │ (OPA)    │  │ (PgSQL)  │  │ (Redis)   │
          └────────────┘  └──────────┘  └──────────┘  └──────────┘
                    │                                        │
          ┌────────▼──┐  ┌──────────┐  ┌──────────┐  ┌────▼─────┐
          │ Orders    │  │ Inventory│  │ Customers│  │ Ledger   │
          │ Service   │  │ Service  │  │ Service  │  │ Service  │
          │ (REST)    │  │ (REST)   │  │ (REST)   │  │ (gRPC)   │
          └───────────┘  └──────────┘  └──────────┘  └──────────┘
```

### Key Points

- **TLS terminates at the load balancer.** Internal traffic between BFF and backend
  services uses mTLS or a service mesh. The BFF does not handle raw TLS.
- **Multiple BFF instances** run behind the load balancer for availability and
  horizontal scaling. They share no in-process state (all shared state is in
  external stores).
- **Backend services** are on the internal network and are not directly reachable
  from the internet. The BFF is the only ingress path for frontend traffic.
- **Infrastructure services** (identity provider, policy engine, workflow store,
  idempotency store) are also on the internal network.

---

## Tenancy Model

Thesa serves multi-tenant organizations. The tenancy hierarchy has three levels:

```
┌───────────────────────────────────────────────────────┐
│                     Organization                       │
│                    (Tenant / Org)                       │
│                                                        │
│  Identified by: tenant_id (from JWT)                   │
│  Examples: "acme-corp", "globex-inc", "initech"        │
│                                                        │
│  ┌──────────────────────┐  ┌──────────────────────┐   │
│  │      Partition A      │  │      Partition B      │   │
│  │   (Workspace / Env)   │  │   (Workspace / Env)   │   │
│  │                       │  │                       │   │
│  │  ID: "us-production"  │  │  ID: "eu-staging"     │   │
│  │                       │  │                       │   │
│  │  ┌────────┐ ┌──────┐ │  │  ┌────────┐ ┌──────┐ │   │
│  │  │ User 1 │ │User 2│ │  │  │ User 3 │ │User 4│ │   │
│  │  │ Alice  │ │ Bob  │ │  │  │Charlie │ │Diana │ │   │
│  │  └────────┘ └──────┘ │  │  └────────┘ └──────┘ │   │
│  └──────────────────────┘  └──────────────────────┘   │
└───────────────────────────────────────────────────────┘
```

### Tenant (Organization)

The top-level isolation boundary. All data is partitioned by tenant. A tenant
represents an organization — a company, a business unit, or whatever the platform
defines as its billing/subscription unit.

**How it flows:**
- The identity provider (OIDC) includes `tenant_id` as a custom claim in the JWT.
- The BFF extracts `tenant_id` from the verified JWT. It is NEVER read from headers,
  query parameters, or request body (see Principle P3).
- Every downstream call includes the tenant ID in the `X-Tenant-Id` header.
- Every WorkflowStore query is scoped by tenant ID.
- Every capability resolution is scoped by tenant ID (different tenants may have
  different role → capability mappings).

**Cross-Tenant Access:** There is no mechanism for cross-tenant access in Thesa.
If an administrative tool needs cross-tenant access, it should use a separate
backend API with its own authentication, not the BFF.

### Partition (Workspace / Environment)

A subdivision within a tenant. Partitions represent different environments,
regions, business units, or workspaces within a single organization.

**Use cases:**
- **Regional partitions:** "us-east", "eu-west" — data is scoped to a region.
- **Environment partitions:** "production", "staging" — separate environments
  within the same tenant.
- **Business unit partitions:** "retail", "wholesale" — separate business lines.
- **Workspace partitions:** "team-alpha", "team-beta" — team-level scoping.

**How it flows:**
- The frontend sends the selected partition in the `X-Partition-Id` header.
- The BFF validates that the partition belongs to the authenticated tenant. This
  validation can be done by:
  - Checking the partition against a claim in the JWT (e.g., `allowed_partitions`).
  - Checking against a partition registry service.
  - Checking against a local configuration file.
- After validation, `partition_id` is included in the RequestContext and propagated
  to backend services.

**Partition vs. Tenant Isolation:**
- Tenant isolation is HARD: no data crosses tenant boundaries, ever.
- Partition isolation is SOFT: a user might have access to multiple partitions
  within their tenant, and some operations might span partitions (depending on
  the domain).

### Subject (User)

The authenticated individual making the request. Identified by `subject_id`
(the JWT `sub` claim).

A subject belongs to exactly one tenant but may have access to multiple partitions
within that tenant. A subject's roles and capabilities may vary by partition.

### Session

A session represents a single authenticated session for a subject, typically tied
to a device. Sessions are identified by session ID (from the JWT or a separate
session cookie).

Sessions are relevant for:
- Concurrent session limits.
- Device-specific capabilities (e.g., mobile devices may have fewer features).
- Audit logging (tying actions to specific sessions).

---

## Service Categories

### Backend Domain Services

These are the business logic services that own domain data. Examples:
- **Orders Service** — manages order lifecycle.
- **Inventory Service** — manages stock levels.
- **Customers Service** — manages customer profiles.
- **Ledger Service** — manages financial entries (double-entry bookkeeping).

**Characteristics:**
- Each service has its own data store.
- Each service exposes an API (REST/OpenAPI or gRPC).
- Each service enforces its own authorization.
- Services are owned by domain teams.
- Services deploy independently.

**How Thesa interacts with them:**
- Thesa loads their OpenAPI specs at startup for validation and dynamic invocation.
- Thesa invokes their operations at runtime based on definitions.
- Thesa forwards the user's authentication token for backend-side authorization.
- Thesa applies circuit breakers, timeouts, and retries per service.

### Infrastructure Services

These are platform services that Thesa depends on:

| Service | Purpose | How Thesa Uses It |
|---------|---------|-------------------|
| Identity Provider (OIDC) | JWT issuance and verification | BFF verifies JWT signatures via JWKS endpoint |
| Policy Engine (OPA/Cedar) | Fine-grained authorization | BFF calls to resolve capabilities |
| Workflow Store (PostgreSQL) | Workflow state persistence | BFF reads/writes workflow instances |
| Idempotency Store (Redis) | Command deduplication | BFF checks/stores idempotency keys |
| Cache (Redis) | Capability caching, lookup caching | BFF caches resolved capabilities |

### Thesa BFF (This System)

Thesa itself runs as a stateless HTTP service (except for the workflow store
connection). Multiple instances run behind a load balancer.

**Scaling characteristics:**
- **CPU-bound:** JSON parsing, schema validation, capability evaluation.
- **IO-bound:** HTTP calls to backend services, policy engine, stores.
- **Memory:** OpenAPI specs + definitions are held in memory. For a system with
  100 backend services and 500 definition files, memory usage is typically under
  200MB.
- **Horizontally scalable:** No in-process state that can't be regenerated from
  external stores. Add more instances for more throughput.

---

## Data Flow Boundaries

Understanding what data crosses which boundaries is critical for security review.

### Boundary 1: Internet → BFF (via load balancer)

| Data | Direction | Carried By |
|------|-----------|------------|
| JWT token | In | Authorization header |
| Partition ID | In | X-Partition-Id header |
| Correlation ID | In | X-Correlation-Id header (optional, generated if absent) |
| Page/form/command requests | In | HTTP request path + body |
| Descriptors | Out | JSON response body |
| Data responses | Out | JSON response body |
| Error responses | Out | JSON response body |

**What NEVER crosses this boundary outbound:**
- Backend service URLs
- Operation IDs
- Capability strings
- Internal field names
- Raw backend responses

### Boundary 2: BFF → Backend Services (internal)

| Data | Direction | Carried By |
|------|-----------|------------|
| User's JWT token | Out | Authorization header (forwarded or exchanged) |
| Tenant ID | Out | X-Tenant-Id header |
| Partition ID | Out | X-Partition-Id header |
| Correlation ID | Out | X-Correlation-Id header |
| Trace context | Out | traceparent header (W3C) |
| Subject ID | Out | X-Request-Subject header |
| Mapped request payload | Out | HTTP request body |
| Backend response | In | HTTP response body |

### Boundary 3: BFF → Infrastructure Services

| Service | Data Out | Data In |
|---------|----------|---------|
| Identity Provider | Token for verification | JWKS keys |
| Policy Engine | Subject, tenant, roles, requested capabilities | Resolved capability set |
| Workflow Store | Workflow instances, events | Persisted state |
| Idempotency Store | Idempotency key + result | Cache hit/miss |
| Cache | Capability set to cache | Cached capability set |

---

## Service Configuration

Each backend service is registered in Thesa's configuration:

```yaml
services:
  orders-svc:
    base_url: "https://orders.internal:8443"
    spec_path: "specs/orders-svc.yaml"
    timeout: 10s
    auth:
      strategy: "forward_token"
    pagination:
      style: "offset"
      page_param: "offset"
      size_param: "limit"
    circuit_breaker:
      failure_threshold: 5
      timeout: 30s
    retry:
      max_attempts: 3
      backoff: "100ms"
      idempotent_only: true

  ledger-svc:
    base_url: "grpc://ledger.internal:9090"
    timeout: 5s
    auth:
      strategy: "service_token"
      client_id: "thesa-bff"
      token_endpoint: "https://auth.internal/oauth/token"
    # No spec_path — this service uses SDK handlers only
```

Service configuration is separate from definitions because:
- It contains infrastructure details (URLs, credentials) that domain teams should
  not see or manage.
- It changes based on environment (dev, staging, production).
- It is managed by the platform team, not domain teams.

---

## Multi-Instance Considerations

When running multiple BFF instances:

### Definition Consistency

All instances must load the same definitions. Options:
1. **Shared filesystem:** Definitions mounted from a shared NFS/EFS volume.
2. **Container image:** Definitions baked into the container image at build time.
3. **Artifact store:** Definitions fetched from S3/GCS at startup.

For hot-reload, all instances must receive the update. Options:
- File watcher on shared filesystem (inherent consistency).
- Pub/sub notification (e.g., Redis pub/sub) triggering reload on all instances.
- Rolling restart (deploy new container image with updated definitions).

### Workflow Store Consistency

Multiple instances read/write the same PostgreSQL database. Conflict prevention:
- **Optimistic locking:** WorkflowInstance has a `version` column. Updates include
  `WHERE version = ?` and increment the version. If another instance updated first,
  the query affects 0 rows and the operation is retried.
- **Event append-only:** Workflow events are insert-only, never updated. No conflicts.

### Capability Cache Consistency

Each instance maintains its own in-memory or Redis-backed capability cache.
- **TTL-based expiry:** Capabilities are cached for a configurable duration
  (default: 60 seconds). Changes in the policy engine propagate within the TTL.
- **For immediate invalidation:** Use a pub/sub channel. When capabilities change,
  publish an invalidation message. All instances clear the affected cache entries.

### Idempotency Store

The idempotency store MUST be shared across instances (Redis or PostgreSQL).
If each instance had its own idempotency store, a retried command hitting a
different instance would not be deduplicated.
