# ADR-005: Capability-Based Authorization Instead of RBAC Endpoints

**Status:** Accepted

**Date:** 2025-01-15

---

## Context

The BFF must enforce authorization at multiple levels:

- **Navigation:** Which menu items does the user see?
- **Pages:** Can the user access a specific page?
- **Columns:** Which table columns are visible?
- **Fields:** Which form fields are editable vs. read-only?
- **Actions:** Which buttons/links are shown?
- **Commands:** Can the user execute a specific mutation?
- **Workflow steps:** Can the user advance a specific step?

Traditional RBAC (Role-Based Access Control) at the endpoint level checks whether
a user's role is in an allowed list for each endpoint. This works for coarse-grained
access (admin vs. viewer) but fails for the fine-grained UI filtering described above.

The BFF needs an authorization model that:

1. Supports per-element filtering (column-level, field-level, action-level).
2. Decouples role definitions from the BFF code.
3. Allows different tenants to have different role-to-permission mappings.
4. Can be evaluated efficiently (capabilities are checked on every request).

## Decision

Authorization uses a capability-based model. A `CapabilitySet` (implemented as
`map[string]bool`) represents all permissions the user has for the current request.

**Capability lifecycle:**

```
1. Transport middleware resolves capabilities once per request:
   CapabilityResolver.Resolve(RequestContext) → CapabilitySet

2. CapabilitySet is attached to the request context.

3. Every provider checks capabilities during descriptor generation:
   - MenuProvider: omit navigation nodes where user lacks capabilities
   - PageProvider: reject page access or omit columns/sections
   - FormProvider: mark fields as read-only based on capabilities
   - ActionProvider: omit actions where user lacks capabilities

4. CommandExecutor checks capabilities again at execution time:
   - Reject command if user lacks required capabilities
   (defense-in-depth against compromised frontends)
```

**Capability naming convention:** `{domain}:{resource}:{action}`

```
orders:list:view          # view orders list page
orders:detail:edit        # see edit action on detail page
orders:cancel:execute     # execute cancel command
orders:notes:edit         # edit the notes field specifically
```

**Resolution chain:**

```
JWT Claims (roles, attributes)
  → PolicyEvaluator (maps roles to capabilities)
    → CapabilitySet (flat set of granted capabilities)
```

The `PolicyEvaluator` is pluggable:
- `StaticPolicyEvaluator` reads mappings from a YAML configuration file.
- `OPAPolicyEvaluator` queries an Open Policy Agent instance via HTTP.

## Consequences

### Positive

- **Fine-grained UI filtering:** Capabilities can be attached to any UI element —
  a specific column, a specific field, a specific action. The frontend renders
  exactly what the user is authorized to see.
- **Decoupled from roles:** Role-to-capability mappings are managed by the policy
  engine, not hardcoded in the BFF. Different tenants can have different mappings
  (e.g., the "manager" role in tenant A has different capabilities than in tenant B).
- **Defense-in-depth:** Capabilities are checked at both descriptor generation
  (preventive) and execution (enforcing). A compromised frontend that shows a hidden
  button still cannot execute the command.
- **Cacheable:** The CapabilitySet for a given `(subject_id, tenant_id, partition_id)`
  tuple can be cached (with TTL), so repeated requests from the same user don't
  re-evaluate policies.
- **Policy engine flexibility:** The `PolicyEvaluator` interface abstracts the
  resolution mechanism. Teams can start with static YAML policies and migrate to
  OPA for more complex scenarios without changing the BFF code.

### Negative

- **Capability proliferation:** Large systems may accumulate hundreds of capability
  strings. Requires discipline in naming conventions and documentation. Mitigated by
  the `{domain}:{resource}:{action}` naming convention and per-domain capability
  namespaces.
- **Cache invalidation complexity:** When a user's roles change, cached capabilities
  must be invalidated. The `CapabilityResolver.Invalidate(subjectId, tenantId)` method
  handles this, but the invalidation signal must be propagated (e.g., via webhook
  from the identity provider).
- **Evaluation cost:** Resolving capabilities for every request adds latency.
  Mitigated by caching (default TTL: 5 minutes) and the efficient `map[string]bool`
  lookup (O(1) per capability check).

## Alternatives Considered

### Middleware RBAC (Per-Endpoint Role Checks)

```go
// In route registration:
router.Handle("/ui/pages/orders.list", RequireRole("order_viewer", handler))
router.Handle("/ui/commands/orders.cancel", RequireRole("order_admin", handler))
```

**Rejected because:**
- Cannot filter at sub-page granularity (columns, fields, actions).
- Role checks are scattered across route registrations, making it hard to audit
  the complete authorization surface.
- Role names are hardcoded in Go code — adding a new role or changing mappings
  requires recompilation.
- Different tenants cannot have different role definitions without code changes.

### Endpoint-Level Roles in Definitions

```yaml
commands:
  - id: orders.cancel
    allowed_roles: ["order_admin", "order_manager"]
```

**Rejected because:**
- Roles are defined by the identity provider, not the BFF. Embedding role names
  in definitions creates coupling between the definition schema and the identity
  system.
- Does not support attribute-based policies (e.g., "manager can only cancel orders
  in their region") without extending the role model.
- Forces a flat RBAC model — cannot express hierarchical or composite permissions
  without adding complexity to the definition schema.

### Pass-Through to Backend Authorization

Let the backend services handle all authorization. The BFF forwards the user token
and lets the backend return 403 for unauthorized operations.

**Rejected because:**
- The frontend would render buttons and fields that the user cannot use, leading to
  a confusing UI experience.
- Every backend call (even unauthorized ones) would incur network latency before
  being rejected.
- Different backend services would enforce authorization differently, leading to
  inconsistent user experiences.
- The BFF cannot generate filtered descriptors without knowing the user's permissions.

## References

- [Principle P5: Capabilities Gate Everything](../01-principles-and-invariants.md)
- [Doc 18: CapabilityResolver, PolicyEvaluator interfaces](../18-core-abstractions-and-interfaces.md)
- [Doc 17: Privilege Escalation Prevention](../17-security-model.md)
