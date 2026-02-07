# 04 — Request Context and Identity

This document describes how Thesa establishes the identity, tenancy, and tracing
context for every request, how that context flows through the system, and how it
is propagated to downstream services.

---

## The RequestContext

Every authenticated request produces a `RequestContext` — a strongly typed value
object that carries all identity, tenancy, and tracing information for the lifetime
of the request.

### Fields

| Field | Type | Source | Required | Description |
|-------|------|--------|----------|-------------|
| `SubjectID` | string | JWT `sub` claim | Yes | The authenticated user's unique identifier |
| `Email` | string | JWT `email` claim | No | User's email address |
| `TenantID` | string | JWT `tenant_id` claim | Yes | The organization this user belongs to |
| `PartitionID` | string | `X-Partition-Id` header | Yes | The workspace/environment the user is operating in |
| `Roles` | []string | JWT `roles` claim | Yes (may be empty) | The user's assigned roles |
| `Claims` | map[string]any | Full JWT payload | No | All JWT claims for policy evaluation |
| `SessionID` | string | JWT `session_id` claim | No | The current session identifier |
| `DeviceID` | string | `X-Device-Id` header | No | Device fingerprint |
| `CorrelationID` | string | `X-Correlation-Id` header or generated | Yes | Request correlation identifier |
| `TraceID` | string | `traceparent` header | No | OpenTelemetry trace ID |
| `SpanID` | string | Current span | No | Current span ID |
| `Locale` | string | `Accept-Language` header | No | User's locale preference (e.g., "en-US") |
| `Timezone` | string | `X-Timezone` header | No | User's timezone (e.g., "America/New_York") |

### Immutability

The RequestContext is **immutable after construction**. Once the middleware pipeline
constructs it and attaches it to the Go context, no component may modify it. This
ensures:

- Thread safety: multiple goroutines may read the context concurrently.
- Audit integrity: the context logged at request start matches the context used
  at request end.
- Predictability: handlers cannot alter identity mid-request.

### Validation

The RequestContext is validated immediately after construction. The following fields
are mandatory — if any are missing, the request is rejected with 401 or 400:

- `SubjectID` — extracted from JWT `sub`. If absent, the token is malformed → 401.
- `TenantID` — extracted from JWT `tenant_id` claim. If absent, the token is
  missing required claims → 401.
- `PartitionID` — extracted from `X-Partition-Id` header. If absent → 400
  ("X-Partition-Id header is required").
- `CorrelationID` — extracted from `X-Correlation-Id` header, or auto-generated
  (UUID v4) if absent. Always present.

---

## Identity Verification

### JWT Verification Flow

```
1. Extract token from Authorization header:
   "Bearer eyJhbGciOiJSUzI1NiIs..."
   → If missing: return 401 { "error": { "code": "UNAUTHORIZED", "message": "Missing authorization header" } }
   → If malformed (not "Bearer {token}"): return 401

2. Parse JWT header (without verification) to extract:
   - `kid` (key ID) — identifies which public key to use
   - `alg` (algorithm) — must be RS256, RS384, RS512, ES256, ES384, or ES512
   → If algorithm is "none" or HMAC: return 401 (algorithm confusion attack prevention)

3. Fetch public key:
   - Look up `kid` in the cached JWKS (JSON Web Key Set)
   - If not found: refresh JWKS from the identity provider's JWKS endpoint
   - If still not found: return 401 { "message": "Unknown signing key" }
   - JWKS is cached with a configurable TTL (default: 1 hour)
   - JWKS refresh is rate-limited (max once per 5 minutes) to prevent abuse

4. Verify token signature using the public key.
   → If invalid: return 401 { "message": "Invalid token signature" }

5. Validate token claims:
   - `exp` (expiration): must be in the future
     → If expired: return 401 { "message": "Token expired" }
   - `nbf` (not before): must be in the past (with 30s clock skew tolerance)
   - `iss` (issuer): must match configured issuer URL
     → If mismatch: return 401 { "message": "Invalid token issuer" }
   - `aud` (audience): must contain the BFF's audience identifier
     → If mismatch: return 401 { "message": "Invalid token audience" }
   - `tenant_id` (custom claim): must be present
     → If missing: return 401 { "message": "Token missing tenant_id claim" }

6. Extract claims into RequestContext fields.
```

### JWKS Caching and Rotation

The identity provider's JWKS endpoint (e.g., `https://auth.example.com/.well-known/jwks.json`)
returns the set of public keys used to sign tokens. Key rotation is handled:

1. Keys are cached in memory with a configurable TTL.
2. When a token presents a `kid` not in the cache, the cache is refreshed.
3. The old key remains valid until it's removed from the JWKS response (allowing
   a grace period for tokens signed with the old key).
4. If the JWKS endpoint is unreachable, the last known keys are used (degraded mode).

### Token Types

Thesa supports these JWT formats:

| Format | Use Case | Configuration |
|--------|----------|---------------|
| Standard JWT (RS256) | Default OIDC tokens | Verify with JWKS |
| JWT with custom claims | Tokens enriched with tenant/role claims | Same as above, custom claim paths configured |
| Opaque tokens | Tokens that require introspection | Call introspection endpoint to resolve claims |

For opaque tokens, the BFF calls the identity provider's introspection endpoint
to resolve the token into claims. The result is cached for the token's remaining
lifetime.

---

## Context Construction Pipeline

### Step-by-Step

```
Incoming HTTP Request
  │
  ├── Step 1: Recovery middleware
  │   (catches panics, not identity-related)
  │
  ├── Step 2: Request ID middleware
  │   Extract X-Correlation-Id from header, or generate UUID v4.
  │   Attach to Go context and response header.
  │
  ├── Step 3: Authentication middleware
  │   a. Skip for health/readiness endpoints (configured whitelist).
  │   b. Extract Bearer token from Authorization header.
  │   c. Verify token (see JWT Verification Flow above).
  │   d. Extract claims into a temporary claims map.
  │   e. Attach verified claims to Go context.
  │   f. On failure: return 401 with error envelope.
  │
  ├── Step 4: Context construction middleware
  │   a. Read verified claims from Go context.
  │   b. Extract SubjectID from claims["sub"].
  │   c. Extract TenantID from claims["tenant_id"].
  │   d. Extract Roles from claims["roles"] (or claims["realm_access"]["roles"] for Keycloak).
  │   e. Extract Email from claims["email"].
  │   f. Extract SessionID from claims["session_id"] or claims["sid"].
  │   g. Read PartitionID from X-Partition-Id header.
  │   h. Validate PartitionID:
  │      - If claims contain "allowed_partitions": verify PartitionID is in the list.
  │      - Or: call partition registry to verify PartitionID belongs to TenantID.
  │      - Or: accept any PartitionID (for systems without partition restrictions).
  │      - On failure: return 403 { "message": "Access denied to partition" }.
  │   i. Read CorrelationID from earlier middleware.
  │   j. Read TraceID/SpanID from OpenTelemetry context.
  │   k. Read DeviceID from X-Device-Id header.
  │   l. Read Locale from Accept-Language header.
  │   m. Read Timezone from X-Timezone header.
  │   n. Construct RequestContext struct.
  │   o. Validate mandatory fields.
  │   p. Attach RequestContext to Go context using WithRequestContext().
  │
  └── Step 5: Handler
      Access RequestContext via MustRequestContext(ctx).
```

### Claim Path Configuration

Different identity providers use different claim structures. Thesa supports
configurable claim paths:

```yaml
identity:
  jwks_url: "https://auth.example.com/.well-known/jwks.json"
  issuer: "https://auth.example.com"
  audience: "thesa-bff"

  # Claim extraction paths (JSONPath-like)
  claim_paths:
    subject: "sub"
    tenant: "tenant_id"                          # or "custom:tenant_id" for AWS Cognito
    roles: "roles"                               # or "realm_access.roles" for Keycloak
    email: "email"
    session: "session_id"                        # or "sid"
    allowed_partitions: "allowed_partitions"     # optional
```

This allows Thesa to work with any OIDC-compliant identity provider without
code changes.

---

## Context Propagation to Backend Services

When the BFF invokes a backend service, the RequestContext is propagated via
standard HTTP headers. This allows backend services to:

- Know who the request is for (authorization, audit logging).
- Know which tenant/partition the request is scoped to.
- Correlate their logs with the frontend request.
- Participate in distributed tracing.

### Propagated Headers

| Header | Value | Purpose |
|--------|-------|---------|
| `Authorization` | `Bearer {token}` | User identity (forwarded or exchanged) |
| `X-Tenant-Id` | RequestContext.TenantID | Tenant scoping |
| `X-Partition-Id` | RequestContext.PartitionID | Partition scoping |
| `X-Correlation-Id` | RequestContext.CorrelationID | Request correlation |
| `X-Request-Subject` | RequestContext.SubjectID | Audit trail |
| `traceparent` | W3C trace context | Distributed tracing |
| `tracestate` | W3C trace state | Vendor-specific trace data |

### Authentication Strategies for Backend Calls

Each backend service is configured with one of four authentication strategies.
The strategy determines how the BFF authenticates to the backend and how the
user's identity is conveyed.

**Strategy selection:**

| Strategy | Config value | When to use |
|----------|-------------|-------------|
| Forward Token | `forward_token` | Backend accepts the same user JWT (shared audience/issuer) |
| Service Token | `service_token` | Backend uses its own auth model; BFF needs its own credentials |
| Token Exchange | `token_exchange` | Backend needs a user-scoped token with a different audience |
| mTLS | `mtls` | Backend authenticates via client certificates (no Bearer token needed) |

#### Strategy 1: Forward Token (`forward_token`)

The user's JWT is forwarded as-is to backend services. This is the default
strategy when no `auth` block is configured for a service.

**Header behavior:**

```
Authorization: Bearer {user's original JWT}
X-Tenant-Id: {from RequestContext}
X-Partition-Id: {from RequestContext}
X-Correlation-Id: {from RequestContext}
X-Request-Subject: {from RequestContext}
```

**Token near expiry:** When the user's token has less than 30 seconds remaining
before expiry (`exp` claim), the BFF still forwards it — the backend is
responsible for rejecting expired tokens. The BFF does not refresh user tokens;
that is the frontend's responsibility. If the backend returns 401, the BFF
propagates it to the frontend, which triggers a token refresh and retry.

**Token refresh flow:**

```
1. Frontend sends request with JWT (exp: T+25s)
2. BFF forwards JWT to backend
3. Backend validates token → succeeds (within clock skew)
4. If backend returns 401:
   a. BFF returns 401 to frontend
   b. Frontend refreshes token with identity provider
   c. Frontend retries with new token
```

**Pros:** Simple, backend has full identity context, no additional tokens needed.
**Cons:** Token must be valid for all backend services (same audience or audience
array), token lifetime must cover the BFF's processing time.

**Implementation note:** The current `buildRequestHeaders` function in
`internal/invoker/openapi.go` always forwards `rctx.Token` regardless of the
configured strategy. Strategy-based dispatch is defined but not yet enforced
at runtime — all services currently receive the forwarded user token.

#### Strategy 2: Service Token (`service_token`)

The BFF obtains its own service token using the OAuth2 client credentials flow
(RFC 6749 §4.4). The user's identity is conveyed via `X-Request-Subject` and
`X-Tenant-Id` headers instead of the Authorization header.

**When to use:**
- Backend services don't accept user tokens (different audience/issuer).
- Backend services use a separate authorization model.
- The BFF needs elevated permissions that the user doesn't have.

**Configuration:**

```yaml
services:
  customers:
    base_url: "https://customers.internal"
    auth:
      strategy: "service_token"
      client_id: "thesa-bff"
      token_endpoint: "https://auth.internal/oauth/token"
      # client_secret is read from THESA_SERVICE_TOKEN_SECRET env var
      # scopes: ["customers.read", "customers.write"]  # optional
```

**OAuth2 client credentials flow:**

```
1. BFF calls token endpoint:
   POST https://auth.internal/oauth/token
   Content-Type: application/x-www-form-urlencoded

   grant_type=client_credentials
   &client_id=thesa-bff
   &client_secret={from THESA_SERVICE_TOKEN_SECRET}
   &scope=customers.read customers.write

2. Response:
   {
     "access_token": "eyJ...",
     "token_type": "bearer",
     "expires_in": 3600
   }

3. BFF caches the token.
4. All requests to this service use the cached service token.
```

**Token caching and refresh:**

The service token is cached in memory. The BFF proactively refreshes the token
before it expires to avoid request failures:

```
Token lifecycle:
  ├── T+0:    Token obtained, cached
  ├── T+0 to T+expiry-60s: Token used as-is
  ├── T+expiry-60s:        Refresh triggered in background
  │   └── New token obtained → replaces cached token
  └── T+expiry:            Old token expires (already replaced)

If refresh fails:
  ├── Continue using old token until actual expiry
  ├── Retry refresh every 10s
  └── If token expires with no replacement → requests fail with 503
```

**Thread-safe token management:**

Multiple concurrent requests may need the service token simultaneously. The
token cache uses `sync.RWMutex` to ensure thread safety:

```
Read path (hot path — concurrent):
  1. RLock
  2. Read cached token
  3. Check expiry
  4. If valid: return token, RUnlock
  5. If expiring soon: trigger background refresh, return current token, RUnlock

Write path (refresh — exclusive):
  1. Lock
  2. Double-check: another goroutine may have refreshed already
  3. Call token endpoint
  4. Store new token
  5. Unlock
```

**Header behavior with service token:**

```
Authorization: Bearer {BFF's service token}
X-Tenant-Id: {from RequestContext — user's tenant}
X-Partition-Id: {from RequestContext}
X-Correlation-Id: {from RequestContext}
X-Request-Subject: {from RequestContext — user's subject ID}
```

The backend trusts `X-Tenant-Id` and `X-Request-Subject` headers because the
request comes from the authenticated BFF (verified via mTLS, API gateway rules,
or service mesh policies).

#### Strategy 3: Token Exchange (`token_exchange`)

The BFF exchanges the user's token for a new token scoped to a specific backend
service using the OAuth2 Token Exchange grant (RFC 8693). This provides the
benefits of both approaches: the backend gets a valid user-identity token with
the correct audience.

**When to use:**
- Backend requires a user-identity token (not a service token).
- Backend has a different audience than the BFF.
- Fine-grained per-service authorization based on the user's identity.
- The identity provider supports RFC 8693.

**Configuration:**

```yaml
services:
  orders:
    base_url: "https://orders.internal"
    auth:
      strategy: "token_exchange"
      client_id: "thesa-bff"
      token_endpoint: "https://auth.internal/oauth/token"
      # audience is derived from the service ID or configured explicitly
      # exchange_audience: "orders-service"
```

**Exchange flow:**

```
1. BFF calls token endpoint:
   POST https://auth.internal/oauth/token
   Content-Type: application/x-www-form-urlencoded

   grant_type=urn:ietf:params:oauth:grant-type:token-exchange
   &client_id=thesa-bff
   &client_secret={from THESA_SERVICE_TOKEN_SECRET}
   &subject_token={user's JWT}
   &subject_token_type=urn:ietf:params:oauth:token-type:jwt
   &audience=orders-service
   &scope=orders.read orders.write

2. Response:
   {
     "access_token": "eyJ...",
     "issued_token_type": "urn:ietf:params:oauth:token-type:jwt",
     "token_type": "bearer",
     "expires_in": 300
   }

3. BFF caches per (subject_id, service_id) with short TTL.
4. Uses exchanged token for backend call.
```

**Exchanged token caching:**

Exchanged tokens are cached per `(subject_id, service_id)` tuple:

| Cache Key | TTL | Eviction |
|-----------|-----|----------|
| `{subject_id}:{service_id}` | min(token.expires_in - 30s, 5 minutes) | LRU with configurable max entries |

The short TTL ensures that if the user's permissions change, the exchanged token
is refreshed quickly. The cache prevents excessive calls to the token endpoint
for users making multiple requests.

**Scope of exchanged token:**

The exchanged token carries the user's identity (`sub`, `tenant_id`, `roles`)
but is scoped to the target service's audience. The identity provider may
restrict the exchanged token's claims based on the target audience — for example,
only including roles relevant to the orders service.

#### Strategy 4: mTLS (`mtls`)

The BFF authenticates to the backend using a client certificate (mutual TLS).
No Bearer token is sent in the Authorization header. The user's identity is
conveyed via `X-Request-Subject` and `X-Tenant-Id` headers.

**When to use:**
- Backend services use certificate-based authentication.
- Service mesh is not available or not trusted for authentication.
- Strongest machine-to-machine authentication is required.

**Configuration:**

```yaml
services:
  ledger:
    base_url: "https://ledger.internal"
    auth:
      strategy: "mtls"
      # cert_file: "/certs/client.crt"    # or from THESA_MTLS_CERT_FILE
      # key_file: "/certs/client.key"     # or from THESA_MTLS_KEY_FILE
      # ca_file: "/certs/ca.crt"          # optional: CA to verify backend cert
```

**TLS client configuration:**

```
1. At startup:
   a. Load client certificate and private key from configured paths.
   b. Optionally load CA certificate for backend verification.
   c. Create TLS config with client certificate.
   d. Create HTTP transport with custom TLS config.

2. Per request:
   a. TLS handshake presents client certificate to backend.
   b. Backend verifies certificate against its CA.
   c. No Authorization header is set.
   d. User identity is in X-Request-Subject and X-Tenant-Id headers.
```

**Header behavior with mTLS:**

```
(No Authorization header)
X-Tenant-Id: {from RequestContext}
X-Partition-Id: {from RequestContext}
X-Correlation-Id: {from RequestContext}
X-Request-Subject: {from RequestContext}
```

**Certificate rotation without restart:**

Certificate rotation is handled by watching the certificate files for changes:

```
Certificate rotation flow:
  1. Operations team deploys new cert/key files to the mounted path.
  2. File watcher detects the change (fsnotify, polling every 60s as fallback).
  3. New certificate is loaded and validated.
  4. TLS config is atomically swapped (sync/atomic pointer swap).
  5. New connections use the new certificate.
  6. Existing connections continue with the old certificate until they close.
  7. No request failures during rotation.

Fallback when cert files are missing at startup:
  - If strategy is "mtls" and cert files are missing → fatal startup error.
  - If cert files disappear after startup → log error, continue with last valid cert.
  - If new cert files are invalid (expired, wrong format) → log error, keep old cert.
```

### Per-Service Configuration Example

The following shows all four strategies configured for different backend services
in the same deployment:

```yaml
services:
  # Forward token: backend shares the same identity provider
  orders:
    base_url: "https://orders.internal"
    timeout: 10s
    auth:
      strategy: "forward_token"
    circuit_breaker:
      failure_threshold: 5
      timeout: 30s

  # Service token: backend has its own auth model
  customers:
    base_url: "https://customers.internal"
    timeout: 10s
    auth:
      strategy: "service_token"
      client_id: "thesa-bff"
      token_endpoint: "https://auth.internal/oauth/token"
    circuit_breaker:
      failure_threshold: 5
      timeout: 30s

  # Token exchange: backend needs user identity with different audience
  payments:
    base_url: "https://payments.internal"
    timeout: 15s
    auth:
      strategy: "token_exchange"
      client_id: "thesa-bff"
      token_endpoint: "https://auth.internal/oauth/token"
    circuit_breaker:
      failure_threshold: 3
      timeout: 60s

  # mTLS: backend requires certificate authentication
  ledger:
    base_url: "https://ledger.internal"
    timeout: 10s
    auth:
      strategy: "mtls"
    circuit_breaker:
      failure_threshold: 5
      timeout: 30s
```

**Environment variables for secrets:**

```
THESA_SERVICE_TOKEN_SECRET=<client secret for OAuth2 client credentials>
THESA_MTLS_CERT_FILE=/certs/client.crt
THESA_MTLS_KEY_FILE=/certs/client.key
THESA_MTLS_CA_FILE=/certs/ca.crt
```

### Strategy Decision Matrix

| Criterion | Forward Token | Service Token | Token Exchange | mTLS |
|-----------|:---:|:---:|:---:|:---:|
| Backend validates user identity directly | Yes | No | Yes | No |
| Backend has different audience | No | Yes | Yes | Yes |
| BFF needs elevated permissions | No | Yes | No | No |
| User context preserved in token | Yes | No (headers only) | Yes | No (headers only) |
| Requires IdP support for grant type | No | client_credentials | token-exchange (RFC 8693) | No |
| Token caching complexity | None | Per-service | Per-user-per-service | None |
| Works if IdP is temporarily unreachable | Yes | Until cached token expires | Until cached token expires | Yes |

---

## Accessing the RequestContext

### In Handlers

```go
func (h *Handler) HandleGetPage(w http.ResponseWriter, r *http.Request) {
    rctx := model.MustRequestContext(r.Context())
    // rctx.SubjectID, rctx.TenantID, rctx.PartitionID, etc.
}
```

`MustRequestContext` panics if the context is not present. This is safe because
the authentication middleware guarantees presence for all handler routes.

### In Internal Components

Components that receive a `context.Context` extract the RequestContext from it:

```go
func (e *CommandExecutor) Execute(ctx context.Context, commandId string, input CommandInput) (*CommandResponse, error) {
    rctx := model.MustRequestContext(ctx)
    // Use rctx for capability checks, input mapping, backend calls
}
```

### In Tests

For unit testing, create a RequestContext directly and attach it to the context:

```go
rctx := &model.RequestContext{
    SubjectID:     "test-user",
    TenantID:      "test-tenant",
    PartitionID:   "test-partition",
    Roles:         []string{"admin"},
    CorrelationID: "test-correlation",
}
ctx := model.WithRequestContext(context.Background(), rctx)
```

---

## Security Considerations

### Token Validation Strictness

- **Clock skew tolerance:** 30 seconds (for `exp` and `nbf` claims). Configurable
  but should not exceed 60 seconds.
- **Algorithm restriction:** Only asymmetric algorithms (RS256, ES256, etc.). Never
  accept `"alg": "none"` or HMAC-based algorithms (unless the BFF is the only
  consumer and shares a secret with the IdP, which is uncommon).
- **Issuer validation:** Exact string match against configured issuer. No pattern matching.
- **Audience validation:** Token must contain the BFF's audience in the `aud` claim.

### Tenant ID Extraction

- `TenantID` is ALWAYS extracted from the verified JWT's custom claims.
- It is NEVER read from: query parameters, request body, path parameters, or
  request headers (X-Tenant-Id on the inbound request is ignored).
- The outbound `X-Tenant-Id` header is set FROM the RequestContext, not from the
  inbound request.

### Partition Validation

- `PartitionID` IS read from the `X-Partition-Id` request header.
- It MUST be validated against the user's allowed partitions before being used.
- If a user sends a partition they don't have access to → 403.
- If a user sends a partition that doesn't belong to their tenant → 403.

### Session and Device Tracking

- `SessionID` and `DeviceID` are used for:
  - Concurrent session limits (if configured).
  - Audit logging (tying actions to specific sessions/devices).
  - Anomaly detection (sudden device change, impossible travel).
- They do NOT affect authorization decisions in the BFF. They are propagated
  to backend services and policy engines that may use them.

---

## Common Patterns

### Pattern 1: Using Context in Capability Resolution

```
CapabilityResolver.Resolve(rctx):
  Cache key: hash(rctx.SubjectID, rctx.TenantID, rctx.PartitionID)
  If cached: return cached CapabilitySet
  Else: call PolicyEvaluator with (SubjectID, TenantID, PartitionID, Roles)
        Cache result
        Return CapabilitySet
```

Capabilities may vary by tenant (different tenants have different role configurations)
and by partition (different partitions may have different feature sets).

### Pattern 2: Using Context in Backend Calls

```
OpenAPIInvoker.Invoke(ctx, rctx, binding, input):
  Set header "Authorization": "Bearer " + rctx.Token
  Set header "X-Tenant-Id": rctx.TenantID
  Set header "X-Partition-Id": rctx.PartitionID
  Set header "X-Correlation-Id": rctx.CorrelationID
  Set header "X-Request-Subject": rctx.SubjectID
```

> **Note:** This shows the forward_token strategy (default). For other strategies,
> the Authorization header is set differently — see "Authentication Strategies
> for Backend Calls" above.

### Pattern 3: Using Context in Input Mapping

The `context.*` expression prefix references RequestContext fields:

```yaml
input:
  path_params:
    tenantId: "context.tenant_id"
  body_mapping: "template"
  body_template:
    updatedBy: "context.subject_id"
    partition: "context.partition_id"
```

The InputMapping resolver extracts these values from the RequestContext at
command execution time.

### Pattern 4: Using Context in Workflow State

When a workflow is started, the RequestContext provides the initial actor:

```
WorkflowInstance.SubjectID = rctx.SubjectID     // who started it
WorkflowInstance.TenantID = rctx.TenantID       // tenant isolation
WorkflowInstance.PartitionID = rctx.PartitionID // partition isolation
```

When a workflow step is advanced, the advancing user's context is recorded:

```
WorkflowEvent.ActorID = rctx.SubjectID          // who advanced this step
```

This allows audit trails to show exactly who performed each step.
