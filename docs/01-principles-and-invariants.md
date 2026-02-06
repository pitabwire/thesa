# 01 — Core Principles and Invariants

This document defines the non-negotiable architectural rules of Thesa. Every design
decision, code review, and operational procedure must be evaluated against these
principles. Violations are treated as bugs — not style issues, not technical debt,
but defects that must be fixed before deployment.

---

## P1: No Implicit Exposure

**Rule:** Every backend operation visible to the UI must appear in an explicit
UI Exposure Definition. Loading a new OpenAPI specification does NOT make its
operations available to the frontend.

**Rationale:** In a multi-tenant enterprise system, accidentally exposing an
internal API to the frontend could leak sensitive data, create unauthorized mutation
paths, or violate compliance requirements. The definition file is the allowlist —
if an operation is not in a definition, it does not exist from the frontend's
perspective.

**What This Means in Practice:**

- When a new backend service is deployed with an OpenAPI spec, nothing changes in
  the frontend until a domain team writes a definition file that references specific
  operations from that spec.
- The BFF validates at startup that every `operation_id` in a definition exists in
  the loaded OpenAPI specs. But the reverse is NOT true — operations in specs that
  are not referenced by any definition are simply invisible.
- There is no "auto-expose all operations" mode. There will never be one. This is
  by design, not by omission.

**Anti-Patterns:**
- Do NOT write a generic proxy that forwards all requests to backend services.
- Do NOT create a "discovery" endpoint that lists all available operations.
- Do NOT auto-generate definitions from OpenAPI specs (manual definition authoring
  is the deliberate security gate).

**Verification:** At startup, the BFF logs the count of operations in OpenAPI specs
vs. the count of operations referenced in definitions. A large gap is expected and
healthy — it means the backend has more capabilities than the UI exposes.

---

## P2: Frontend Receives Descriptors, Not Definitions

**Rule:** Definitions are internal configuration. The frontend receives descriptors —
resolved, filtered, UI-safe projections of definitions. Descriptors never contain
backend URLs, operation IDs, internal field names, or capability strings.

**Rationale:** If the frontend had access to definitions, it would know:
- The internal URLs of backend services (security risk).
- The internal operation IDs (allows reverse-engineering the API surface).
- The capability strings (allows crafting privilege escalation attempts).
- The mapping between UI field names and backend field names (tight coupling).

By sending only descriptors, the frontend is completely decoupled from backend
internals. It knows what to render and what actions to offer, but not how those
actions are fulfilled.

**What This Means in Practice:**

- When a definition says `operation_id: "listOrders"`, the descriptor says
  `data_endpoint: "/ui/pages/orders.list/data"`.
- When a definition says `capabilities: ["orders:edit:execute"]`, the descriptor
  simply omits the action if the user lacks the capability. The frontend never
  sees capability strings.
- When a definition says `field_map: { order_number: "orderNumber" }`, the
  descriptor and data responses use `order_number`. The backend field name
  `orderNumber` never reaches the frontend.

**Testing:** Every descriptor response should be validated against a schema that
explicitly rejects fields like `operation_id`, `handler`, `capabilities`,
`service_id`, or any URL pointing to a backend service.

---

## P3: Tenant Isolation Is Structural

**Rule:** Tenant ID is extracted from the verified JWT token and injected into every
downstream call. No code path allows a request to operate on a tenant other than the
authenticated one. This is enforced at the middleware level, not by individual handlers.

**Rationale:** In a multi-tenant system, cross-tenant data access is the highest-severity
security vulnerability. A single bug in a single handler that reads tenant ID from a
query parameter could expose all tenants' data. By extracting tenant ID exclusively
from the verified token and propagating it structurally, cross-tenant access becomes
architecturally impossible, not merely unlikely.

**What This Means in Practice:**

- The `X-Tenant-Id` header sent to backend services is ALWAYS derived from the JWT
  `tenant_id` claim. It is never read from the frontend's request headers or body.
- The `WorkflowStore` query layer includes `tenant_id` in every WHERE clause. There is
  no `GetByID(id)` method — only `Get(tenantId, id)`.
- If a user attempts to access a workflow instance belonging to a different tenant,
  they receive a 404 (not 403), preventing tenant ID enumeration.
- Partition (workspace) ID IS accepted from a request header, but only after validation
  that the partition belongs to the authenticated tenant.

**Anti-Patterns:**
- `req.Header.Get("X-Tenant-Id")` — NEVER as a source of truth. Only for backend
  propagation after extraction from JWT.
- `store.Get(instanceId)` without tenant scoping — NEVER. Always scope by tenant first.
- `403 Forbidden: wrong tenant` — NEVER. Use 404 to prevent enumeration.

**Verification:** Write integration tests that authenticate as Tenant A and attempt to
access resources belonging to Tenant B. Every such attempt must return 404.

---

## P4: Commands Are the Only Mutation Path

**Rule:** The frontend cannot invoke backend APIs directly. All mutations go through
`POST /ui/commands/{commandId}`, which validates, authorizes, maps, invokes, and
translates the operation.

**Rationale:** Uniform enforcement. If mutations could happen through arbitrary
endpoints, each endpoint would need its own authorization, validation, audit logging,
rate limiting, and error translation logic. By funneling all mutations through a
single pipeline, these concerns are implemented once and applied universally.

**What This Means in Practice:**

- The frontend has exactly ONE way to change data: `POST /ui/commands/{commandId}`.
- Even "simple" operations like deleting a record go through a command definition.
- Even operations triggered by workflows go through the same invocation pipeline
  (the workflow engine uses the same `OperationInvoker` that commands use).
- There are no `PUT /ui/orders/{id}` or `DELETE /ui/orders/{id}` endpoints. Those
  are backend concerns, hidden behind command definitions.

**Benefits:**
- Every mutation is logged with the same structured format.
- Every mutation is traced with the same span hierarchy.
- Every mutation is metered with the same metric labels.
- Every mutation is capability-checked through the same pipeline.
- Idempotency is available for every mutation, not just some.

**The Exception:** Data-fetching operations (GET requests for page data, form data,
lookup data, search) do NOT go through the command pipeline. They have their own
read-oriented pipeline with capability checks but without idempotency or rate limiting.

---

## P5: Capabilities Gate Everything

**Rule:** Every navigation node, page, table column, form field, action, workflow step,
and search provider declares required capabilities. The BFF evaluates these capabilities
and filters all descriptors before serving them to the frontend.

**Rationale:** The frontend should never display a button the user cannot click, a page
the user cannot see, or a field the user cannot edit. This prevents confusion, reduces
support tickets ("why can't I click this?"), and provides defense-in-depth (even if the
frontend has a bug that shows a forbidden button, the BFF rejects the command).

**What This Means in Practice:**

- A page definition declares `capabilities: ["orders:list:view"]`. If the user lacks
  this capability, the BFF returns 403 for `GET /ui/pages/orders.list`.
- A column definition declares `visible: "orders:sensitive_data:view"`. If the user
  lacks this capability, the column is omitted from the TableDescriptor.
- An action definition declares `capabilities: ["orders:cancel:execute"]`. If the user
  lacks this capability, the action is omitted from the ActionDescriptor list.
- A field definition declares `read_only: "orders:notes:edit"`. If the user lacks this
  capability, the field appears with `read_only: true` in the FieldDescriptor.
- A command definition declares `capabilities: ["orders:update:execute"]`. If the user
  lacks this capability, the BFF returns 403 for `POST /ui/commands/orders.update`.

**Capability evaluation happens at TWO levels:**
1. **Descriptor generation** (preventive) — filter out elements the user can't use.
2. **Execution time** (enforcing) — reject commands/workflow advances if capabilities
   are missing, even if the frontend somehow sends them.

This defense-in-depth ensures that even a compromised or buggy frontend cannot execute
unauthorized operations.

---

## P6: Definitions Are Validated at Startup

**Rule:** Every definition is validated against the loaded OpenAPI specifications at
process startup. If a definition references a non-existent `operationId`, the process
fails to start. This provides a compile-time-like guarantee for a runtime-configured system.

**Rationale:** Runtime discovery of misconfigured definitions is catastrophic in
production. If a command definition references `operation_id: "updteOrder"` (typo),
the error should be caught at startup, not when a user clicks "Update" in the UI.

**What This Means in Practice:**

- The BFF startup sequence includes a mandatory validation phase.
- Validation checks include:
  - Every `operation_id` reference resolves to an existing operation in the OpenAPI index.
  - Every `handler` reference (for SDK bindings) resolves to a registered SDK handler.
  - Every workflow transition references existing step IDs.
  - Every form's `submit_command` references an existing command definition.
  - Every action's `command_id`, `form_id`, `workflow_id` references exist.
  - Capability strings match the expected format.
  - IDs are unique within and across domains.
- If ANY fatal validation error is found, the process logs the errors and exits with
  a non-zero status code. It does not start serving traffic.

**CI Integration:** The same validation logic should be available as a CLI tool or
library that domain teams run in their CI pipelines. This catches errors before
they reach deployment.

**Warning-level validations** (non-fatal) are also performed:
- Orphan forms (forms not referenced by any page, action, or workflow).
- Response mapping paths that don't match the OpenAPI response schema.
- Unreachable workflow steps.

These are logged as warnings but do not prevent startup.

---

## P7: Backend Evolution Does Not Break the Frontend

**Rule:** The BFF contains an adapter layer that maps between backend response shapes
and UI descriptor contracts. Backend teams can evolve their APIs; the definition's
response mapping absorbs the change. The frontend contract remains stable.

**Rationale:** In an organization with multiple backend teams deploying independently,
the frontend cannot be redeployed every time a backend field is renamed or a response
structure changes. The BFF's mapping layer provides a stable frontier.

**What This Means in Practice:**

- If the Orders service renames `orderNumber` to `order_num`, the definition author
  updates the field_map: `order_number: "order_num"`. The frontend continues to
  receive `order_number`. No frontend change needed.
- If the Orders service adds pagination metadata under `meta.pagination.total` instead
  of `meta.total`, the definition author updates `total_path: "meta.pagination.total"`.
  The frontend's DataResponse format is unchanged.
- If the Orders service changes its error code from `ORDER_NOT_FOUND` to `NOT_FOUND`,
  the definition author updates the error_map. The frontend error display is unchanged.

**Limits:** The adapter layer handles field renaming, path changes, and structure
reorganization. It does NOT handle semantic changes (e.g., a field that changes from
containing a price in cents to a price in dollars). Semantic changes require coordinated
updates to the definition mapping and possibly the frontend.

---

## P8: Workflows Are Persistent and Resumable

**Rule:** Workflow instances are persisted to a durable store. A server restart, user
session change, or network interruption does not lose workflow state. Every workflow step
is idempotent and authorized independently.

**Rationale:** Multi-step processes like approvals, dispute resolution, and onboarding
can span hours or days. They involve multiple users (the submitter and the approver may
be different people). The workflow state must survive:
- BFF process restarts (deployments, crashes).
- User session changes (user logs out, switches device, token expires and renews).
- Multiple BFF instances behind a load balancer (request may hit different instance).

**What This Means in Practice:**

- Every workflow state change is persisted to the WorkflowStore before the response is
  sent to the frontend.
- If a system step (automatic backend invocation) fails mid-execution and the BFF
  restarts, the workflow remains in the last persisted state. The step can be retried.
- If the frontend loses connection during a workflow advance, it can call
  `GET /ui/workflows/{instanceId}` to retrieve the current state and resume.
- Workflow step execution is idempotent: advancing to the same step with the same
  event twice produces the same result.

**WorkflowStore implementations:**
- PostgreSQL (recommended for production) — durable, shared across instances.
- In-memory (for testing only) — fast but lost on restart.

---

## P9: Definition Files Are Integrity-Checked

**Rule:** Definitions are checksummed (SHA-256) at load time. Any tampering between load
and runtime is detectable. In production, definitions should be loaded from a read-only
filesystem or a verified artifact store.

**Rationale:** Definition files control what the UI can do. A tampered definition could:
- Expose unauthorized backend operations.
- Bypass capability requirements.
- Redirect commands to malicious endpoints (if service URLs were in definitions, which
  they are not — they're in service configuration, but the principle stands).

**What This Means in Practice:**

- At startup, the BFF computes SHA-256 of each definition file.
- The checksums are stored in memory and included in the health check response.
- In production, definitions can be verified against a manifest file signed by CI/CD.
- If hot-reload is enabled, checksum changes are logged as audit events.
- In strict mode, hot-reload rejects changes that don't match a signed manifest.

**In Production:**
- Mount definitions from a read-only volume.
- Use immutable container images that bundle definitions.
- Or load definitions from a versioned artifact store (e.g., S3 with versioning).

---

## P10: No Recompilation for New APIs

**Rule:** Adding a new backend service, new pages, new commands, or new workflows requires
only adding/modifying YAML definition files and (for OpenAPI services) providing the spec.
The BFF binary does not change.

**Rationale:** If the BFF binary needed to change for every new API, it would become a
deployment bottleneck. Every domain team would need to coordinate with the BFF team for
every change. This defeats the purpose of a platform.

**What This Means in Practice:**

- A domain team can onboard a completely new service by:
  1. Deploying their service with an OpenAPI spec.
  2. Adding their OpenAPI spec to the BFF's spec directory.
  3. Writing a definition YAML file.
  4. Deploying (or triggering hot-reload of) the definitions.
  No BFF code changes. No BFF recompilation. No BFF redeployment (if hot-reload is enabled).

**The Exception:** Adding a new SDK handler (for streaming, high-integrity, or complex
orchestration use cases) DOES require recompilation. SDK handlers are compiled Go code.
This is intentional — these are high-integrity paths where compile-time guarantees are
valued over deployment flexibility.

---

## Summary Matrix

| # | Principle | Core Concern | Violated By |
|---|-----------|-------------|-------------|
| P1 | No implicit exposure | Security | Auto-exposing APIs, discovery endpoints |
| P2 | Descriptors only | Decoupling | Sending definitions or backend details to frontend |
| P3 | Structural tenant isolation | Security | Reading tenant from input; unscoped queries |
| P4 | Commands only | Consistency | Multiple mutation endpoints; direct backend calls |
| P5 | Capabilities gate everything | Authorization | Unguarded pages, actions, or fields |
| P6 | Validated at startup | Reliability | Deferring validation to runtime |
| P7 | Backend isolation | Stability | Forwarding backend responses directly |
| P8 | Persistent workflows | Durability | In-memory-only workflow state in production |
| P9 | Integrity-checked definitions | Security | Unverified definition sources |
| P10 | No recompilation | Extensibility | Hardcoded routes, handlers, or mappings |
