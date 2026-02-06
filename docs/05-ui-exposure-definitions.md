# 05 — UI Exposure Definitions

This document describes the definition system: what definitions are, how they are
authored, loaded, validated, and managed. For the complete YAML schema of each
definition type, see [06 — Definition Schema Reference](06-definition-schema-reference.md).

---

## What Is a Definition?

A **UI Exposure Definition** is a declarative YAML file that describes how backend
operations are exposed to the frontend as UI constructs. It is the bridge between
backend capabilities and frontend rendering.

A definition answers: _"Given this backend operation, what should the UI show,
who can see it, and how should user input map to backend parameters?"_

### Definitions vs. Backend APIs

Backend APIs describe **what the backend can do.** Definitions describe **what the
UI should show.** These are deliberately separate concerns.

A backend service might expose 50 OpenAPI operations. A definition might expose
only 12 of them to the UI. The remaining 38 operations might be:
- Internal endpoints used only by other services.
- Admin endpoints not needed in the user-facing UI.
- Deprecated endpoints being phased out.
- Endpoints for features not yet designed in the UI.

### Definitions vs. Descriptors

Definitions are **internal configuration** loaded at startup. They contain backend-
specific details (operation IDs, field mappings, capability strings) that the frontend
should never see.

Descriptors are **external contracts** sent to the frontend at runtime. They are the
resolved, filtered, UI-safe projections of definitions. See [08 — UI Descriptor Model](08-ui-descriptor-model.md).

---

## Definition Ownership Model

Definitions are owned by **domain teams**, not by the BFF team or the frontend team.

```
┌──────────────────────────────────────────────────────────────┐
│                     Ownership Model                           │
│                                                              │
│  Orders Team          │  Inventory Team      │  BFF Team     │
│  ──────────           │  ──────────────      │  ────────     │
│  ✓ orders-svc API     │  ✓ inventory-svc API │  ✓ BFF core   │
│  ✓ orders-svc spec    │  ✓ inventory spec    │  ✓ Transport  │
│  ✓ orders definition  │  ✓ inventory defn    │  ✓ Engine     │
│  ✓ orders capabilities│  ✓ inventory caps    │  ✗ Definitions│
│                       │                      │               │
│  Contributes YAML     │  Contributes YAML    │  Reviews &    │
│  to BFF definitions/  │  to BFF definitions/ │  validates    │
└──────────────────────────────────────────────────────────────┘
```

The domain team:
- Writes and maintains their definition YAML.
- Decides which operations to expose to the UI.
- Defines the capability requirements.
- Defines the field mappings and transformations.
- Submits definition changes through the standard code review process.

The BFF team:
- Maintains the BFF core code.
- Reviews definition changes for structural correctness.
- Does NOT need to understand domain-specific business logic.

The frontend team:
- Consumes the descriptors produced from definitions.
- Does NOT need to read or modify definition files.
- Coordinates with domain teams on new UI features.

---

## File Organization

### Directory Structure

```
definitions/
  ├── orders/
  │   └── definition.yaml          # Complete definition for the Orders domain
  ├── inventory/
  │   └── definition.yaml          # Complete definition for Inventory
  ├── customers/
  │   └── definition.yaml          # Complete definition for Customers
  ├── ledger/
  │   └── definition.yaml          # Complete definition for Ledger
  └── _shared/
      └── lookups.yaml             # Shared lookup definitions (optional)
```

### Single File per Domain (Recommended)

Each domain has a single `definition.yaml` containing all of its pages, forms,
commands, workflows, searches, and lookups. This keeps related definitions together
and makes it easy to understand the full scope of a domain's UI surface.

### Multi-File per Domain (Alternative)

For very large domains, the definition can be split into multiple files:

```
definitions/
  └── orders/
      ├── pages.yaml
      ├── forms.yaml
      ├── commands.yaml
      ├── workflows.yaml
      └── searches.yaml
```

The DefinitionLoader merges all files within a domain directory into a single
DomainDefinition. The `domain` field must be the same across all files.

### Naming Conventions

- **Domain IDs:** lowercase, alphanumeric, hyphens allowed. Examples: `orders`,
  `inventory`, `customer-support`.
- **Page IDs:** `{domain}.{name}`. Examples: `orders.list`, `orders.detail`.
- **Form IDs:** `{domain}.{name}_form`. Examples: `orders.edit_form`, `orders.create_form`.
- **Command IDs:** `{domain}.{action}`. Examples: `orders.update`, `orders.cancel`.
- **Workflow IDs:** `{domain}.{name}`. Examples: `orders.approval`, `orders.cancellation`.
- **Search IDs:** `{domain}.search`. Examples: `orders.search`.
- **Lookup IDs:** `{domain}.{name}`. Examples: `orders.statuses`, `customers.search`.

---

## Definition Loading

### DefinitionLoader Interface

```
DefinitionLoader
  ├── LoadAll(directories []string) → ([]DomainDefinition, error)
  │     Scans all directories, loads all YAML files, returns parsed definitions.
  │
  ├── LoadFile(path string) → (DomainDefinition, error)
  │     Loads a single file, returns parsed definition.
  │
  └── Watch(directories []string, onChange func([]DomainDefinition)) → (stop func, error)
        Watches directories for changes, calls onChange with new definitions.
```

### Loading Sequence

```
LoadAll(["definitions/"]):
  1. Scan directory recursively for *.yaml and *.yml files.
  2. Group files by parent directory (= domain).
  3. For each file:
     a. Read file contents.
     b. Compute SHA-256 checksum of raw bytes.
     c. Parse YAML into DomainDefinition struct.
     d. Set DomainDefinition.Checksum = computed hash.
     e. If parsing fails: return error with file path and parse error.
  4. If multiple files per domain: merge into single DomainDefinition.
     - Concatenate Pages, Forms, Commands, etc.
     - Navigation must be in exactly one file per domain.
  5. Return all loaded DomainDefinitions.
```

### Error Handling During Loading

| Error | Behavior |
|-------|----------|
| File not readable | Fatal — log error, abort startup |
| Invalid YAML syntax | Fatal — log error with line number, abort startup |
| Missing required field | Fatal — log field path, abort startup |
| Duplicate ID within domain | Fatal — log both occurrences, abort startup |
| Duplicate ID across domains | Fatal — log both occurrences, abort startup |
| Unknown field in YAML | Warning — log warning, continue (allows forward compatibility) |

---

## Definition Validation

After loading, every definition undergoes validation. Validation has two levels:
**structural** (YAML structure and required fields) and **referential** (references
to OpenAPI operations, forms, commands, etc.).

### Structural Validation

Structural validation checks that the definition conforms to the expected schema.

| Check | Applies To | Severity |
|-------|-----------|----------|
| `domain` field is non-empty | DomainDefinition | Fatal |
| `version` field follows semver | DomainDefinition | Warning |
| `id` field is non-empty | All types with ID | Fatal |
| `id` matches pattern `[a-z][a-z0-9._-]*` | All types with ID | Fatal |
| `capabilities` entries match pattern `[a-z_]+:[a-z_]+:[a-z_]+` | All types with capabilities | Fatal |
| `layout` is one of: list, detail, dashboard, custom | PageDefinition | Fatal |
| `type` is one of valid field types | FieldDefinition | Fatal |
| `type` is one of valid column types | ColumnDefinition | Fatal |
| `type` is one of valid action types | ActionDefinition | Fatal |
| `type` is one of valid step types | StepDefinition | Fatal |
| `operation.type` is "openapi" or "sdk" | OperationBinding | Fatal |
| If `operation.type` is "openapi", `operation_id` is non-empty | OperationBinding | Fatal |
| If `operation.type` is "sdk", `handler` is non-empty | OperationBinding | Fatal |
| `page_size` is between 1 and 200 | TableDefinition | Warning (default applied) |
| `body_mapping` is one of: passthrough, template, projection | InputMapping | Fatal |
| `transitions` reference existing step IDs in `from` and `to` | WorkflowDefinition | Fatal |
| `initial_step` matches a step ID | WorkflowDefinition | Fatal |
| At least one step with type "terminal" exists | WorkflowDefinition | Warning |

### Referential Validation (Against OpenAPI)

Referential validation checks that definitions correctly reference backend operations.

```
For each definition with OperationBinding where type == "openapi":
  1. Verify (service_id, operation_id) exists in the OpenAPIIndex.
     → Fatal if not found: "Operation 'updateOrder' not found in service 'orders-svc'"

  2. If InputMapping has path_params:
     Verify each param name matches a path parameter in the OpenAPI operation.
     → Warning if mismatch: "Path param 'orderId' not found in operation 'updateOrder'"

  3. If InputMapping has query_params:
     Verify each param name matches a query parameter in the OpenAPI operation.
     → Warning if mismatch.

  4. If ResponseMapping has items_path:
     Attempt to resolve the path against the OpenAPI response schema.
     → Warning if path doesn't exist in schema (schema may use dynamic types).

For each definition with OperationBinding where type == "sdk":
  1. Verify handler name is registered in the SDKInvokerRegistry.
     → Fatal if not found: "SDK handler 'ledger.PostEntry' not registered"
```

### Cross-Reference Validation

```
For each ActionDefinition with type == "command":
  Verify command_id exists in the domain's commands.
  → Fatal if not found: "Action 'orders.edit_action' references unknown command 'orders.update'"

For each ActionDefinition with type == "form":
  Verify form_id exists in the domain's forms.
  → Fatal if not found.

For each ActionDefinition with type == "workflow":
  Verify workflow_id exists in the domain's workflows.
  → Fatal if not found.

For each FormDefinition:
  Verify submit_command exists in the domain's commands.
  → Fatal if not found.

  If load_source is defined:
    Verify its operation_id exists in the OpenAPI index.
    → Fatal if not found.

For each FilterDefinition with options.lookup_id:
  Verify lookup_id exists in the domain's lookups or _shared lookups.
  → Fatal if not found.

For each FieldDefinition with lookup.lookup_id:
  Verify lookup_id exists.
  → Fatal if not found.

For each StepDefinition with form_id:
  Verify form_id exists in the domain's forms.
  → Fatal if not found.
```

### Validation Report

At the end of validation, the loader produces a report:

```
Definition Validation Report
=============================
Loaded: 4 domains, 12 pages, 8 forms, 15 commands, 3 workflows, 4 searches
OpenAPI: 4 services, 127 operations indexed
Referenced: 42 operations (33% of available)

FATAL errors: 0
WARNINGS: 2
  - orders/definition.yaml:147 — Response path "data.orders" could not be verified against OpenAPI schema for operation "listOrders"
  - inventory/definition.yaml:89 — Form "inventory.adjustment_form" is not referenced by any page or workflow

Status: PASSED (0 fatal errors)
```

If there are fatal errors, the process exits with code 1 and the report is written
to stderr.

---

## DefinitionRegistry

The DefinitionRegistry is the runtime store for all loaded and validated definitions.
It provides fast, thread-safe lookups by ID.

### Interface

```
DefinitionRegistry
  ├── GetDomain(domainId string) → (DomainDefinition, bool)
  ├── GetPage(pageId string) → (PageDefinition, bool)
  ├── GetForm(formId string) → (FormDefinition, bool)
  ├── GetCommand(commandId string) → (CommandDefinition, bool)
  ├── GetWorkflow(workflowId string) → (WorkflowDefinition, bool)
  ├── GetSearch(searchId string) → (SearchDefinition, bool)
  ├── GetLookup(lookupId string) → (LookupDefinition, bool)
  ├── GetAction(actionId string) → (ActionDefinition, bool)
  ├── AllDomains() → []DomainDefinition
  ├── AllSearches() → []SearchDefinition
  ├── AllLookups() → []LookupDefinition
  ├── Replace(definitions []DomainDefinition)    // atomic swap
  └── Checksum() → string                        // combined checksum
```

### Implementation Strategy

**Recommended: Immutable Snapshot with Atomic Pointer**

```
1. At startup: build a definitionSnapshot struct containing all maps.
2. Store a pointer to the snapshot using atomic.Pointer[definitionSnapshot].
3. All read operations dereference the pointer (no locks needed).
4. For hot-reload: build a new snapshot, atomically swap the pointer.
5. Old snapshot is garbage-collected when no goroutine references it.
```

This provides:
- Lock-free reads (most operations are reads).
- Atomic updates (no partial state visible during reload).
- No read/write contention.

### ID Uniqueness

IDs must be unique across all domains. If two domains define a page with the
same ID, the loader rejects both definitions at startup.

To avoid collisions, IDs are namespace-prefixed by convention:
- `orders.list` — page in the orders domain
- `inventory.list` — page in the inventory domain

The registry enforces uniqueness but does not enforce naming conventions.

---

## Hot-Reload

### When to Use Hot-Reload

- **Development:** Rapid iteration on definitions without restarting the BFF.
- **Staged rollout:** Update definitions in production without redeployment
  (only if definitions are loaded from a shared filesystem or artifact store).

### When NOT to Use Hot-Reload

- **If definitions are baked into the container image:** Hot-reload is unnecessary;
  just deploy a new image.
- **If the risk of serving invalid definitions is unacceptable:** Startup validation
  is the strongest guarantee. Hot-reload validation happens at runtime and a
  transient error could leave the system in an inconsistent state.

### Hot-Reload Flow

```
1. File watcher detects changes in definitions/ directory.
2. Debounce: wait 2 seconds for additional changes (batch multiple file saves).
3. Load all definitions from scratch (full reload, not incremental).
4. Validate new definitions against OpenAPI index.
5. If validation PASSES:
   a. Build new definition snapshot.
   b. Atomically swap the registry's snapshot pointer.
   c. Log: "Definitions reloaded: {old_checksum} → {new_checksum}"
   d. Emit metric: definition.reload.success.
6. If validation FAILS:
   a. Log: "Definition reload failed: {errors}"
   b. Keep the current definitions (no swap).
   c. Emit metric: definition.reload.failure.
   d. Optionally: send alert.
```

### Strict Mode

In strict mode (recommended for production), hot-reload additionally:

1. Verifies the new definitions against a signed manifest file.
2. Rejects changes that don't match the manifest signature.
3. Logs any unsigned reload attempts as security events.

---

## Definition Authoring Guide

### Step 1: Identify the Backend Operations

Start by listing the OpenAPI operations your domain exposes:

```
listOrders        GET    /api/v1/orders
getOrder          GET    /api/v1/orders/{orderId}
createOrder       POST   /api/v1/orders
updateOrder       PATCH  /api/v1/orders/{orderId}
cancelOrder       POST   /api/v1/orders/{orderId}/cancel
confirmOrder      POST   /api/v1/orders/{orderId}/confirm
searchOrders      GET    /api/v1/orders/search
getOrderStatuses  GET    /api/v1/orders/statuses
```

### Step 2: Decide Which Operations to Expose

Not all operations need UI exposure. Consider:
- Is this operation used by the UI? (Internal endpoints: no)
- Does the user need to see this? (Background sync endpoints: no)
- Is the frontend ready for this? (Future features: not yet)

### Step 3: Define Capabilities

For each exposed operation, define the capability that gates it:

```
orders:list:view        → needed to see the orders list page
orders:detail:view      → needed to see order details
orders:edit:execute     → needed to update orders
orders:cancel:execute   → needed to cancel orders
orders:approve:execute  → needed to approve orders
```

### Step 4: Write the Definition

Start with the top-level structure:

```yaml
domain: "orders"
version: "1.0.0"

navigation:
  label: "Orders"
  icon: "shopping_cart"
  order: 10
  capabilities: ["orders:nav:view"]
  children:
    - label: "All Orders"
      route: "/orders"
      page_id: "orders.list"
      capabilities: ["orders:list:view"]
      order: 1

pages: [...]
forms: [...]
commands: [...]
workflows: [...]
searches: [...]
lookups: [...]
```

Then fill in each section. See [06 — Definition Schema Reference](06-definition-schema-reference.md)
for the complete schema of each type.

### Step 5: Test Locally

1. Run the BFF with your definition file.
2. Verify it starts without validation errors.
3. Call `GET /ui/navigation` and verify your domain appears.
4. Call `GET /ui/pages/{pageId}` and verify the descriptor is correct.
5. Call `GET /ui/pages/{pageId}/data` and verify data flows through.

### Step 6: Submit for Review

Submit the definition file through the standard code review process. The BFF team
reviews for structural correctness; the domain team reviews for business correctness.

---

## Versioning Strategy

### Definition Version Field

Each definition has a `version` field following semantic versioning:

```yaml
domain: "orders"
version: "2.1.0"
```

- **Major version bump:** Breaking changes (renamed IDs, removed pages/commands).
- **Minor version bump:** New pages, forms, commands, or workflows.
- **Patch version bump:** Bug fixes, label changes, field mapping updates.

The version field is informational — the BFF does not enforce compatibility rules
based on versions. But it provides traceability: logs and metrics include the
definition version, making it easy to correlate behavior changes with definition
updates.

### Change Management

| Change Type | Risk | Requires Frontend Change? | Requires Backend Change? |
|-------------|------|---------------------------|--------------------------|
| Add new page | Low | No (frontend ignores unknown pages) | No |
| Add new column to table | Low | No (frontend renders unknown columns) | No |
| Add new action | Low | No | No |
| Add new command | Low | No | No (but backend operation must exist) |
| Change field_map | None | No | Absorbs backend change |
| Change operation_id | None | No | New operation must exist |
| Change page title | None | No | No |
| Remove a column | Low | No (frontend adapts) | No |
| Add new capability requirement | Medium | No | Policy engine must be updated |
| Rename page ID | High | Yes (routes change) | No |
| Rename command ID | High | Yes (action references change) | No |
| Remove a page | High | Yes | No |
| Remove a command | High | Yes (if frontend references it) | No |

---

## Anti-Patterns

### Anti-Pattern 1: Auto-Generating Definitions from OpenAPI

**Wrong:**
```bash
# DO NOT: auto-generate definitions from specs
./generate-definitions --from specs/orders-svc.yaml --output definitions/orders/
```

**Why it's wrong:** Auto-generation violates Principle P1 (No Implicit Exposure).
Every exposed operation must be a deliberate choice by the domain team. Auto-generation
would expose internal endpoints, admin endpoints, and deprecated endpoints.

### Anti-Pattern 2: Embedding Backend Logic in Definitions

**Wrong:**
```yaml
commands:
  - id: "orders.update"
    # DO NOT: embed business validation rules
    validation_rules:
      - if: "input.total > 10000"
        then: "requires_approval"
    # Business rules belong in the backend service
```

**Why it's wrong:** Definitions describe UI presentation, not business logic. Validation
rules, business constraints, and domain logic belong in backend services.

### Anti-Pattern 3: Using Definitions as Documentation

**Wrong:** Treating definition files as the primary source of API documentation.

**Why it's wrong:** Definitions describe the UI surface, which is a subset of the backend
API. The OpenAPI spec is the API documentation. The definition is the UI mapping layer.

### Anti-Pattern 4: One Definition for Everything

**Wrong:** A single global definition file containing all domains.

**Why it's wrong:** Violates domain ownership. Different teams need to edit their
definitions independently without merge conflicts on a shared file.
