# 07 — Capabilities and Permissions

This document describes the capability-based authorization model used by Thesa:
how capabilities are defined, resolved, cached, and evaluated to control every
aspect of the UI.

---

## What Is a Capability?

A **capability** is a UI-level permission string that gates access to a specific
UI element or action. Capabilities use a hierarchical, colon-separated format:

```
{namespace}:{resource}:{action}
```

### Examples

| Capability | Meaning |
|-----------|---------|
| `orders:list:view` | Can see the orders list page |
| `orders:detail:view` | Can see order detail pages |
| `orders:detail:edit` | Can see edit actions on order detail |
| `orders:cancel:execute` | Can execute order cancellation |
| `orders:notes:view` | Can see the notes section |
| `orders:notes:edit` | Can edit the notes field (field-level) |
| `orders:approve:execute` | Can approve orders (workflow) |
| `orders:export:execute` | Can export orders (bulk action) |
| `orders:*` | Wildcard: all orders capabilities |
| `*` | Superadmin: all capabilities |

### Three-Part Format

- **Namespace:** The domain scope. Matches the domain ID in definitions. Each domain
  team owns their namespace. Examples: `orders`, `inventory`, `customers`, `ledger`.

- **Resource:** The specific resource or UI construct within the namespace. Examples:
  `list`, `detail`, `notes`, `line_items`, `approve`, `cancel`.

- **Action:** The permitted action on the resource. Common actions:
  - `view` — can see the element
  - `edit` — can modify data through the element
  - `execute` — can trigger commands or workflows
  - `delete` — can delete through the element
  - `export` — can export data

### Two-Part Format (Shorthand)

For simple cases, a two-part format is also supported:

```
{namespace}:{action}
```

Example: `admin:access` — grants general admin access.

### Wildcard Matching

Wildcards allow broad grants:

| Pattern | Matches |
|---------|---------|
| `orders:*` | Any capability starting with `orders:` |
| `orders:list:*` | `orders:list:view`, `orders:list:export`, etc. |
| `*` | Every capability (superadmin) |

Wildcard matching is resolved at evaluation time, not at definition time.

---

## Capabilities vs. Backend Permissions

Capabilities are NOT the same as backend service permissions. They are a separate,
UI-specific authorization layer.

| Aspect | Capabilities (UI) | Backend Permissions |
|--------|-------------------|---------------------|
| Purpose | Control what the UI shows | Control what the API allows |
| Granularity | Per-page, per-field, per-action | Per-endpoint, per-resource |
| Enforcement | BFF filters descriptors | Backend rejects unauthorized requests |
| Authority | BFF policy evaluator | Backend authorization middleware |
| Scope | UI constructs | API operations |

**The BFF's capability check is a pre-filter, not a security boundary.** Even if a
buggy frontend sends a command the user shouldn't have access to, the backend service
still enforces its own authorization.

However, the capability system provides:
- **Better UX:** Users never see buttons they can't click.
- **Defense in depth:** Two independent authorization checks.
- **Reduced backend load:** Unauthorized commands are rejected at the BFF without
  calling the backend.

### Mapping Between Layers

The policy engine maintains the mapping between backend permissions and UI capabilities.

Example:
```
Backend permission: "orders.admin"
  ↓ maps to UI capabilities:
    orders:list:view
    orders:detail:view
    orders:detail:edit
    orders:cancel:execute
    orders:approve:execute
    orders:notes:view
    orders:notes:edit

Backend permission: "orders.viewer"
  ↓ maps to UI capabilities:
    orders:list:view
    orders:detail:view
    orders:notes:view
```

This mapping is defined in the policy engine, not in Thesa definitions.

---

## CapabilitySet

A `CapabilitySet` is the resolved set of all capabilities a user has in a given context.
It is a simple set (map of strings to booleans) with helper methods.

### Structure

```
CapabilitySet = map[string]bool

Example:
{
  "orders:list:view": true,
  "orders:detail:view": true,
  "orders:detail:edit": true,
  "orders:notes:view": true,
  "inventory:list:view": true,
}
```

### Operations

| Method | Description |
|--------|-------------|
| `Has(cap)` | Returns true if the exact capability is in the set |
| `HasAll(caps)` | Returns true if ALL given capabilities are in the set |
| `HasAny(caps)` | Returns true if ANY of the given capabilities are in the set |
| `Matches(cap)` | Returns true if the capability matches, including wildcards |
| `Merge(other)` | Adds all capabilities from another set |

### Example Evaluations

Given the set above:

| Check | Result | Reason |
|-------|--------|--------|
| `Has("orders:list:view")` | true | Exact match |
| `Has("orders:cancel:execute")` | false | Not in set |
| `HasAll(["orders:list:view", "orders:detail:view"])` | true | Both present |
| `HasAll(["orders:list:view", "orders:cancel:execute"])` | false | cancel not present |
| `HasAny(["orders:cancel:execute", "orders:list:view"])` | true | list is present |

---

## CapabilityResolver

The CapabilityResolver is responsible for resolving the full set of capabilities
for a given RequestContext. It is the primary authorization interface used by
all BFF components.

### Interface

```
CapabilityResolver
  ├── Resolve(RequestContext) → (CapabilitySet, error)
  │     Returns the full capability set for the given context.
  │
  └── Invalidate(subjectId string, tenantId string)
        Clears cached capabilities for the given user/tenant.
```

### Resolution Flow

```
Resolve(requestContext):
  │
  ├── 1. Compute cache key: hash(subjectId + tenantId + partitionId)
  │
  ├── 2. Check cache:
  │      If hit and not expired → return cached CapabilitySet
  │
  ├── 3. Call PolicyEvaluator.ResolveCapabilities(requestContext)
  │      │
  │      ├── a. Map roles to capabilities:
  │      │      For each role in requestContext.Roles:
  │      │        Look up role → capability mappings
  │      │        Add to set
  │      │
  │      ├── b. Apply tenant-specific overrides:
  │      │      Some tenants may have custom role → capability mappings
  │      │      (e.g., tenant A has "orders:approve" for managers,
  │      │       tenant B requires "orders:approve" only for directors)
  │      │
  │      ├── c. Apply partition-specific overrides:
  │      │      Some partitions may restrict or expand capabilities
  │      │      (e.g., "staging" partition allows all capabilities,
  │      │       "production" partition restricts destructive actions)
  │      │
  │      ├── d. Apply attribute-based rules:
  │      │      User attributes (department, level, etc.) may grant
  │      │      additional capabilities
  │      │
  │      └── e. Return merged CapabilitySet
  │
  ├── 4. Cache result with TTL
  │
  └── 5. Return CapabilitySet
```

### Cache Configuration

```yaml
capability_cache:
  enabled: true
  ttl: 60s                    # How long to cache resolved capabilities
  max_entries: 10000           # Maximum cache entries (LRU eviction)
  key_strategy: "subject+tenant+partition"  # Cache key components
```

### Cache Invalidation

Capabilities may change when:
- An admin changes a user's roles → Invalidate for that user.
- A tenant's role mappings change → Invalidate all users in that tenant.
- The policy engine is updated → Invalidate all entries.

Invalidation can be:
- **TTL-based:** Changes propagate within the cache TTL (eventual consistency).
  This is the simplest approach and sufficient for most cases.
- **Event-driven:** Listen for pub/sub events from the policy engine and
  selectively invalidate affected entries (faster propagation).

---

## PolicyEvaluator

The PolicyEvaluator is the backend implementation that actually resolves capabilities.
Thesa supports multiple implementation strategies.

### Interface

```
PolicyEvaluator
  ├── ResolveCapabilities(RequestContext) → (CapabilitySet, error)
  │     Resolves the full capability set from roles, tenant, and attributes.
  │
  ├── Evaluate(RequestContext, capability string, resource map[string]any) → (bool, error)
  │     Evaluates a single capability with optional resource context.
  │     Used for fine-grained checks (e.g., "can this user edit THIS order?").
  │
  ├── EvaluateAll(RequestContext, capabilities []string, resource map[string]any) → (map[string]bool, error)
  │     Evaluates multiple capabilities at once (batch optimization).
  │
  └── Sync() → error
        Refreshes policy data (e.g., reload from policy engine).
```

### Implementation Strategy 1: Static Configuration

For simpler deployments, capabilities are mapped from roles in a configuration file:

```yaml
# policy.yaml
roles:
  admin:
    capabilities:
      - "orders:*"
      - "inventory:*"
      - "customers:*"

  order_manager:
    capabilities:
      - "orders:list:view"
      - "orders:detail:view"
      - "orders:detail:edit"
      - "orders:cancel:execute"
      - "orders:approve:execute"
      - "orders:notes:view"
      - "orders:notes:edit"

  order_viewer:
    capabilities:
      - "orders:list:view"
      - "orders:detail:view"
      - "orders:notes:view"

  inventory_manager:
    capabilities:
      - "inventory:*"
```

Resolution: iterate user's roles, merge all matching capabilities.

### Implementation Strategy 2: External Policy Engine (OPA)

For complex deployments, delegate to Open Policy Agent (OPA) or similar:

```
BFF → POST http://opa.internal:8181/v1/data/thesa/capabilities
  Body: {
    "input": {
      "subject_id": "user-123",
      "tenant_id": "acme-corp",
      "partition_id": "us-production",
      "roles": ["order_manager"],
      "claims": { "department": "operations", "level": 3 }
    }
  }

OPA Response:
  {
    "result": {
      "capabilities": [
        "orders:list:view",
        "orders:detail:view",
        "orders:detail:edit",
        ...
      ]
    }
  }
```

OPA policies can implement complex rules:
- Role-based: `role == "admin"` → all capabilities
- Attribute-based: `claims.level >= 3` → approval capabilities
- Tenant-specific: `tenant_id == "acme-corp"` → specific overrides
- Time-based: `current_time.hour >= 9 && current_time.hour < 17` → business hours only

### Implementation Strategy 3: Database-Backed

For systems with a permission management UI, capabilities are stored in a database:

```sql
-- role_capabilities table
SELECT capability
FROM role_capabilities rc
JOIN user_roles ur ON ur.role_id = rc.role_id
WHERE ur.user_id = ? AND ur.tenant_id = ?
```

---

## Where Capabilities Are Evaluated

Capabilities are evaluated at every level of the UI descriptor hierarchy. Here is a
complete map of where evaluation happens and what effect a missing capability has.

### Navigation (Menu Generation)

```
For each domain:
  If user lacks domain.navigation.capabilities → skip entire domain

  For each navigation child:
    If user lacks child.capabilities → skip this menu item
```

**Effect:** Domain or menu item is not included in the NavigationTree sent to frontend.
The user never knows it exists.

### Pages

```
GET /ui/pages/{pageId}:
  Load PageDefinition
  If user lacks page.capabilities → return 403 Forbidden
```

**Effect:** Hard denial. If the user navigates directly to the URL (e.g., from a bookmark),
they get a 403 error page.

### Table Columns

```
For each column in table:
  If column.visible is a capability string:
    If user lacks that capability → omit column from TableDescriptor
```

**Effect:** Column is silently removed. Frontend renders table without that column.
Use case: hide sensitive columns (e.g., "Internal Cost") from external users.

### Table Filters

```
Filters that reference columns with capability-based visibility are also removed
when the column is removed.
```

### Sections

```
For each section:
  If section.capabilities defined:
    If user lacks any → omit section from descriptor
```

**Effect:** Entire section (with all its fields) is not sent to frontend.

### Fields

```
For each field:
  If field.visibility is a capability string:
    If user lacks capability → omit field from descriptor

  If field.read_only is a capability string:
    If user HAS capability → field.read_only = false (editable)
    If user LACKS capability → field.read_only = true (read-only)
```

**Effect:** Field is either hidden entirely or made read-only.

### Actions

```
For each action:
  If user lacks any action.capabilities → omit action from descriptor
```

**Effect:** Button/menu item is not shown. User cannot trigger the action.

### Commands (Execution Time)

```
POST /ui/commands/{commandId}:
  Load CommandDefinition
  If user lacks any command.capabilities → return 403 Forbidden
```

**Effect:** Hard denial at execution time. Defense in depth — even if the frontend
somehow displays the button, the command is rejected.

### Workflows (Start Time)

```
POST /ui/workflows/{workflowId}/start:
  Load WorkflowDefinition
  If user lacks any workflow.capabilities → return 403 Forbidden
```

### Workflow Steps (Advance Time)

```
POST /ui/workflows/{instanceId}/advance:
  Load current StepDefinition
  If step.capabilities defined:
    If user lacks any → return 403 Forbidden
```

**Effect:** User cannot advance past a step they're not authorized for. Another user
with the right capabilities must advance that step.

### Search Providers

```
For each SearchDefinition:
  If user lacks any search.capabilities → skip this provider
```

**Effect:** Search results from this domain are not included. The user doesn't see
results they couldn't access.

---

## Capability Namespacing

### Rules

1. Each domain owns its capability namespace (matching its domain ID).
2. A definition may only declare capabilities under its own namespace.
3. Cross-namespace capability references are not allowed in definitions
   (but the policy engine can grant cross-domain capabilities).

### Enforcement at Validation Time

```
# Validation rule:
For each capability string in definition with domain "orders":
  Verify capability starts with "orders:"
  → Fatal if not: "Capability 'inventory:list:view' in orders domain crosses namespace boundary"
```

### Shared Capabilities

For capabilities that span domains (e.g., `admin:access`), define them in a
`_shared` domain or use a dedicated `platform` namespace:

```yaml
# definitions/_shared/capabilities.yaml
domain: "_platform"
# This domain has no pages/forms/commands — it just declares shared capabilities
# that the policy engine uses for cross-domain grants.
```

---

## Row-Level and Resource-Level Policies

For some use cases, the capability set alone is insufficient. The question isn't
"can this user view orders?" but "can this user view THIS specific order?"

### When Resource-Level Policies Are Needed

| Scenario | Simple Capability | Resource-Level Policy |
|----------|-------------------|----------------------|
| Can user see the orders page? | `orders:list:view` | Not needed |
| Can user see order ORD-123? | Not sufficient | Check if user owns the order or is in the assigned team |
| Can user cancel THIS order? | `orders:cancel:execute` | Check if order status allows cancellation AND user is assigned |
| Can user approve amounts > $10K? | Not sufficient | Check order amount against user's approval limit |

### PolicyEvaluator for Resource-Level Checks

The `Evaluate` method on PolicyEvaluator accepts a resource context:

```
PolicyEvaluator.Evaluate(
  requestContext,
  "orders:cancel:execute",
  resource: { "order_id": "ord-123", "status": "pending", "amount": 5000, "assigned_to": "user-123" }
) → true/false
```

The policy engine evaluates:
- Does the user have the `orders:cancel:execute` capability? (base check)
- Is the order status "pending"? (state-based rule)
- Is the order assigned to this user or a team they belong to? (ownership rule)
- Is the amount within the user's authority limit? (attribute-based rule)

### When Resource-Level Checks Happen

- **NOT during descriptor generation** — descriptors are generated without knowing
  which specific resource the user will interact with.
- **During command execution** — when the user submits a command, the resource is
  known and can be checked.
- **During workflow advancement** — the workflow state provides resource context.
- **Optionally during data loading** — for row-level filtering (but this is usually
  better handled by the backend service).

---

## Testing Capabilities

### Unit Testing

Test capability resolution with mock policy evaluators:

```go
// Create a mock evaluator that returns fixed capabilities
mockEvaluator := &StaticPolicyEvaluator{
    Capabilities: model.CapabilitySet{
        "orders:list:view":    true,
        "orders:detail:view":  true,
    },
}

// Create resolver with mock
resolver := capability.NewResolver(mockEvaluator, cache)

// Test resolution
caps, err := resolver.Resolve(testContext)
assert.True(t, caps.Has("orders:list:view"))
assert.False(t, caps.Has("orders:cancel:execute"))
```

### Integration Testing

Test end-to-end with different user roles:

```
Test: "order_viewer role sees list page but not edit action"
  1. Authenticate as user with role "order_viewer"
  2. GET /ui/pages/orders.list
  3. Assert: page descriptor returned (200)
  4. Assert: no "edit" action in page.actions
  5. POST /ui/commands/orders.update
  6. Assert: 403 Forbidden

Test: "order_manager role sees edit action"
  1. Authenticate as user with role "order_manager"
  2. GET /ui/pages/orders.list
  3. Assert: "edit" action present in page.actions
  4. POST /ui/commands/orders.update with valid input
  5. Assert: 200 OK
```

### Capability Coverage Analysis

It's useful to verify that every UI element has appropriate capability guards.
A lint tool can check definitions for:

- Pages without capabilities (warning: page accessible to everyone).
- Actions without capabilities (warning: action accessible to everyone).
- Commands without capabilities (error: mutations must be guarded).
- Workflows without capabilities (error: workflows must be guarded).
