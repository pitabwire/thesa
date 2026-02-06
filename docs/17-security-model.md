# 17 — Security Model

This document describes the threat model, security controls, and hardening measures
for Thesa. It covers every attack vector relevant to a multi-tenant BFF and the
mitigations implemented.

---

## Threat Model

### Assets to Protect

| Asset | Impact if Compromised |
|-------|----------------------|
| Tenant data | Cross-tenant data exposure; regulatory violation |
| User tokens | Account takeover; unauthorized actions |
| Workflow state | Process manipulation; financial fraud |
| Definition files | Unauthorized API exposure; capability bypass |
| Backend credentials | Unauthorized backend access |
| Audit logs | Covering tracks after breach |

### Threat Actors

| Actor | Capability | Goal |
|-------|-----------|------|
| Malicious user (authenticated) | Valid token for their tenant | Access other tenants' data; escalate privileges |
| Compromised frontend | Can send arbitrary requests | Invoke unauthorized commands; bypass UI restrictions |
| Network attacker | Can intercept/modify traffic | Token theft; man-in-the-middle |
| Insider (malicious admin) | Can modify definitions/config | Add unauthorized API exposure; bypass capabilities |

---

## Security Controls

### 1. Cross-Tenant Data Access Prevention

**Threat:** User A (tenant "acme") accesses data belonging to tenant "globex".

**Control: Structural tenant isolation (Principle P3)**

- Tenant ID is extracted exclusively from the verified JWT token.
- Tenant ID is NEVER read from request headers, query parameters, or body.
- Every backend call includes `X-Tenant-Id` derived from the JWT.
- WorkflowStore queries are always scoped by tenant ID.
- Resources from the wrong tenant return 404 (not 403) to prevent enumeration.

**Verification:**

```
Test: Cross-tenant workflow access
  1. Authenticate as Tenant A, start workflow → wf-001
  2. Authenticate as Tenant B
  3. GET /ui/workflows/wf-001
  4. Assert: 404 NOT_FOUND (not 403, not 200)

Test: Cross-tenant data access
  1. Authenticate as Tenant A
  2. GET /ui/pages/orders.list/data with intercepted request modified to add
     X-Tenant-Id: tenant-b header
  3. Assert: BFF ignores the header; uses JWT tenant
  4. Assert: Data returned is for Tenant A only
```

### 2. Privilege Escalation Prevention

**Threat:** User with "viewer" role executes "admin" actions.

**Control: Capability-based authorization (Principle P5)**

- Capabilities are resolved from the policy engine based on JWT roles/claims.
- The frontend cannot influence capability resolution.
- Every command and workflow step checks capabilities.
- Capability checks happen at both descriptor generation AND execution time.

**Verification:**

```
Test: Viewer cannot execute admin command
  1. Authenticate as user with role "order_viewer"
  2. POST /ui/commands/orders.cancel
  3. Assert: 403 FORBIDDEN

Test: Descriptor filtering
  1. Authenticate as "order_viewer"
  2. GET /ui/pages/orders.detail
  3. Assert: "Cancel" action is NOT in the descriptor
```

### 3. Direct Backend API Invocation Prevention

**Threat:** Frontend bypasses BFF and calls backend services directly.

**Control: Network isolation + backend authentication**

- Backend services are on an internal network, not reachable from the internet.
- Backend services authenticate requests via:
  - mTLS (only BFF has a valid client certificate)
  - API gateway rules (only BFF IP range allowed)
  - Service mesh policies (Istio, Linkerd)
- Even if a user obtains a backend URL, they cannot reach it.

### 4. Command Replay Prevention

**Threat:** Attacker captures a command request and replays it.

**Control: Idempotency keys with TTL**

- Commands with idempotency configured require a unique key per invocation.
- Replaying the same key + input returns the cached result (no side effect).
- Replaying the same key + different input returns 409 Conflict.
- Keys expire after the configured TTL (e.g., 24 hours).
- Tokens have short expiry; replayed tokens are rejected by JWT verification.

### 5. Workflow Tampering Prevention

**Threat:** Attacker modifies workflow state to skip approval steps.

**Control: Server-side state management**

- Workflow state is stored in the WorkflowStore (PostgreSQL), not in the client.
- The frontend cannot modify workflow state directly.
- Every workflow advance validates:
  - The event is valid for the current step (transition check).
  - The user has the required capabilities for the step.
  - The tenant ID matches.
  - Optimistic locking prevents concurrent modification.
- Workflow events are append-only (immutable audit trail).

**Verification:**

```
Test: Cannot skip approval step
  1. Start workflow at "review" step
  2. POST /ui/workflows/{id}/advance with event "completed"
     (trying to skip approval)
  3. Assert: 422 INVALID_TRANSITION (no transition from "review" with event "completed")

Test: Cannot advance someone else's step
  1. Start workflow at "review" step (assignee: role "approver")
  2. Authenticate as user without "approver" role
  3. POST /ui/workflows/{id}/advance with event "approved"
  4. Assert: 403 FORBIDDEN (user lacks step capability)
```

### 6. Definition Tampering Prevention

**Threat:** Attacker modifies definition files to expose unauthorized APIs or bypass capabilities.

**Control: Integrity verification**

- SHA-256 checksums are computed at load time.
- In strict mode: checksums are verified against a signed manifest.
- In production: definitions are loaded from read-only filesystems or verified artifacts.
- Definition changes are logged as audit events with old and new checksums.
- Hot-reload can be disabled in production.

**Deployment best practices:**

```
1. Definitions are built and signed in CI/CD.
2. Container images bundle definitions as read-only layers.
3. At runtime, the filesystem is mounted read-only.
4. Any modification attempt fails at the OS level.
```

### 7. Token Security

**Threats:** Token theft, token forgery, algorithm confusion.

**Controls:**

| Control | Implementation |
|---------|---------------|
| Signature verification | JWT verified with asymmetric key (RS256/ES256) from JWKS |
| Algorithm restriction | Only RS256, RS384, RS512, ES256, ES384, ES512; NEVER "none" or HMAC |
| Expiry enforcement | `exp` claim checked with 30s clock skew tolerance |
| Issuer validation | `iss` claim must match configured issuer |
| Audience validation | `aud` claim must contain BFF audience identifier |
| Key rotation | JWKS cached and refreshed; old keys remain valid during rotation |

### 8. Injection Prevention

**Threats:** SQL injection, XSS, command injection.

**Controls:**

| Vector | Prevention |
|--------|-----------|
| SQL injection | BFF uses parameterized queries for WorkflowStore; no string concatenation in SQL |
| XSS | BFF returns JSON only (no HTML); Content-Type: application/json |
| Command injection | BFF never executes shell commands from user input |
| Path traversal | Definition files loaded from fixed directories only; paths are not user-influenced |
| SSRF | Backend URLs are from configuration only; user input never influences target URLs |
| Header injection | All propagated headers are sanitized (newlines stripped) |

### 9. Enumeration Prevention

**Threats:** Attacker enumerates tenant IDs, user IDs, resource IDs.

**Controls:**

- Wrong-tenant resources return 404 (not 403).
- Sequential IDs are not used for workflows (UUID v4 instead).
- Rate limiting prevents automated enumeration attempts.
- Failed authentication attempts are rate-limited by IP.

### 10. Information Leakage Prevention

**Threats:** Error messages reveal internal details.

**Controls:**

- 5xx errors return generic messages only.
- Backend URLs are never in responses.
- Operation IDs are never in responses.
- Stack traces are logged server-side, never returned to clients.
- Definition content is never exposed to the frontend.
- OpenAPI spec content is never exposed to the frontend.

---

## Security Headers

The BFF sets these security headers on all responses:

```
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Type: application/json
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 0            (disabled; modern browsers use CSP instead)
Cache-Control: no-store         (for authenticated responses)
Referrer-Policy: strict-origin-when-cross-origin
```

---

## Audit Logging

### What Gets Audited

| Event | Logged Fields |
|-------|--------------|
| Authentication success | subject_id, tenant_id, ip_address, user_agent |
| Authentication failure | ip_address, reason, user_agent |
| Command execution | subject_id, tenant_id, command_id, success/failure |
| Workflow start | subject_id, tenant_id, workflow_id, instance_id |
| Workflow advance | subject_id, tenant_id, instance_id, step_id, event |
| Workflow cancel | subject_id, tenant_id, instance_id, reason |
| Definition reload | old_checksum, new_checksum, initiator |
| Capability cache invalidation | subject_id, tenant_id, reason |

### Audit Log Format

```json
{
  "type": "audit",
  "event": "command.executed",
  "timestamp": "2025-01-15T10:30:00Z",
  "subject_id": "user-123",
  "tenant_id": "acme-corp",
  "partition_id": "us-production",
  "correlation_id": "corr-456",
  "command_id": "orders.update",
  "resource_id": "ord-123",
  "success": true,
  "ip_address": "198.51.100.42",
  "user_agent": "ThesaFlutter/1.0"
}
```

### Audit Log Integrity

Audit logs should be written to a tamper-evident store:
- Append-only log (e.g., S3 with object lock).
- Signed log entries (each entry includes a hash of the previous entry).
- Separate from application logs (different log stream/destination).

---

## Dependency Security

### Supply Chain Considerations

- Pin Go module versions in `go.sum`.
- Regularly scan dependencies for vulnerabilities (`govulncheck`).
- Use minimal base images for containers (distroless or Alpine).
- Sign container images.

### Secret Management

| Secret | Storage | Rotation |
|--------|---------|----------|
| Identity provider JWKS URL | Configuration file | N/A (public endpoint) |
| Service token client secret | Environment variable or secrets manager | Quarterly |
| Database connection string | Environment variable or secrets manager | Quarterly |
| Redis connection string | Environment variable or secrets manager | Quarterly |
| mTLS client certificates | Mounted from secrets manager | Annually |

**Never in definition files:** Secrets are never stored in definition YAML files.
Definitions are configuration, not credentials.
