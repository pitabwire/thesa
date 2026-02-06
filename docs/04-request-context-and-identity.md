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

#### Strategy 1: Forward Token (Default)

The user's JWT is forwarded as-is to backend services. The backend re-validates
the token and extracts its own authorization context.

**Pros:** Simple, backend has full identity context, no additional tokens needed.
**Cons:** Token must be valid for all backend services (same audience or audience
array), token lifetime must cover the BFF's processing time.

#### Strategy 2: Service Token (Machine-to-Machine)

The BFF obtains its own service token using OAuth2 client credentials flow.
The user's identity is conveyed via `X-Request-Subject` and `X-Tenant-Id` headers.

**When to use:**
- Backend services don't accept user tokens (different audience/issuer).
- Backend services use a separate authorization model.
- The BFF needs elevated permissions that the user doesn't have.

**Flow:**
```
1. BFF calls token endpoint: POST https://auth.internal/oauth/token
   Body: grant_type=client_credentials&client_id=thesa-bff&client_secret=***&scope=orders.read orders.write
2. Receives access token scoped to BFF's permissions.
3. Caches token until expiry.
4. Uses cached token for backend calls.
5. Backend trusts X-Tenant-Id / X-Request-Subject headers when they come from the BFF
   (verified via mTLS or API gateway rules).
```

#### Strategy 3: Token Exchange (RFC 8693)

The BFF exchanges the user's token for a new token scoped to a specific backend
service. This provides the benefits of both approaches: the backend gets a valid
user token with the correct audience.

**Flow:**
```
1. BFF calls token endpoint: POST https://auth.internal/oauth/token
   Body: grant_type=urn:ietf:params:oauth:grant-type:token-exchange
         &subject_token={user_jwt}
         &subject_token_type=urn:ietf:params:oauth:token-type:jwt
         &audience=orders-service
2. Receives a new JWT for the user, scoped to the orders service.
3. Caches per (user, service) with short TTL.
4. Uses for backend call.
```

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
  Set header "Authorization": "Bearer " + rctx.OriginalToken
  Set header "X-Tenant-Id": rctx.TenantID
  Set header "X-Partition-Id": rctx.PartitionID
  Set header "X-Correlation-Id": rctx.CorrelationID
  Set header "X-Request-Subject": rctx.SubjectID
```

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
