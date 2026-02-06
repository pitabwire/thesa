# Thesa — Backend-For-Frontend Architecture

## Table of Contents

1. [Architectural Overview](#1-architectural-overview)
2. [Core Principles and Invariants](#2-core-principles-and-invariants)
3. [System Topology](#3-system-topology)
4. [Transport and Invocation Model](#4-transport-and-invocation-model)
5. [Request Context and Identity](#5-request-context-and-identity)
6. [Explicit UI Exposure Definitions](#6-explicit-ui-exposure-definitions)
7. [Definition Schema Reference](#7-definition-schema-reference)
8. [Capability and Permission System](#8-capability-and-permission-system)
9. [UI Descriptor Model](#9-ui-descriptor-model)
10. [Server-Driven UI Metadata APIs](#10-server-driven-ui-metadata-apis)
11. [Command and Action Model](#11-command-and-action-model)
12. [Workflow Engine](#12-workflow-engine)
13. [Global Search](#13-global-search)
14. [API Mapping and Invocation Engine](#14-api-mapping-and-invocation-engine)
15. [Schema and Contract Stability](#15-schema-and-contract-stability)
16. [Error Handling and Validation](#16-error-handling-and-validation)
17. [Observability and Reliability](#17-observability-and-reliability)
18. [Security Model](#18-security-model)
19. [Core Abstractions and Interfaces](#19-core-abstractions-and-interfaces)
20. [Go Package Structure](#20-go-package-structure)
21. [Example Domain: Orders](#21-example-domain-orders)
22. [Example End-to-End Flow](#22-example-end-to-end-flow)
23. [Deployment and Operational Model](#23-deployment-and-operational-model)

---

## 1. Architectural Overview

Thesa is a metadata-driven Backend-For-Frontend (BFF) that sits between a Flutter/React
UI and an ecosystem of backend domain services. It is the **sole backend entry point**
for the frontend. The frontend never communicates with domain services directly.

### What Thesa Does

Thesa dynamically assembles UI metadata (menus, pages, tables, forms, actions, workflows)
by combining three inputs:

1. **UI Exposure Definitions** — declarative YAML files authored by domain teams that
   describe what backend operations are available and how they map to UI constructs.
2. **Capabilities** — the resolved set of permissions for the current user, tenant, and
   partition, evaluated against a policy engine.
3. **OpenAPI Specifications** — the machine-readable contracts of backend services, used
   to validate definitions and dynamically invoke backend operations at runtime.

The frontend receives **descriptors** — fully resolved, capability-filtered, UI-oriented
data structures that describe what to render and what actions are available. The frontend
renders these descriptors using a metadata-driven rendering engine.

### What Thesa Does NOT Do

- It does not embed domain business logic.
- It does not expose backend schemas directly to the frontend.
- It does not auto-discover or auto-expose backend APIs.
- It does not allow the frontend to construct arbitrary API calls.
- It does not own data storage (except workflow state and session caches).

### High-Level Data Flow

```
┌─────────────┐
│  Flutter UI  │
└──────┬───────┘
       │  HTTPS / JSON
       ▼
┌──────────────────────────────────────────────────────────────────┐
│                         T H E S A  (BFF)                         │
│                                                                  │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌───────────┐  │
│  │ Transport  │→ │  Context   │→ │ Capability │→ │ Providers │  │
│  │ (HTTP)     │  │ Extraction │  │ Resolution │  │ & Engine  │  │
│  └────────────┘  └────────────┘  └────────────┘  └─────┬─────┘  │
│                                                        │        │
│  ┌────────────────────────────────────────────────────┐│        │
│  │              Definition Registry                   ││        │
│  │  (loaded from YAML, validated against OpenAPI)     ││        │
│  └────────────────────────────────────────────────────┘│        │
│                                                        │        │
│  ┌────────────────────────────────────────────────────┐│        │
│  │            Invocation Layer                        │◄        │
│  │  ┌──────────────────┐  ┌────────────────────┐     │        │
│  │  │ OpenAPI Invoker  │  │   SDK Invoker      │     │        │
│  │  │ (dynamic HTTP)   │  │ (typed clients)    │     │        │
│  │  └────────┬─────────┘  └────────┬───────────┘     │        │
│  └───────────┼─────────────────────┼─────────────────┘        │
└──────────────┼─────────────────────┼──────────────────────────┘
               │                     │
               ▼                     ▼
        ┌──────────┐          ┌──────────┐
        │ Service A│          │ Service B│
        │ (OpenAPI)│          │ (gRPC)   │
        └──────────┘          └──────────┘
```

---

## 2. Core Principles and Invariants

These are non-negotiable architectural rules. Violations are treated as bugs.

### P1: No implicit exposure
Every backend operation visible to the UI must appear in an explicit UI Exposure
Definition. Loading a new OpenAPI spec does NOT make its operations available.
The definition is the allowlist.

### P2: Frontend receives descriptors, not definitions
Definitions are internal configuration. The frontend receives descriptors — resolved,
filtered, safe projections of definitions. Descriptors never contain backend URLs,
operation IDs, or internal field names.

### P3: Tenant isolation is structural
Tenant ID is extracted from the verified token and injected into every downstream call.
No code path allows a request to operate on a tenant other than the authenticated one.
Partition (workspace) isolation follows the same pattern.

### P4: Commands are the only mutation path
The frontend cannot invoke backend APIs directly. All mutations go through
`POST /ui/commands/{commandId}`, which validates, authorizes, maps, invokes, and
translates the operation.

### P5: Capabilities gate everything
Every navigation node, page, table column, form field, action, and workflow step
declares required capabilities. The BFF filters all descriptors against the resolved
capability set before serving them to the frontend.

### P6: Definitions are validated at startup
Every definition is validated against the loaded OpenAPI specifications at process
startup. If a definition references a non-existent `operationId`, the process fails
to start. This provides a compile-time-like guarantee for a runtime-configured system.

### P7: Backend evolution does not break the frontend
The BFF contains an adapter layer that maps between backend response shapes and
UI descriptor contracts. Backend teams can evolve their APIs; the definition's
response mapping absorbs the change. The frontend contract remains stable.

### P8: Workflows are persistent and resumable
Workflow instances are persisted. A server restart, user session change, or timeout
does not lose workflow state. Every workflow step is idempotent and authorized
independently.

### P9: Definition files are integrity-checked
Definitions are checksummed (SHA-256) at load time. Any tampering between load and
runtime is detectable. In production, definitions should be loaded from a read-only
filesystem or a verified artifact store.

### P10: No recompilation for new APIs
Adding a new backend service, new pages, new commands, or new workflows requires only
adding/modifying YAML definition files and (for OpenAPI services) providing the spec.
The BFF binary does not change.

---

## 3. System Topology

### Multi-Service Landscape

```
                  ┌──────────────────┐
                  │   Identity       │
                  │   Provider       │
                  │   (OIDC / JWT)   │
                  └────────┬─────────┘
                           │ token verification
                           ▼
┌───────────┐    ┌─────────────────────┐    ┌───────────────────┐
│ Flutter   │───▶│      THESA BFF      │───▶│  Policy Engine    │
│ UI        │◀───│                     │◀───│  (OPA / Cedar /   │
└───────────┘    │  ┌───────────────┐  │    │   custom)         │
                 │  │  Definition   │  │    └───────────────────┘
                 │  │  Registry     │  │
                 │  └───────────────┘  │    ┌───────────────────┐
                 │  ┌───────────────┐  │───▶│  Orders Service   │
                 │  │  Workflow     │  │    │  (OpenAPI)        │
                 │  │  Store        │  │    └───────────────────┘
                 │  └───────────────┘  │
                 │  ┌───────────────┐  │    ┌───────────────────┐
                 │  │  OpenAPI      │  │───▶│  Inventory Svc    │
                 │  │  Index        │  │    │  (OpenAPI)        │
                 │  └───────────────┘  │    └───────────────────┘
                 │  ┌───────────────┐  │
                 │  │  SDK Client   │  │    ┌───────────────────┐
                 │  │  Registry     │  │───▶│  Ledger Service   │
                 │  └───────────────┘  │    │  (Connect RPC)    │
                 └─────────────────────┘    └───────────────────┘
```

### Tenancy Model

```
Organization (Tenant)
 └── Partition (Workspace / Environment)
      └── User (Subject)
           └── Session (Device + Token)
```

Every request is scoped to exactly one `(tenant, partition, subject)` triple.
The BFF never holds cross-tenant state in memory.

---

## 4. Transport and Invocation Model

### 4.1 Inbound: BFF HTTP API (Frontend-Facing)

The BFF exposes a small, fixed set of endpoints to the frontend. These endpoints
do NOT change when new domains or backend services are added.

| Method | Path                                  | Purpose                           |
|--------|---------------------------------------|-----------------------------------|
| GET    | `/ui/navigation`                      | Menu tree for current user        |
| GET    | `/ui/pages/{pageId}`                  | Page descriptor                   |
| GET    | `/ui/pages/{pageId}/data`             | Data for a page's table/sections  |
| GET    | `/ui/forms/{formId}`                  | Form descriptor                   |
| GET    | `/ui/forms/{formId}/data`             | Pre-populated form data           |
| POST   | `/ui/commands/{commandId}`            | Execute a command                 |
| GET    | `/ui/workflows/{instanceId}`          | Workflow instance state           |
| POST   | `/ui/workflows/{workflowId}/start`    | Start a new workflow              |
| POST   | `/ui/workflows/{instanceId}/advance`  | Advance a workflow step           |
| GET    | `/ui/search`                          | Global search                     |
| GET    | `/ui/lookups/{lookupId}`              | Reference data lookup             |
| GET    | `/ui/health`                          | Health check (no auth)            |

All endpoints except `/ui/health` require authentication. All responses use
a consistent JSON envelope.

### 4.2 Outbound: OpenAPI-Driven Dynamic Invocation (Primary Path)

This is the primary mechanism for calling backend services.

**Startup sequence:**

1. Load all OpenAPI specification files from configured directories.
2. Parse each spec and extract all operations.
3. Build an **Operation Index**: a map from `(serviceId, operationId)` to the
   fully resolved operation metadata (method, path template, parameters, request
   body schema, response schemas).
4. Load all UI Exposure Definitions.
5. **Validate** every definition's `operation_id` references against the Operation
   Index. Fail startup if any reference is invalid.

**Runtime invocation sequence:**

1. Receive a command or data request.
2. Look up the definition to find the `OperationBinding`.
3. Look up the operation in the Operation Index by `(service_id, operation_id)`.
4. Resolve the service's base URL from configuration.
5. Apply `InputMapping` to construct path parameters, query parameters, headers,
   and request body from the request context and user input.
6. Validate the constructed request body against the operation's OpenAPI request
   schema.
7. Execute the HTTP request with circuit breaker, timeout, and retry policies.
8. Validate the response status code against expected codes in the OpenAPI spec.
9. Apply `OutputMapping` / `ResponseMapping` to transform the response into
   the UI-safe shape.
10. Return the result to the frontend.

**The OpenAPI invoker never generates code.** It builds HTTP requests dynamically
at runtime from the indexed operation metadata. This is what allows adding new
APIs without recompilation.

### 4.3 Outbound: SDK / Typed Client Invocation (Secondary Path)

For use cases where dynamic HTTP invocation is insufficient:

- **Streaming APIs** (server-sent events, bidirectional streams via Connect RPC)
- **Complex multi-call orchestration** (e.g., ledger double-entry that requires
  transactional semantics across calls)
- **High-integrity paths** where compile-time type safety is required (settlement,
  reconciliation)

SDK invokers are Go implementations that satisfy the same `OperationInvoker`
interface as the OpenAPI invoker. They are registered in the **Invoker Registry**
at startup with a handler name.

Definitions reference them via:

```yaml
operation:
  type: sdk
  handler: "ledger.PostEntry"
  service_id: "ledger"
```

The BFF binary must be recompiled to add new SDK invokers (since they contain
compiled Go code), but this is intentional — these are high-integrity paths
where compile-time guarantees are valued.

### 4.4 Unified Invocation Abstraction

Both invocation paths satisfy the same interface:

```
OperationInvoker
  ├── Invoke(context, requestContext, binding, input) → (result, error)
  └── Supports(binding) → bool
```

The **Invoker Registry** holds all registered invokers and dispatches based on
the `OperationBinding.Type` field:

```
InvokerRegistry
  ├── Register(invoker)
  ├── Invoke(context, requestContext, binding, input) → (result, error)
  └── (iterates invokers, finds one where Supports() returns true)
```

---

## 5. Request Context and Identity

### 5.1 RequestContext Structure

Every authenticated request produces a `RequestContext` that flows through the
entire processing pipeline:

```
RequestContext
  ├── SubjectID       string       // authenticated user ID (from JWT sub)
  ├── Email           string       // user email (from JWT)
  ├── TenantID        string       // tenant ID (from JWT custom claim)
  ├── PartitionID     string       // workspace (from request header X-Partition-Id)
  ├── Roles           []string     // roles (from JWT)
  ├── Claims          map[string]any  // all JWT claims
  ├── SessionID       string       // session identifier
  ├── DeviceID        string       // device fingerprint (from header)
  ├── CorrelationID   string       // request correlation ID (generated or from header)
  ├── TraceID         string       // distributed trace ID
  ├── SpanID          string       // current span ID
  ├── Locale          string       // Accept-Language header
  └── Timezone        string       // X-Timezone header
```

### 5.2 Context Construction Pipeline

```
Incoming HTTP Request
  │
  ├─ 1. TLS termination (at load balancer)
  │
  ├─ 2. Rate limiting middleware (by IP / token fingerprint)
  │
  ├─ 3. Authentication middleware:
  │     a. Extract Bearer token from Authorization header
  │     b. Verify token signature (JWKS from identity provider)
  │     c. Validate token claims (expiry, audience, issuer)
  │     d. Extract SubjectID, TenantID, Roles, Claims
  │     e. Reject if token invalid → 401
  │
  ├─ 4. Context construction middleware:
  │     a. Extract PartitionID from X-Partition-Id header
  │     b. Validate PartitionID belongs to TenantID (via partition registry or claim)
  │     c. Generate CorrelationID (or accept from X-Correlation-Id)
  │     d. Extract TraceID/SpanID from traceparent header
  │     e. Extract Locale from Accept-Language
  │     f. Extract Timezone from X-Timezone
  │     g. Extract DeviceID from X-Device-Id
  │     h. Build RequestContext, attach to Go context
  │
  ├─ 5. Capability resolution middleware (optional — can be lazy):
  │     a. Call CapabilityResolver with RequestContext
  │     b. Attach CapabilitySet to context
  │
  └─ 6. Handler receives context with RequestContext and CapabilitySet
```

### 5.3 Context Propagation to Backend Services

When the BFF invokes a backend service, the `RequestContext` is propagated via
HTTP headers:

| Header              | Source                     | Purpose                        |
|---------------------|----------------------------|--------------------------------|
| Authorization       | Original token (forwarded) | Backend re-validates identity  |
| X-Tenant-Id         | RequestContext.TenantID    | Explicit tenant scoping        |
| X-Partition-Id      | RequestContext.PartitionID | Workspace scoping              |
| X-Correlation-Id    | RequestContext.CorrelationID | Request tracing               |
| traceparent         | OpenTelemetry span         | Distributed tracing            |
| X-Request-Subject   | RequestContext.SubjectID   | Audit trail                    |

Backend services MUST validate the tenant ID against their own authorization.
The BFF's propagation is a convenience, not a substitute for backend-side checks.

---

## 6. Explicit UI Exposure Definitions

### 6.1 Definition File Structure

Definitions are organized by domain. Each domain team owns their definition files.

```
definitions/
  ├── orders/
  │   └── definition.yaml
  ├── inventory/
  │   └── definition.yaml
  ├── customers/
  │   └── definition.yaml
  └── ledger/
      └── definition.yaml
```

Each file is a complete `DomainDefinition` containing all pages, forms, commands,
workflows, searches, and lookups for that domain.

### 6.2 Definition Loading Pipeline

```
Startup:
  1. Scan definition directories
  2. For each YAML file:
     a. Read file contents
     b. Compute SHA-256 checksum
     c. Parse YAML into DomainDefinition
     d. Validate structural integrity (required fields, valid references)
     e. Validate against OpenAPI index:
        - Every operation_id must exist
        - Input mappings must reference valid parameters
        - Response mappings must reference valid response fields
     f. Validate capability references (well-formed strings)
     g. Validate workflow transitions (all step IDs exist, no orphan steps)
     h. Register in DefinitionRegistry

  If any validation fails → log error, refuse to start.

Runtime (optional hot-reload):
  1. Watch definition directories for changes
  2. On change: load new definition into shadow registry
  3. Validate shadow registry
  4. Atomic swap: replace active registry with shadow
  5. Log reload event with checksum diff
```

### 6.3 DefinitionLoader Interface

```
DefinitionLoader
  ├── LoadAll(directories []string) → ([]DomainDefinition, error)
  ├── LoadFile(path string) → (DomainDefinition, error)
  └── Watch(directories []string, onChange callback) → (stop func, error)
```

### 6.4 DefinitionRegistry Interface

The registry is a read-optimized, concurrency-safe store of all loaded definitions.

```
DefinitionRegistry
  ├── GetDomain(domainId string) → (DomainDefinition, bool)
  ├── GetPage(pageId string) → (PageDefinition, bool)
  ├── GetForm(formId string) → (FormDefinition, bool)
  ├── GetCommand(commandId string) → (CommandDefinition, bool)
  ├── GetWorkflow(workflowId string) → (WorkflowDefinition, bool)
  ├── GetSearch(searchId string) → (SearchDefinition, bool)
  ├── GetLookup(lookupId string) → (LookupDefinition, bool)
  ├── AllDomains() → []DomainDefinition
  ├── AllSearches() → []SearchDefinition
  ├── Replace(definitions []DomainDefinition)    // atomic swap for hot-reload
  └── Checksum() → string                        // combined checksum of all definitions
```

Implementation: use `sync.RWMutex`-protected maps for thread safety, or an
`atomic.Pointer` to an immutable snapshot for lock-free reads.

### 6.5 Definition Validation Rules

| Rule                             | Severity | Check                                          |
|----------------------------------|----------|-------------------------------------------------|
| Unique IDs within domain         | Fatal    | No two pages, forms, commands etc. share an ID  |
| Globally unique IDs across domains | Fatal  | Page "orders.list" cannot collide with "inventory.list" unless namespaced |
| Valid operation_id references    | Fatal    | Every operation_id exists in the OpenAPI index  |
| Valid step references in workflows | Fatal  | Every transition references existing step IDs   |
| Initial step exists              | Fatal    | Workflow's initial_step matches a step ID       |
| Terminal states reachable        | Warning  | Every workflow has at least one terminal step    |
| Capability format                | Fatal    | Capabilities match pattern `[a-z]+:[a-z_]+:[a-z_]+` |
| No orphan forms                  | Warning  | Every form is referenced by a page or workflow   |
| Response mapping paths valid     | Warning  | JSON paths match expected response structure     |

---

## 7. Definition Schema Reference

### 7.1 DomainDefinition (Top Level)

```yaml
domain: "orders"              # unique domain identifier
version: "2.1.0"              # semver for change tracking

navigation:                    # how this domain appears in the main menu
  label: "Orders"
  icon: "shopping_cart"
  order: 10
  capabilities: ["orders:nav:view"]
  children:
    - label: "All Orders"
      icon: "list"
      route: "/orders"
      page_id: "orders.list"
      capabilities: ["orders:list:view"]
      order: 1
    - label: "Create Order"
      route: "/orders/new"
      page_id: "orders.create"
      capabilities: ["orders:create:view"]
      order: 2

pages: [...]
forms: [...]
commands: [...]
workflows: [...]
searches: [...]
lookups: [...]
```

### 7.2 PageDefinition

```yaml
pages:
  - id: "orders.list"
    title: "Orders"
    route: "/orders"
    layout: "list"                # list | detail | dashboard | custom
    capabilities: ["orders:list:view"]
    refresh_interval: 30          # optional auto-refresh in seconds

    table:
      data_source:
        operation_id: "listOrders"
        service_id: "orders-svc"
        mapping:
          items_path: "data.orders"
          total_path: "data.total"
          field_map:
            order_number: "orderNumber"   # backend → UI field renaming
            created_at: "createdAt"

      columns:
        - field: "order_number"
          label: "Order #"
          type: "link"
          sortable: true
          link:
            route: "/orders/{id}"
            params:
              id: "id"

        - field: "status"
          type: "status"
          label: "Status"
          sortable: true
          status_map:
            pending: "warning"
            confirmed: "info"
            shipped: "success"
            cancelled: "danger"

        - field: "total_amount"
          label: "Total"
          type: "currency"
          format: "USD"
          sortable: true

        - field: "created_at"
          label: "Created"
          type: "datetime"
          sortable: true

      filters:
        - field: "status"
          label: "Status"
          type: "select"
          operator: "eq"
          options:
            static:
              - { label: "Pending", value: "pending" }
              - { label: "Confirmed", value: "confirmed" }
              - { label: "Shipped", value: "shipped" }
              - { label: "Cancelled", value: "cancelled" }

        - field: "date_range"
          label: "Date Range"
          type: "date-range"
          operator: "between"

      row_actions:
        - id: "orders.view"
          label: "View"
          icon: "visibility"
          type: "navigate"
          navigate_to: "/orders/{id}"
          capabilities: ["orders:detail:view"]

      default_sort: "created_at"
      sort_dir: "desc"
      page_size: 25
      selectable: true

      bulk_actions:
        - id: "orders.bulk_export"
          label: "Export Selected"
          icon: "download"
          type: "command"
          command_id: "orders.export"
          capabilities: ["orders:export:execute"]

    actions:                       # page-level actions (toolbar)
      - id: "orders.create_action"
        label: "New Order"
        icon: "add"
        style: "primary"
        type: "navigate"
        navigate_to: "/orders/new"
        capabilities: ["orders:create:view"]
```

### 7.3 Detail Page with Sections

```yaml
  - id: "orders.detail"
    title: "Order Details"
    route: "/orders/{id}"
    layout: "detail"
    capabilities: ["orders:detail:view"]

    breadcrumb:
      - label: "Orders"
        route: "/orders"
      - label: "{order_number}"   # resolved from page data

    sections:
      - id: "header"
        title: "Order Information"
        layout: "grid"
        columns: 3
        fields:
          - field: "order_number"
            label: "Order Number"
            type: "text"
            read_only: "true"
          - field: "status"
            label: "Status"
            type: "status"
            read_only: "true"
          - field: "customer_name"
            label: "Customer"
            type: "reference"
            read_only: "true"
          - field: "total_amount"
            label: "Total Amount"
            type: "currency"
            format: "USD"
            read_only: "true"
          - field: "created_at"
            label: "Created"
            type: "datetime"
            read_only: "true"

      - id: "line_items"
        title: "Line Items"
        layout: "card"
        capabilities: ["orders:line_items:view"]
        # (would contain a nested table or repeated section)

      - id: "notes"
        title: "Internal Notes"
        layout: "card"
        capabilities: ["orders:notes:view"]
        collapsible: true
        collapsed: true
        fields:
          - field: "internal_notes"
            label: "Notes"
            type: "rich-text"
            read_only: "orders:notes:edit"   # capability-driven: read_only unless user has edit cap
            visibility: "orders:notes:view"

    actions:
      - id: "orders.edit_action"
        label: "Edit"
        icon: "edit"
        type: "form"
        form_id: "orders.edit_form"
        capabilities: ["orders:edit:execute"]
        conditions:
          - field: "status"
            operator: "in"
            value: ["pending", "confirmed"]
            effect: "show"

      - id: "orders.cancel_action"
        label: "Cancel Order"
        icon: "cancel"
        style: "danger"
        type: "workflow"
        workflow_id: "orders.cancellation"
        capabilities: ["orders:cancel:execute"]
        confirmation:
          title: "Cancel Order?"
          message: "This will cancel order {order_number}. This action cannot be undone."
          confirm: "Yes, Cancel"
          style: "danger"
        conditions:
          - field: "status"
            operator: "in"
            value: ["pending", "confirmed"]
            effect: "show"

      - id: "orders.approve_action"
        label: "Approve"
        icon: "check"
        style: "primary"
        type: "workflow"
        workflow_id: "orders.approval"
        capabilities: ["orders:approve:execute"]
        conditions:
          - field: "status"
            operator: "eq"
            value: "pending"
            effect: "show"
```

### 7.4 FormDefinition

```yaml
forms:
  - id: "orders.edit_form"
    title: "Edit Order"
    capabilities: ["orders:edit:execute"]
    submit_command: "orders.update"
    load_source:
      operation_id: "getOrder"
      service_id: "orders-svc"
      mapping:
        items_path: "data"
    success_route: "/orders/{id}"
    success_message: "Order updated successfully"

    sections:
      - id: "basics"
        title: "Order Details"
        layout: "grid"
        columns: 2
        fields:
          - field: "customer_id"
            label: "Customer"
            type: "reference"
            required: true
            lookup:
              lookup_id: "customers.search"
            span: 2

          - field: "shipping_address"
            label: "Shipping Address"
            type: "textarea"
            required: true
            validation:
              max_length: 500

          - field: "notes"
            label: "Notes"
            type: "textarea"
            required: false
            visibility: "orders:notes:edit"

          - field: "priority"
            label: "Priority"
            type: "select"
            lookup:
              static:
                - { label: "Normal", value: "normal" }
                - { label: "High", value: "high" }
                - { label: "Urgent", value: "urgent" }
```

### 7.5 CommandDefinition

```yaml
commands:
  - id: "orders.update"
    capabilities: ["orders:edit:execute"]
    operation:
      type: "openapi"
      operation_id: "updateOrder"
      service_id: "orders-svc"
    input:
      path_params:
        orderId: "route.id"             # from the current route parameter
      body_mapping: "projection"
      field_projection:
        customerId: "input.customer_id"
        shippingAddress: "input.shipping_address"
        notes: "input.notes"
        priority: "input.priority"
    output:
      type: "project"
      fields:
        id: "data.id"
        order_number: "data.orderNumber"
      success_message: "Order updated successfully"
      error_map:
        ORDER_NOT_FOUND: "This order no longer exists"
        INVALID_STATUS: "This order cannot be edited in its current status"
    idempotency:
      key_source: "header:Idempotency-Key"
      ttl: "24h"

  - id: "orders.cancel"
    capabilities: ["orders:cancel:execute"]
    operation:
      type: "openapi"
      operation_id: "cancelOrder"
      service_id: "orders-svc"
    input:
      path_params:
        orderId: "input.order_id"
      body_mapping: "template"
      body_template:
        reason: "input.reason"
        cancelledBy: "context.subject_id"
    output:
      type: "envelope"
      success_message: "Order cancelled"
```

### 7.6 WorkflowDefinition

```yaml
workflows:
  - id: "orders.approval"
    name: "Order Approval"
    capabilities: ["orders:approve:execute"]
    initial_step: "review"
    timeout: "72h"
    on_timeout: "expired"

    steps:
      - id: "review"
        name: "Review Order"
        type: "approval"
        capabilities: ["orders:approve:execute"]
        form_id: "orders.approval_form"
        assignee:
          type: "role"
          value: "order_approver"

      - id: "process"
        name: "Process Approved Order"
        type: "system"
        operation:
          type: "openapi"
          operation_id: "confirmOrder"
          service_id: "orders-svc"
        input:
          path_params:
            orderId: "workflow.order_id"
          body_mapping: "template"
          body_template:
            approvedBy: "workflow.approved_by"
            approvalNotes: "workflow.approval_notes"

      - id: "notify"
        name: "Send Notification"
        type: "system"
        operation:
          type: "sdk"
          handler: "notifications.SendOrderApproved"
        input:
          body_mapping: "template"
          body_template:
            orderId: "workflow.order_id"
            customerEmail: "workflow.customer_email"

      - id: "approved"
        name: "Approved"
        type: "terminal"

      - id: "rejected"
        name: "Rejected"
        type: "terminal"

      - id: "expired"
        name: "Expired"
        type: "terminal"

    transitions:
      - from: "review"
        to: "process"
        event: "approved"

      - from: "review"
        to: "rejected"
        event: "rejected"

      - from: "review"
        to: "expired"
        event: "timeout"

      - from: "process"
        to: "notify"
        event: "completed"

      - from: "process"
        to: "rejected"
        event: "error"

      - from: "notify"
        to: "approved"
        event: "completed"

      - from: "notify"
        to: "approved"
        event: "error"          # notification failure is non-blocking
```

### 7.7 SearchDefinition

```yaml
searches:
  - id: "orders.search"
    domain: "orders"
    capabilities: ["orders:search:execute"]
    operation:
      type: "openapi"
      operation_id: "searchOrders"
      service_id: "orders-svc"
    result_mapping:
      items_path: "data.results"
      title_field: "orderNumber"
      subtitle_field: "customerName"
      category_field: "status"
      route: "/orders/{id}"
      id_field: "id"
    weight: 10
    max_results: 5
```

### 7.8 LookupDefinition

```yaml
lookups:
  - id: "orders.statuses"
    operation:
      type: "openapi"
      operation_id: "getOrderStatuses"
      service_id: "orders-svc"
    label_field: "label"
    value_field: "code"
    cache:
      ttl: "5m"
      scope: "global"

  - id: "customers.search"
    operation:
      type: "openapi"
      operation_id: "searchCustomers"
      service_id: "customers-svc"
    label_field: "name"
    value_field: "id"
    search_field: "query"
```

---

## 8. Capability and Permission System

### 8.1 Capability Model

A **capability** is a UI-level permission string with the format:

```
{namespace}:{resource}:{action}
```

Examples:
- `orders:list:view` — can see the orders list page
- `orders:detail:edit` — can edit order details
- `orders:approve:execute` — can execute the approval workflow
- `orders:notes:view` — can see the notes section
- `orders:*` — wildcard: all permissions in the orders namespace

Capabilities are **not** backend permissions. They are a UI-specific permission
layer. The mapping between UI capabilities and backend permissions is defined
in the policy engine.

### 8.2 Capability Resolution Flow

```
RequestContext
     │
     ▼
CapabilityResolver
     │
     ├── 1. Check local cache (keyed by subject_id + tenant_id + partition_id)
     │
     ├── 2. If miss: call PolicyEvaluator
     │       │
     │       ├── Fetch role → capability mappings (from policy store)
     │       ├── Fetch tenant-specific overrides
     │       ├── Fetch partition-specific overrides
     │       ├── Evaluate attribute-based rules
     │       └── Return merged CapabilitySet
     │
     ├── 3. Cache result (TTL: configurable, default 60s)
     │
     └── 4. Return CapabilitySet
```

### 8.3 CapabilityResolver Interface

```
CapabilityResolver
  ├── Resolve(RequestContext) → (CapabilitySet, error)
  └── Invalidate(subjectId, tenantId string)          // cache invalidation
```

### 8.4 PolicyEvaluator Interface

```
PolicyEvaluator
  ├── Evaluate(RequestContext, capability string, resource map[string]any) → (bool, error)
  ├── EvaluateAll(RequestContext, capabilities []string, resource map[string]any) → (map[string]bool, error)
  └── Sync() → error                                   // refresh policy data
```

### 8.5 CapabilitySet Type

```
CapabilitySet = map[string]bool

Methods:
  Has(capability string) → bool             // exact match
  HasAll(capabilities []string) → bool      // all must match
  HasAny(capabilities []string) → bool      // at least one must match
  Matches(capability string) → bool         // includes wildcard matching
  Merge(other CapabilitySet)                // union
```

### 8.6 Where Capabilities Are Evaluated

| UI Construct         | When Evaluated          | Effect if Missing                    |
|----------------------|------------------------|--------------------------------------|
| Navigation node      | Menu generation         | Node hidden from menu                |
| Page                 | Page descriptor request | 403 Forbidden                        |
| Table column         | Page resolution         | Column omitted from descriptor       |
| Section              | Page/form resolution    | Section omitted                      |
| Field                | Form resolution         | Field omitted or set to read-only    |
| Action               | Action resolution       | Action omitted from descriptor       |
| Command              | Command execution       | 403 Forbidden                        |
| Workflow             | Workflow start          | 403 Forbidden                        |
| Workflow step        | Step advancement        | 403 Forbidden                        |
| Search provider      | Search execution        | Provider skipped                     |

### 8.7 Capability → Backend Permission Mapping

The BFF does NOT enforce backend permissions. It uses capabilities to determine
what the UI should show. When a command is executed, the BFF forwards the user's
original token to the backend service, which performs its own authorization.

The capability system is a **pre-filter** that prevents the UI from showing
actions the user cannot perform. This is a UX optimization, not a security
boundary. The security boundary is at the backend service.

However, the policy engine may consult the same permission store as the backend
services, ensuring consistency. The recommended pattern:

```
Backend permission: "orders.write"
  ↓ (mapped in policy engine)
UI capabilities: ["orders:edit:execute", "orders:create:view", "orders:cancel:execute"]
```

### 8.8 Hierarchical Capability Namespaces

Capabilities are hierarchical. Domain teams own their namespace:

```
orders:*                        ← all orders capabilities
orders:list:*                   ← all list capabilities
orders:list:view                ← specific capability
```

Wildcard resolution: `orders:*` implies `orders:list:view`, `orders:detail:edit`, etc.

Namespaces are enforced by convention and validated at definition load time.
Each domain definition may only declare capabilities under its own namespace.

---

## 9. UI Descriptor Model

Descriptors are the resolved, capability-filtered structures sent to the frontend.
They differ from definitions in these ways:

| Aspect          | Definition (internal)           | Descriptor (sent to frontend)       |
|-----------------|--------------------------------|-------------------------------------|
| Operation refs  | Contains operationId, handler  | Stripped; replaced with endpoints    |
| Capabilities    | Contains capability strings    | Already evaluated; fields filtered  |
| Field mapping   | Contains backend field names   | Contains UI field names only        |
| Conditions      | May reference capabilities     | Resolved to show/hide/enable states |
| Lookups         | Reference lookup IDs           | Resolved to option lists or endpoint URLs |

### 9.1 Descriptor Type Hierarchy

```
NavigationTree
  └── NavigationNode
       ├── label, icon, route
       ├── children: []NavigationNode
       └── badge: BadgeDescriptor

PageDescriptor
  ├── id, title, route, layout
  ├── breadcrumb: []BreadcrumbItem
  ├── table: TableDescriptor
  │    ├── columns: []ColumnDescriptor
  │    ├── filters: []FilterDescriptor
  │    ├── row_actions: []ActionDescriptor
  │    ├── bulk_actions: []ActionDescriptor
  │    └── data_endpoint, page_size, default_sort
  ├── sections: []SectionDescriptor
  │    └── fields: []FieldDescriptor
  └── actions: []ActionDescriptor

FormDescriptor
  ├── id, title
  ├── sections: []SectionDescriptor
  │    └── fields: []FieldDescriptor
  ├── actions: []ActionDescriptor
  ├── submit_endpoint
  └── success_route, success_message

ActionDescriptor
  ├── id, label, icon, style, type
  ├── enabled, visible
  ├── command_id (opaque to frontend — used in POST /ui/commands/{commandId})
  ├── navigate_to
  ├── workflow_id
  ├── confirmation: ConfirmationDescriptor
  └── conditions: []ConditionDescriptor  (client-side state-dependent visibility)

WorkflowDescriptor
  ├── id, workflow_id, name, status
  ├── current_step: StepDescriptor
  ├── steps: []StepDescriptor
  │    ├── id, name, type, status
  │    ├── form: FormDescriptor
  │    └── actions: []ActionDescriptor
  └── history: []WorkflowEventDescriptor
```

### 9.2 Descriptor Resolution Process

```
Definition + RequestContext + CapabilitySet → Descriptor

For each element in the definition:
  1. Check if the user has the required capabilities
  2. If not → omit the element from the descriptor
  3. If yes → resolve the element:
     a. Replace operation references with BFF endpoint URLs
     b. Resolve lookups to option lists or lookup endpoint URLs
     c. Evaluate field read-only expressions against capabilities
     d. Evaluate condition expressions
     e. Strip all internal metadata
  4. Assemble the filtered, resolved descriptor
```

---

## 10. Server-Driven UI Metadata APIs

### 10.1 Navigation: `GET /ui/navigation`

**Returns:** `NavigationTree` filtered by user capabilities.

**Resolution process:**
1. Iterate all domain definitions.
2. For each domain's navigation entry, check capabilities.
3. For each child navigation item, check capabilities.
4. Sort by `order` field.
5. Optionally resolve badges by invoking badge operations.
6. Return the filtered tree.

**Response:**
```json
{
  "items": [
    {
      "id": "orders",
      "label": "Orders",
      "icon": "shopping_cart",
      "children": [
        {
          "id": "orders.list",
          "label": "All Orders",
          "icon": "list",
          "route": "/orders",
          "badge": { "count": 12, "style": "warning" }
        }
      ]
    }
  ]
}
```

### 10.2 Page: `GET /ui/pages/{pageId}`

**Returns:** `PageDescriptor` filtered by user capabilities.

**Resolution process:**
1. Look up `PageDefinition` in registry.
2. Verify user has page-level capabilities → 403 if not.
3. Resolve table: filter columns, filters, actions by capabilities.
4. Resolve sections: filter fields by capabilities and visibility.
5. Resolve page-level actions by capabilities.
6. Replace data source references with BFF data endpoint URL.
7. Return descriptor.

### 10.3 Page Data: `GET /ui/pages/{pageId}/data`

**Query parameters:** `page`, `page_size`, `sort`, `sort_dir`, filters as
key-value pairs.

**Resolution process:**
1. Look up `PageDefinition` to find the `DataSourceDefinition`.
2. Verify capabilities.
3. Apply standard pagination/sort/filter mapping to the backend operation.
4. Invoke the backend via the Operation Invoker.
5. Apply `ResponseMapping` to transform the response.
6. Return `DataResponse`.

### 10.4 Form: `GET /ui/forms/{formId}`

**Returns:** `FormDescriptor`.

### 10.5 Form Data: `GET /ui/forms/{formId}/data?id={resourceId}`

**Returns:** Pre-populated form field values.

### 10.6 Command: `POST /ui/commands/{commandId}`

See [Section 11: Command and Action Model](#11-command-and-action-model).

### 10.7 Workflow: See [Section 12: Workflow Engine](#12-workflow-engine).

### 10.8 Search: `GET /ui/search?q={query}&page={page}&page_size={pageSize}`

See [Section 13: Global Search](#13-global-search).

### 10.9 Lookup: `GET /ui/lookups/{lookupId}?q={searchTerm}`

**Returns:** `LookupResponse` — list of options for a select/autocomplete.

---

## 11. Command and Action Model

### 11.1 Design Rationale

The frontend NEVER constructs API calls. All mutations flow through a single
command endpoint. This provides:

- **Uniform authorization:** Every mutation is capability-checked.
- **Uniform validation:** Payloads are validated against OpenAPI schemas.
- **Uniform observability:** Every mutation is logged, traced, metered.
- **Uniform error handling:** Backend errors are translated to UI errors.
- **Idempotency:** Commands can be made idempotent via keys.
- **Auditability:** Every mutation is tied to a subject, tenant, and correlation ID.

### 11.2 Command Endpoint

```
POST /ui/commands/{commandId}
Content-Type: application/json

{
  "input": { ... },               // user-provided data
  "route_params": { "id": "..." }, // current route parameters
  "idempotency_key": "..."         // optional
}
```

### 11.3 Command Execution Pipeline

```
POST /ui/commands/orders.update
     │
     ├── 1. Extract RequestContext (from middleware)
     │
     ├── 2. Look up CommandDefinition "orders.update" in registry
     │      → 404 if not found
     │
     ├── 3. Evaluate capabilities
     │      → 403 if user lacks required capabilities
     │
     ├── 4. Check idempotency (if configured)
     │      → Return cached result if duplicate key
     │
     ├── 5. Check rate limit (if configured)
     │      → 429 if rate exceeded
     │
     ├── 6. Apply InputMapping:
     │      a. Resolve path parameters from route_params, input, or context
     │      b. Resolve query parameters
     │      c. Resolve headers
     │      d. Build request body from input using body_mapping strategy:
     │         - "passthrough": send input as-is
     │         - "template": merge input values into body_template
     │         - "projection": map input fields to output fields via field_projection
     │
     ├── 7. Validate constructed request body against OpenAPI schema
     │      → 422 with field errors if validation fails
     │
     ├── 8. Invoke backend via OperationInvoker:
     │      → OpenAPIOperationInvoker for type: "openapi"
     │      → SDKOperationInvoker for type: "sdk"
     │
     ├── 9. Handle backend response:
     │      a. If success (2xx):
     │         - Apply OutputMapping to project/transform response
     │         - Store idempotency key → result (if configured)
     │         - Return CommandResponse { success: true, data: {...} }
     │      b. If client error (4xx):
     │         - Map backend error codes via error_map
     │         - Extract field-level validation errors
     │         - Return CommandResponse { success: false, errors: [...] }
     │      c. If server error (5xx):
     │         - Log error with full context
     │         - Return generic error (do NOT leak backend details)
     │
     └── 10. Emit metrics: command.executed, command.duration, command.error
```

### 11.4 CommandExecutor Interface

```
CommandExecutor
  ├── Execute(context, RequestContext, commandId string, input CommandInput) → (CommandResponse, error)
  └── Validate(commandId string, input CommandInput) → []FieldError
```

```
CommandInput
  ├── Input        map[string]any   // user-provided payload
  ├── RouteParams  map[string]string // current route parameters
  └── IdempotencyKey string
```

### 11.5 Input Mapping Expression Language

Input mappings use dot-path expressions to reference values:

| Expression              | Resolves to                                |
|------------------------|--------------------------------------------|
| `input.field_name`     | Value from user input payload              |
| `route.param_name`     | Value from current route parameters        |
| `context.subject_id`   | Value from RequestContext                  |
| `context.tenant_id`    | Value from RequestContext                  |
| `context.partition_id` | Value from RequestContext                  |
| `workflow.field_name`  | Value from workflow state (in workflow context) |

This is NOT a general-purpose expression language. It is a restricted set of
source references, validated at definition load time.

---

## 12. Workflow Engine

### 12.1 Design Goals

The workflow engine manages multi-step, stateful interactions that span multiple
user actions and backend calls. Examples:

- Order approval (review → approve/reject → process → notify)
- Dispute resolution (open → evidence → review → resolve)
- Onboarding (register → verify → configure → activate)

The engine is **definition-driven**: workflow structure is declared in YAML, not
coded. The engine executes the definition.

### 12.2 Workflow Lifecycle

```
                        ┌──────────┐
                        │  Start   │
                        └────┬─────┘
                             │
                    ┌────────▼────────┐
                    │  initial_step   │◄──── WorkflowInstance created
                    │  (e.g. review)  │      status = "active"
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │ approved     │ rejected     │ timeout
              ▼              ▼              ▼
        ┌──────────┐  ┌──────────┐  ┌──────────┐
        │ process  │  │ rejected │  │ expired  │
        │ (system) │  │(terminal)│  │(terminal)│
        └────┬─────┘  └──────────┘  └──────────┘
             │ completed
             ▼
        ┌──────────┐
        │  notify  │
        │ (system) │
        └────┬─────┘
             │ completed
             ▼
        ┌──────────┐
        │ approved │
        │(terminal)│
        └──────────┘
```

### 12.3 Workflow State Model

```
WorkflowInstance
  ├── ID               string              // unique instance ID (UUID)
  ├── WorkflowID       string              // references WorkflowDefinition.ID
  ├── TenantID         string              // tenant isolation
  ├── PartitionID      string              // partition isolation
  ├── SubjectID        string              // who started the workflow
  ├── CurrentStep      string              // current step ID
  ├── Status           string              // "active", "completed", "failed", "cancelled", "suspended"
  ├── State            map[string]any      // accumulated workflow state
  ├── CreatedAt        time.Time
  ├── UpdatedAt        time.Time
  ├── ExpiresAt        *time.Time          // computed from workflow timeout
  └── IdempotencyKey   string              // prevents duplicate workflow creation
```

```
WorkflowEvent
  ├── ID                  string
  ├── WorkflowInstanceID  string
  ├── StepID              string
  ├── Event               string            // "step_entered", "step_completed", "step_failed",
  │                                         // "approved", "rejected", "transition", "timeout"
  ├── ActorID             string            // who triggered this event
  ├── Data                map[string]any    // event payload
  ├── Comment             string            // optional human comment
  └── Timestamp           time.Time
```

### 12.4 Step Types

| Type           | Behavior                                                    |
|----------------|-------------------------------------------------------------|
| `action`       | Requires user action. Renders a form. Waits for input.      |
| `approval`     | Special action type with approve/reject semantics.          |
| `system`       | Executes a backend operation automatically (no user input).  |
| `wait`         | Pauses for a duration or external event.                     |
| `notification` | Sends a notification (fire-and-forget, non-blocking).        |
| `terminal`     | End state. No transitions out. Marks workflow as completed.  |

### 12.5 Workflow Engine Interface

```
WorkflowEngine
  ├── Start(context, RequestContext, workflowId string, input map[string]any) → (WorkflowInstance, error)
  ├── Advance(context, RequestContext, instanceId string, event string, input map[string]any) → (WorkflowInstance, error)
  ├── Get(context, RequestContext, instanceId string) → (WorkflowDescriptor, error)
  ├── Cancel(context, RequestContext, instanceId string, reason string) → error
  ├── List(context, RequestContext, filters WorkflowFilters) → ([]WorkflowInstance, error)
  └── ProcessTimeouts(context) → error    // called by background job
```

### 12.6 Workflow Execution: Start

```
POST /ui/workflows/orders.approval/start
  { "order_id": "ord-123", "customer_email": "..." }
     │
     ├── 1. Extract RequestContext
     ├── 2. Look up WorkflowDefinition "orders.approval"
     ├── 3. Evaluate workflow-level capabilities → 403 if missing
     ├── 4. Check idempotency (optional)
     ├── 5. Create WorkflowInstance:
     │      - ID: new UUID
     │      - WorkflowID: "orders.approval"
     │      - TenantID, PartitionID, SubjectID: from context
     │      - CurrentStep: initial_step ("review")
     │      - Status: "active"
     │      - State: { order_id: "ord-123", customer_email: "..." }
     │      - ExpiresAt: now + 72h
     ├── 6. Persist instance to WorkflowStore
     ├── 7. Append "step_entered" event for initial step
     ├── 8. If initial step is type "system":
     │      → Execute the step's operation immediately
     │      → Auto-advance on completion
     ├── 9. Return WorkflowDescriptor
     └── 10. Emit metrics: workflow.started
```

### 12.7 Workflow Execution: Advance

```
POST /ui/workflows/{instanceId}/advance
  { "event": "approved", "input": { "approval_notes": "Looks good" } }
     │
     ├── 1. Extract RequestContext
     ├── 2. Load WorkflowInstance from store
     ├── 3. Verify tenant isolation (instance.TenantID == context.TenantID)
     ├── 4. Verify instance status is "active"
     ├── 5. Look up current StepDefinition
     ├── 6. Evaluate step-level capabilities → 403 if missing
     ├── 7. Validate the event is valid for the current step (check transitions)
     ├── 8. Merge input into workflow state
     ├── 9. Find matching transition:
     │      - From: current step
     │      - Event: "approved"
     │      - Condition: evaluate against workflow state (if present)
     │      → 422 if no valid transition found
     ├── 10. Append "step_completed" event
     ├── 11. Transition to new step:
     │       a. Update CurrentStep
     │       b. Append "step_entered" event
     │       c. If new step is type "system":
     │          → Execute operation (see step execution below)
     │          → On success: auto-advance with "completed" event
     │          → On failure: record error, may transition to error step
     │       d. If new step is type "terminal":
     │          → Set Status = "completed"
     │          → Append "workflow_completed" event
     ├── 12. Persist updated instance
     ├── 13. Return updated WorkflowDescriptor
     └── 14. Emit metrics: workflow.step_advanced
```

### 12.8 System Step Execution

When the engine enters a `system` or `notification` step:

```
1. Look up step's OperationBinding
2. Apply InputMapping using workflow state as the source
3. Invoke backend via OperationInvoker
4. On success:
   a. Apply OutputMapping, merge result into workflow state
   b. Generate "completed" event
   c. Find and execute next transition
5. On error:
   a. Record error in workflow state
   b. Generate "error" event
   c. Find error transition (if defined)
   d. If no error transition: suspend the workflow
```

### 12.9 Workflow Timeout Processing

A background goroutine (or cron job) periodically:

1. Queries WorkflowStore for active instances where `ExpiresAt < now`.
2. For each expired instance:
   a. Check if current step has `on_timeout` transition.
   b. If yes: auto-advance with "timeout" event.
   c. If no: set Status = "failed", append "timeout" event.

Step-level timeouts follow the same pattern but check per-step expiry.

### 12.10 WorkflowStore Interface

```
WorkflowStore
  ├── Create(context, WorkflowInstance) → error
  ├── Get(context, tenantId string, instanceId string) → (WorkflowInstance, error)
  ├── Update(context, WorkflowInstance) → error
  ├── AppendEvent(context, WorkflowEvent) → error
  ├── GetEvents(context, tenantId string, instanceId string) → ([]WorkflowEvent, error)
  ├── FindActive(context, tenantId string, filters map[string]string) → ([]WorkflowInstance, error)
  ├── FindExpired(context, cutoff time.Time) → ([]WorkflowInstance, error)
  └── Delete(context, tenantId string, instanceId string) → error  // for cleanup
```

The store MUST enforce tenant isolation at the query level. A `Get` for a
different tenant than the authenticated user must return not-found, not forbidden
(to prevent ID enumeration).

Implementations:
- **PostgreSQL** (recommended for production) — using `workflow_instances` and
  `workflow_events` tables, partitioned by tenant_id.
- **In-memory** (for testing) — using maps protected by mutex.

---

## 13. Global Search

### 13.1 Search Flow

```
GET /ui/search?q=ORD-2024&page=1&page_size=20
     │
     ├── 1. Extract RequestContext and CapabilitySet
     │
     ├── 2. Get all SearchDefinitions from registry
     │
     ├── 3. Filter to those where user has required capabilities
     │
     ├── 4. For each eligible search provider (in parallel):
     │      a. Apply InputMapping: map "q" to the provider's search field
     │      b. Invoke backend via OperationInvoker
     │      c. Apply ResultMapping to extract search results
     │      d. Score/weight results based on provider weight
     │
     ├── 5. Merge results from all providers
     │
     ├── 6. Sort by score (descending)
     │
     ├── 7. Apply pagination (page, page_size)
     │
     └── 8. Return SearchResponse
```

### 13.2 SearchProvider Interface

```
SearchProvider
  ├── Search(context, RequestContext, query string, pagination Pagination) → (SearchResponse, error)
  └── (internally iterates all registered search definitions)
```

### 13.3 Parallel Execution with Timeout

Search providers are called in parallel with a per-provider timeout. If a
provider fails or times out, its results are omitted (not an error).
The response includes metadata about which providers responded.

### 13.4 Result Deduplication

If multiple providers return the same entity (matched by route + id), the
result with the highest score wins.

---

## 14. API Mapping and Invocation Engine

### 14.1 OpenAPI Index

At startup, Thesa loads OpenAPI specifications and builds an index:

```
OpenAPIIndex
  ├── Load(specs []SpecSource) → error
  ├── GetOperation(serviceId, operationId string) → (IndexedOperation, bool)
  ├── ValidateRequest(serviceId, operationId string, body any) → []ValidationError
  └── AllOperationIDs(serviceId string) → []string
```

```
SpecSource
  ├── ServiceID  string    // logical service name
  ├── BaseURL    string    // runtime base URL
  ├── SpecPath   string    // path to spec file (YAML or JSON)
  └── Timeout    duration  // per-request timeout for this service
```

```
IndexedOperation
  ├── ServiceID     string
  ├── OperationID   string
  ├── Method        string         // GET, POST, PUT, DELETE, PATCH
  ├── PathTemplate  string         // e.g., "/api/v1/orders/{orderId}"
  ├── Parameters    []ParameterDef // path, query, header params with types
  ├── RequestBody   *SchemaDef     // JSON Schema for request body
  ├── Responses     map[int]SchemaDef  // status code → response schema
  ├── Security      []SecurityDef
  └── BaseURL       string         // resolved from SpecSource
```

### 14.2 OpenAPI Invoker

The OpenAPI invoker is the primary `OperationInvoker` implementation.

**Invocation sequence:**

```
Invoke(context, requestContext, binding, input)
     │
     ├── 1. Lookup: index.GetOperation(binding.ServiceID, binding.OperationID)
     │      → error if not found (should not happen — validated at startup)
     │
     ├── 2. Build URL:
     │      a. Start with operation.BaseURL + operation.PathTemplate
     │      b. Substitute path parameters from input.PathParams
     │      c. Append query parameters from input.QueryParams
     │
     ├── 3. Build headers:
     │      a. Set Content-Type: application/json (if body present)
     │      b. Set Accept: application/json
     │      c. Forward Authorization header from original request
     │      d. Set X-Tenant-Id, X-Partition-Id, X-Correlation-Id from context
     │      e. Set traceparent from current span
     │      f. Add any custom headers from input.Headers
     │
     ├── 4. Validate request body against operation.RequestBody schema
     │      → error if validation fails
     │
     ├── 5. Execute HTTP request:
     │      a. Use connection pool (per service)
     │      b. Apply circuit breaker (per service)
     │      c. Apply timeout (from service config)
     │      d. Apply retry policy (idempotent methods only: GET, PUT, DELETE)
     │
     ├── 6. Parse response:
     │      a. Read status code
     │      b. Parse JSON body
     │      c. Return InvocationResult { StatusCode, Body, Headers }
     │
     └── 7. Emit metrics: invoker.openapi.duration, invoker.openapi.status
```

### 14.3 SDK Invoker Registry

```
SDKInvokerRegistry
  ├── Register(name string, handler SDKHandler)
  ├── Get(name string) → (SDKHandler, bool)
  └── Names() → []string
```

```
SDKHandler interface
  ├── Invoke(context, RequestContext, input InvocationInput) → (InvocationResult, error)
  └── Name() string
```

SDK handlers are registered at application startup:

```go
// Example registration (pseudo-code)
registry.Register("ledger.PostEntry", &LedgerPostEntryHandler{client: ledgerClient})
registry.Register("notifications.SendOrderApproved", &NotificationHandler{client: notifClient})
```

The SDK invoker implementation wraps the registry:

```
SDKOperationInvoker
  ├── Supports(binding) → bool  // true if binding.Type == "sdk"
  └── Invoke(context, requestContext, binding, input):
       1. Look up handler by binding.Handler in registry
       2. Call handler.Invoke(context, requestContext, input)
       3. Return result
```

### 14.4 Pagination, Filtering, and Sorting Standardization

The BFF standardizes pagination across all backend services:

**Frontend sends (standard):**
```
?page=2&page_size=25&sort=created_at&sort_dir=desc&status=pending
```

**BFF maps to backend (per definition's InputMapping):**

The definition's `DataSourceDefinition` specifies how standard pagination
parameters map to the backend's conventions:

```yaml
data_source:
  operation_id: "listOrders"
  service_id: "orders-svc"
  mapping:
    items_path: "data.orders"
    total_path: "meta.total"
    field_map:
      created_at: "createdAt"    # sort field name translation
```

The invoker translates:
- `page` / `page_size` → backend's pagination parameters (configurable per service:
  `offset`/`limit`, `page`/`per_page`, `cursor`, etc.)
- `sort` → backend sort field name (using field_map for translation)
- Filter parameters → backend query parameters (using filter definitions)

Pagination strategy is configured per service:

```yaml
# In service configuration (not definition files)
services:
  orders-svc:
    base_url: "https://orders.internal"
    pagination:
      style: "offset"           # offset | cursor | page
      page_param: "offset"
      size_param: "limit"
      sort_param: "sort_by"
      sort_dir_param: "order"
```

### 14.5 Update Masks

For PATCH operations, the BFF can generate update masks from form input.
Only fields that were actually changed by the user are included in the
backend request.

The frontend sends:

```json
{
  "input": {
    "shipping_address": "123 New St",
    "notes": "Updated notes"
  },
  "_changed_fields": ["shipping_address", "notes"]
}
```

The BFF uses `_changed_fields` to:
1. Include only changed fields in the request body.
2. Set an update mask header (e.g., `X-Update-Mask: shippingAddress,notes`)
   if the backend supports field masks.

---

## 15. Schema and Contract Stability

### 15.1 The Adapter Layer

The BFF sits as an adapter between two independently evolving contracts:

```
Frontend Contract          BFF Adapter Layer          Backend Contract
(descriptors)             (definitions)              (OpenAPI specs)
                                │
stable, versioned         mapping rules          evolves independently
changed only by           authored by            changed by domain teams
BFF/UI teams              domain teams
```

### 15.2 Response Projection

Backend responses are never forwarded directly. The `ResponseMapping` and
`OutputMapping` in definitions control exactly which fields are extracted
and how they're renamed.

When a backend service renames a field:
- **Old:** `{ "orderNumber": "ORD-123" }`
- **New:** `{ "order_num": "ORD-123" }`

The domain team updates only the definition's field_map:
```yaml
field_map:
  order_number: "order_num"   # changed from "orderNumber"
```

The frontend continues to see `order_number`. No frontend change required.

### 15.3 Versioned Descriptor Contracts

The BFF API uses a version prefix in the Accept header:

```
Accept: application/vnd.thesa.v1+json
```

When the descriptor schema must change in a breaking way, a new version
is introduced. The BFF can serve both v1 and v2 simultaneously during
migration.

### 15.4 Compatibility Rules for Definitions

When updating a definition file:

| Change Type                    | Compatibility     | Notes                           |
|-------------------------------|-------------------|---------------------------------|
| Add new page                  | Compatible        | Frontend may not render it yet  |
| Add new column to table       | Compatible        | Frontend ignores unknown fields |
| Remove column from table      | Compatible        | Frontend adapts                 |
| Change field_map              | Compatible        | Backend change absorbed         |
| Change operation_id           | Compatible        | Transparent to frontend         |
| Change page ID                | Breaking          | Frontend routes change          |
| Change command ID             | Breaking          | Frontend action references break|
| Change field names in form    | Breaking          | Frontend input mapping breaks   |

Breaking changes require coordinated frontend + definition deployment.

---

## 16. Error Handling and Validation

### 16.1 Error Envelope

All error responses use a consistent envelope:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "One or more fields are invalid",
    "details": [
      {
        "field": "shipping_address",
        "code": "REQUIRED",
        "message": "Shipping address is required"
      },
      {
        "field": "priority",
        "code": "INVALID_VALUE",
        "message": "Priority must be one of: normal, high, urgent"
      }
    ],
    "trace_id": "abc123"
  }
}
```

### 16.2 Error Categories

| HTTP Status | Code                  | When                                             |
|-------------|----------------------|--------------------------------------------------|
| 400         | BAD_REQUEST          | Malformed JSON, missing required top-level fields |
| 401         | UNAUTHORIZED         | Missing or invalid token                          |
| 403         | FORBIDDEN            | Valid token but insufficient capabilities         |
| 404         | NOT_FOUND            | Page, command, workflow, or resource not found     |
| 409         | CONFLICT             | Idempotency conflict or stale state               |
| 422         | VALIDATION_ERROR     | Input validation failed (field-level details)     |
| 429         | RATE_LIMITED         | Rate limit exceeded                               |
| 500         | INTERNAL_ERROR       | Unexpected server error (no backend details leaked)|
| 502         | BACKEND_UNAVAILABLE  | Backend service unreachable                       |
| 504         | BACKEND_TIMEOUT      | Backend service timed out                         |

### 16.3 Backend Error Translation

Backend errors are never forwarded directly. The command definition's `error_map`
translates known backend error codes to UI-friendly messages.

Unknown backend errors are logged and replaced with a generic message.

```
Backend response: 422 { "error": "ORDER_NOT_FOUND", "message": "Order xyz not found" }

Definition error_map:
  ORDER_NOT_FOUND: "This order no longer exists. It may have been deleted."

BFF response: 422 {
  "error": {
    "code": "ORDER_NOT_FOUND",
    "message": "This order no longer exists. It may have been deleted.",
    "trace_id": "abc123"
  }
}
```

### 16.4 Validation Layers

1. **Transport validation:** JSON parsing, required envelope fields.
2. **Definition validation:** Command exists, required input fields present.
3. **Capability validation:** User authorized for this command.
4. **Schema validation:** Input body matches OpenAPI request schema.
5. **Backend validation:** Backend returns field-level errors (mapped through).

Each layer short-circuits: if transport validation fails, later layers are skipped.

### 16.5 Workflow-Specific Errors

| Code                     | When                                                |
|--------------------------|-----------------------------------------------------|
| WORKFLOW_NOT_FOUND       | Instance ID doesn't exist (or different tenant)     |
| WORKFLOW_NOT_ACTIVE      | Attempting to advance a completed/cancelled workflow |
| INVALID_TRANSITION       | Event is not valid for the current step             |
| STEP_UNAUTHORIZED        | User lacks capability for the current step          |
| WORKFLOW_EXPIRED         | Workflow or step has timed out                      |

---

## 17. Observability and Reliability

### 17.1 Structured Logging

All logs are JSON-structured with consistent fields:

```json
{
  "level": "info",
  "msg": "command executed",
  "command_id": "orders.update",
  "tenant_id": "t-123",
  "subject_id": "u-456",
  "correlation_id": "corr-789",
  "trace_id": "trace-abc",
  "duration_ms": 145,
  "backend_status": 200,
  "timestamp": "2025-01-15T10:30:00Z"
}
```

Required fields on every log entry:
- `tenant_id`, `subject_id`, `correlation_id` (from RequestContext)
- `trace_id` (from OpenTelemetry)
- `level`, `msg`, `timestamp`

Sensitive data (token contents, request bodies) are NEVER logged. Request bodies
may be logged at DEBUG level with PII fields redacted.

### 17.2 Distributed Tracing

Using OpenTelemetry with the following span hierarchy:

```
Span: HTTP Request (transport layer)
  └── Span: Command Execution (command/executor)
       └── Span: Capability Resolution (capability/resolver)
       └── Span: Input Mapping (command/executor)
       └── Span: Backend Invocation (invoker/openapi)
            └── Span: HTTP Client Call
       └── Span: Output Mapping (command/executor)
```

Trace context is propagated to backend services via the `traceparent` header.

### 17.3 Metrics

Metrics are exposed in Prometheus format at `/metrics`.

| Metric                             | Type      | Labels                              |
|-------------------------------------|-----------|--------------------------------------|
| `thesa_http_requests_total`         | Counter   | method, path, status                |
| `thesa_http_request_duration_seconds` | Histogram | method, path                       |
| `thesa_command_executions_total`    | Counter   | command_id, status                  |
| `thesa_command_duration_seconds`    | Histogram | command_id                          |
| `thesa_workflow_starts_total`       | Counter   | workflow_id                         |
| `thesa_workflow_advances_total`     | Counter   | workflow_id, step_id, event         |
| `thesa_workflow_active_instances`   | Gauge     | workflow_id                         |
| `thesa_backend_requests_total`      | Counter   | service_id, operation_id, status    |
| `thesa_backend_request_duration_seconds` | Histogram | service_id                     |
| `thesa_backend_circuit_breaker_state` | Gauge   | service_id                          |
| `thesa_capability_cache_hits_total` | Counter   | —                                   |
| `thesa_capability_cache_misses_total` | Counter | —                                   |
| `thesa_definition_reload_total`     | Counter   | status (success/failure)            |
| `thesa_search_duration_seconds`     | Histogram | —                                   |

### 17.4 Circuit Breakers

Each backend service has an independent circuit breaker:

```
States: Closed → Open → Half-Open → Closed

Closed:  requests flow through normally
         → transitions to Open after N consecutive failures or error rate > threshold

Open:    all requests fail immediately with 502
         → transitions to Half-Open after timeout

Half-Open: allows one probe request
           → if success: transition to Closed
           → if failure: transition to Open
```

Configuration per service:
```yaml
services:
  orders-svc:
    circuit_breaker:
      failure_threshold: 5        # failures before opening
      success_threshold: 2        # successes in half-open before closing
      timeout: 30s                # how long to stay open
      error_rate_threshold: 0.5   # alternative: error rate
      error_rate_window: 60s
```

### 17.5 Retries

Retries are applied only for:
- Idempotent HTTP methods (GET, PUT, DELETE) — never for POST unless the
  command has idempotency configured.
- Network errors and 502/503/504 responses.
- Maximum 3 retries with exponential backoff (100ms, 200ms, 400ms).

### 17.6 Timeouts

Three-layer timeout model:

1. **Client timeout** (Flutter → BFF): 30s default
2. **BFF processing timeout** (overall handler): 25s (must be less than client)
3. **Backend call timeout** (BFF → backend): 10s per call (configurable per service)

---

## 18. Security Model

### 18.1 Threat Model

| Threat                          | Mitigation                                            |
|--------------------------------|-------------------------------------------------------|
| Cross-tenant data access       | Tenant ID from verified JWT, not from input; enforced in every query |
| Privilege escalation           | Capabilities resolved from policy engine, not from client |
| Direct backend API invocation  | Backend services accept only BFF-originated requests (mTLS or API gateway rules) |
| Command replay                 | Idempotency keys with TTL; replay returns cached result |
| Workflow tampering             | Workflow state in server-side store; all transitions validated |
| Definition tampering           | SHA-256 checksums computed at load; read-only filesystem in production |
| Token theft                    | Short-lived tokens; token binding to device (optional) |
| Injection (SQL, XSS, etc.)    | Input validated against OpenAPI schemas; no string interpolation in queries |
| Enumeration attacks            | Not-found for wrong tenant (not forbidden); rate limiting |

### 18.2 Tenant Isolation Enforcement

**Structural rule:** Tenant ID is NEVER accepted from request parameters or body.
It is ALWAYS extracted from the verified JWT.

```
# CORRECT: tenant from token
requestContext.TenantID = jwt.Claims["tenant_id"]

# INCORRECT: tenant from header/body — NEVER
requestContext.TenantID = request.Header.Get("X-Tenant-Id")  // ← WRONG
```

The partition (workspace) ID IS accepted from a header (`X-Partition-Id`), but
the middleware validates that the partition belongs to the authenticated tenant.

### 18.3 Definition Integrity

At startup:
1. Load each definition file.
2. Compute SHA-256 of file contents.
3. Store checksum in memory.
4. Optionally verify checksum against a manifest file signed by CI/CD.

At runtime (if hot-reload is enabled):
1. Compute checksum of new file.
2. Log the change with old and new checksums.
3. In strict mode: reject reload if checksums don't match signed manifest.

### 18.4 Backend Authentication

The BFF authenticates to backend services using one of:

1. **Token forwarding:** The user's JWT is forwarded as-is. Backend re-validates.
   This is the default and recommended approach.

2. **Service-to-service token:** The BFF obtains its own service token (via
   client credentials flow) and forwards it alongside the user context in
   custom headers. Used when backend services require machine-to-machine auth.

3. **mTLS:** The BFF presents a client certificate to backend services.
   Used in high-security environments.

Configuration per service:

```yaml
services:
  orders-svc:
    auth:
      strategy: "forward_token"    # forward_token | service_token | mtls

  ledger-svc:
    auth:
      strategy: "service_token"
      client_id: "thesa-bff"
      token_endpoint: "https://auth.internal/oauth/token"
```

### 18.5 Request Sanitization

The BFF does NOT blindly forward request bodies. Every command goes through:

1. **Input mapping:** Only explicitly mapped fields are extracted.
2. **Schema validation:** The constructed body is validated against the
   OpenAPI schema.
3. **No raw passthrough in production:** Even when `body_mapping: passthrough`
   is used, the body is still validated against the operation's schema.

---

## 19. Core Abstractions and Interfaces

### Complete Interface Catalog

```
┌─────────────────────┐
│   RequestContext     │  Value object: identity + tenancy + tracing
└─────────────────────┘

┌─────────────────────┐
│  DefinitionLoader   │  Loads YAML/JSON definitions from filesystem
│  ├ LoadAll()        │  Watches for changes (hot-reload)
│  ├ LoadFile()       │
│  └ Watch()          │
└─────────────────────┘

┌─────────────────────┐
│ DefinitionRegistry  │  Read-optimized store of loaded definitions
│  ├ GetPage()        │  Supports atomic replacement for hot-reload
│  ├ GetForm()        │  Thread-safe for concurrent reads
│  ├ GetCommand()     │
│  ├ GetWorkflow()    │
│  ├ GetSearch()      │
│  ├ GetLookup()      │
│  ├ AllDomains()     │
│  └ Replace()        │
└─────────────────────┘

┌─────────────────────┐
│CapabilityResolver   │  Resolves capabilities for a request context
│  ├ Resolve()        │  Caches results per (subject, tenant, partition)
│  └ Invalidate()     │
└─────────────────────┘

┌─────────────────────┐
│  PolicyEvaluator    │  Fine-grained policy evaluation
│  ├ Evaluate()       │  Called by CapabilityResolver
│  ├ EvaluateAll()    │  May call external engines (OPA, Cedar)
│  └ Sync()           │
└─────────────────────┘

┌─────────────────────┐
│  MenuProvider       │  Builds NavigationTree from definitions + capabilities
│  └ GetMenu()        │
└─────────────────────┘

┌─────────────────────┐
│  PageProvider       │  Resolves PageDescriptor from definition + capabilities
│  ├ GetPage()        │
│  └ GetPageData()    │
└─────────────────────┘

┌─────────────────────┐
│  FormProvider       │  Resolves FormDescriptor from definition + capabilities
│  ├ GetForm()        │
│  └ GetFormData()    │
└─────────────────────┘

┌─────────────────────┐
│  ActionProvider     │  Resolves ActionDescriptor from definition + capabilities
│  └ ResolveActions() │  Evaluates conditions against resource state
└─────────────────────┘

┌─────────────────────┐
│ CommandExecutor     │  Executes commands through the full pipeline
│  ├ Execute()        │  Validates, authorizes, maps, invokes, translates
│  └ Validate()       │
└─────────────────────┘

┌─────────────────────┐
│  WorkflowEngine     │  Manages workflow lifecycle
│  ├ Start()          │  Persistent, resumable, timeout-aware
│  ├ Advance()        │
│  ├ Get()            │
│  ├ Cancel()         │
│  ├ List()           │
│  └ ProcessTimeouts()│
└─────────────────────┘

┌─────────────────────┐
│  WorkflowStore      │  Persists workflow instances and events
│  ├ Create()         │  Tenant-isolated at query level
│  ├ Get()            │  Implementations: PostgreSQL, in-memory
│  ├ Update()         │
│  ├ AppendEvent()    │
│  ├ GetEvents()      │
│  ├ FindActive()     │
│  └ FindExpired()    │
└─────────────────────┘

┌─────────────────────┐
│  SearchProvider     │  Aggregates search across domains
│  └ Search()         │  Parallel execution with timeout
└─────────────────────┘

┌─────────────────────┐
│ OperationInvoker    │  Unified interface for backend invocation
│  ├ Invoke()         │  Dispatches to OpenAPI or SDK invoker
│  └ Supports()       │
└─────────────────────┘

┌─────────────────────┐
│OpenAPIOperationInvoker│ Dynamic HTTP invocation from OpenAPI specs
│  └ (implements       │  Builds requests at runtime
│     OperationInvoker)│  No code generation
└─────────────────────┘

┌─────────────────────┐
│SDKOperationInvoker  │  Typed client invocation for specific handlers
│  └ (implements       │  Wraps SDKInvokerRegistry
│     OperationInvoker)│
└─────────────────────┘

┌─────────────────────┐
│  OpenAPIIndex       │  Indexes OpenAPI specs by (serviceId, operationId)
│  ├ Load()           │  Validates definitions against specs
│  ├ GetOperation()   │  Provides schema validation
│  └ ValidateRequest()│
└─────────────────────┘

┌─────────────────────┐
│ InvokerRegistry     │  Holds all OperationInvoker implementations
│  ├ Register()       │  Dispatches based on OperationBinding.Type
│  └ Invoke()         │
└─────────────────────┘
```

### Interface Dependency Graph

```
Transport (HTTP Handlers)
  ├── MenuProvider
  │    ├── DefinitionRegistry
  │    └── CapabilityResolver
  ├── PageProvider
  │    ├── DefinitionRegistry
  │    ├── CapabilityResolver
  │    └── InvokerRegistry (for data fetching)
  ├── FormProvider
  │    ├── DefinitionRegistry
  │    ├── CapabilityResolver
  │    └── InvokerRegistry (for data loading)
  ├── CommandExecutor
  │    ├── DefinitionRegistry
  │    ├── CapabilityResolver
  │    ├── InvokerRegistry
  │    └── OpenAPIIndex (for validation)
  ├── WorkflowEngine
  │    ├── DefinitionRegistry
  │    ├── CapabilityResolver
  │    ├── InvokerRegistry
  │    └── WorkflowStore
  └── SearchProvider
       ├── DefinitionRegistry
       ├── CapabilityResolver
       └── InvokerRegistry
```

No circular dependencies. All arrows point downward.

---

## 20. Go Package Structure

```
github.com/pitabwire/thesa/
│
├── cmd/
│   └── bff/
│       └── main.go                 # Application entry point, wiring
│
├── model/                          # Public domain types
│   ├── context.go                  # RequestContext, context helpers
│   ├── capability.go               # Capability, CapabilitySet, interfaces
│   ├── definition.go               # All definition types (DomainDefinition, etc.)
│   ├── descriptor.go               # All descriptor types (PageDescriptor, etc.)
│   ├── workflow.go                 # WorkflowInstance, WorkflowEvent
│   ├── invoker.go                  # OperationInvoker, InvocationInput/Result
│   └── errors.go                   # ErrorEnvelope, FieldError, error codes
│
├── internal/                       # Private implementations
│   ├── config/
│   │   └── config.go               # Application configuration
│   │
│   ├── definition/
│   │   ├── loader.go               # YAML file loading, checksumming
│   │   ├── registry.go             # In-memory definition store
│   │   └── validator.go            # Definition validation against OpenAPI
│   │
│   ├── capability/
│   │   ├── resolver.go             # CapabilityResolver with caching
│   │   └── policy.go               # PolicyEvaluator implementations
│   │
│   ├── metadata/
│   │   ├── menu.go                 # MenuProvider
│   │   ├── page.go                 # PageProvider
│   │   └── form.go                 # FormProvider
│   │
│   ├── command/
│   │   └── executor.go             # CommandExecutor
│   │
│   ├── workflow/
│   │   ├── engine.go               # WorkflowEngine
│   │   ├── store.go                # WorkflowStore interface + PostgreSQL impl
│   │   └── memstore.go             # In-memory WorkflowStore (testing)
│   │
│   ├── search/
│   │   └── provider.go             # SearchProvider
│   │
│   ├── invoker/
│   │   ├── registry.go             # InvokerRegistry
│   │   ├── openapi.go              # OpenAPIOperationInvoker
│   │   └── sdk.go                  # SDKOperationInvoker + handler registry
│   │
│   ├── openapi/
│   │   └── index.go                # OpenAPI spec loading and indexing
│   │
│   ├── transport/
│   │   ├── router.go               # HTTP router setup (chi)
│   │   ├── middleware.go            # Auth, context, recovery, CORS
│   │   ├── handler_navigation.go   # GET /ui/navigation
│   │   ├── handler_page.go         # GET /ui/pages/{pageId}, /data
│   │   ├── handler_form.go         # GET /ui/forms/{formId}, /data
│   │   ├── handler_command.go      # POST /ui/commands/{commandId}
│   │   ├── handler_workflow.go     # Workflow endpoints
│   │   ├── handler_search.go       # GET /ui/search
│   │   ├── handler_lookup.go       # GET /ui/lookups/{lookupId}
│   │   └── response.go             # Response helpers, error rendering
│   │
│   └── observability/
│       └── telemetry.go            # Tracing, metrics, logging setup
│
├── definitions/                    # Domain definition files (YAML)
│   ├── orders/
│   │   └── definition.yaml
│   ├── inventory/
│   │   └── definition.yaml
│   └── customers/
│       └── definition.yaml
│
├── specs/                          # OpenAPI specification files
│   ├── orders-svc.yaml
│   ├── inventory-svc.yaml
│   └── customers-svc.yaml
│
├── config/                         # Runtime configuration
│   ├── config.yaml                 # Service URLs, timeouts, etc.
│   └── config.production.yaml
│
├── go.mod
├── go.sum
└── ARCHITECTURE.md                 # This document
```

### Package Responsibilities

| Package               | Responsibility                                             | Dependencies (internal)         |
|-----------------------|-----------------------------------------------------------|---------------------------------|
| `model`               | Shared types, interfaces                                  | None                            |
| `internal/config`     | Configuration loading                                     | None                            |
| `internal/openapi`    | OpenAPI spec loading, indexing, schema validation          | `model`                         |
| `internal/definition` | Definition loading, registry, validation                  | `model`, `openapi`              |
| `internal/capability` | Capability resolution, policy evaluation, caching         | `model`                         |
| `internal/invoker`    | Operation invocation (OpenAPI + SDK)                      | `model`, `openapi`              |
| `internal/metadata`   | Menu, page, form descriptor resolution                    | `model`, `definition`, `capability`, `invoker` |
| `internal/command`    | Command execution pipeline                                | `model`, `definition`, `capability`, `invoker`, `openapi` |
| `internal/workflow`   | Workflow engine and storage                               | `model`, `definition`, `capability`, `invoker` |
| `internal/search`     | Global search aggregation                                 | `model`, `definition`, `capability`, `invoker` |
| `internal/transport`  | HTTP handlers, middleware, routing                        | All of the above                |
| `internal/observability` | Telemetry setup                                        | None (configures global state)  |
| `cmd/bff`             | Wiring, startup, shutdown                                 | All of the above                |

### Key Design Decisions

1. **`model/` is NOT under `internal/`**: The type package is public so that
   definition tooling, test utilities, and SDK handler implementations in
   separate modules can import it.

2. **One handler file per resource**: Transport handlers are split by resource
   (navigation, page, form, command, workflow, search, lookup) for maintainability.

3. **No `pkg/` directory**: The only public package is `model/`. Everything else
   is internal.

4. **`definitions/` and `specs/` are sibling directories**: Both are loaded
   at startup. In production, they may be mounted from different artifact
   sources.

---

## 21. Example Domain: Orders

### 21.1 Complete Definition File

See section 7 for the full YAML. The orders domain defines:

| Element               | ID                      | Description                          |
|-----------------------|-------------------------|--------------------------------------|
| Page (list)           | `orders.list`           | Paginated, filterable order table    |
| Page (detail)         | `orders.detail`         | Order detail with sections           |
| Form (edit)           | `orders.edit_form`      | Edit order details                   |
| Form (approval)       | `orders.approval_form`  | Approval form with notes             |
| Command (update)      | `orders.update`         | Update order via PATCH               |
| Command (cancel)      | `orders.cancel`         | Cancel order via POST                |
| Workflow (approval)   | `orders.approval`       | Multi-step approval workflow         |
| Workflow (cancellation)| `orders.cancellation`  | Cancellation with reason             |
| Search                | `orders.search`         | Global search integration            |
| Lookup                | `orders.statuses`       | Order status options                 |

### 21.2 Capability Namespace

```
orders:nav:view              # can see the Orders section in navigation
orders:list:view             # can see the orders list page
orders:detail:view           # can see order detail pages
orders:detail:edit           # can see edit action on detail page
orders:create:view           # can see the create order page
orders:edit:execute          # can execute the update command
orders:cancel:execute        # can execute the cancel command
orders:approve:execute       # can execute the approval workflow
orders:export:execute        # can execute bulk export
orders:line_items:view       # can see the line items section
orders:notes:view            # can see the notes section
orders:notes:edit            # can edit the notes field
orders:search:execute        # results appear in global search
```

### 21.3 Referenced OpenAPI Operations

These operation IDs must exist in the `orders-svc` OpenAPI spec:

| operationId       | Method | Path                        | Purpose              |
|-------------------|--------|-----------------------------|----------------------|
| `listOrders`      | GET    | /api/v1/orders              | List with pagination |
| `getOrder`        | GET    | /api/v1/orders/{orderId}    | Get single order     |
| `updateOrder`     | PATCH  | /api/v1/orders/{orderId}    | Update order         |
| `cancelOrder`     | POST   | /api/v1/orders/{orderId}/cancel | Cancel order     |
| `confirmOrder`    | POST   | /api/v1/orders/{orderId}/confirm | Confirm (approve) |
| `searchOrders`    | GET    | /api/v1/orders/search       | Full-text search     |
| `getOrderStatuses`| GET    | /api/v1/orders/statuses     | Status enum          |

---

## 22. Example End-to-End Flow

### Scenario: User approves an order

**Actors:**
- Alice (role: `order_approver`, tenant: `acme-corp`, partition: `us-west`)
- Orders Service (backend)
- Notification Service (backend, SDK)

**Flow:**

```
Step 1: Alice opens the Orders list page
─────────────────────────────────────────
  Flutter → GET /ui/navigation
    BFF: resolve capabilities for Alice → {orders:nav:view, orders:list:view,
         orders:detail:view, orders:approve:execute, ...}
    BFF: build NavigationTree, include "Orders" section
    Flutter: render sidebar with "All Orders" link

  Flutter → GET /ui/pages/orders.list
    BFF: verify orders:list:view capability ✓
    BFF: resolve table columns, filters, actions
    BFF: Alice has orders:approve:execute, so "Approve" row action is visible
    Flutter: render page shell with table descriptor

  Flutter → GET /ui/pages/orders.list/data?page=1&page_size=25&sort=created_at&sort_dir=desc
    BFF: invoke listOrders on orders-svc
    BFF: map response via items_path "data.orders", rename fields via field_map
    Flutter: render table rows


Step 2: Alice clicks on order ORD-2024-001
──────────────────────────────────────────
  Flutter → GET /ui/pages/orders.detail?id=ord-123
    BFF: invoke getOrder(orderId=ord-123)
    BFF: resolve sections and actions based on capabilities and order status
    BFF: order.status == "pending" → "Approve" action visible (condition met)
    BFF: Alice has orders:approve:execute → action included
    Flutter: render detail page with "Approve" button


Step 3: Alice clicks "Approve"
──────────────────────────────
  Flutter sees action: { type: "workflow", workflow_id: "orders.approval" }
  Flutter → POST /ui/workflows/orders.approval/start
    Body: { "order_id": "ord-123", "customer_email": "bob@example.com" }

    BFF: verify orders:approve:execute capability ✓
    BFF: create WorkflowInstance:
      - ID: "wf-001"
      - WorkflowID: "orders.approval"
      - CurrentStep: "review"
      - State: { order_id: "ord-123", customer_email: "bob@example.com" }
      - Status: "active"
    BFF: persist instance + "step_entered" event
    BFF: return WorkflowDescriptor with current step = "review"

  Flutter: render approval form (form_id: "orders.approval_form")
    - shows approval notes textarea
    - shows "Approve" and "Reject" buttons


Step 4: Alice fills in notes and clicks "Approve"
──────────────────────────────────────────────────
  Flutter → POST /ui/workflows/wf-001/advance
    Body: {
      "event": "approved",
      "input": { "approval_notes": "Verified with warehouse. Stock available.",
                 "approved_by": "alice@acme-corp.com" }
    }

    BFF: load WorkflowInstance wf-001
    BFF: verify tenant isolation (instance.TenantID == alice.TenantID) ✓
    BFF: verify instance status == "active" ✓
    BFF: current step = "review", event = "approved"
    BFF: find transition: review → process (event: approved) ✓
    BFF: merge input into workflow state
    BFF: append "step_completed" event for "review"
    BFF: transition to "process" step
    BFF: "process" step is type "system" → execute automatically:
      │
      ├── invoke confirmOrder on orders-svc:
      │   path: /api/v1/orders/ord-123/confirm
      │   body: { "approvedBy": "alice@acme-corp.com",
      │           "approvalNotes": "Verified with warehouse. Stock available." }
      │   response: 200 OK
      │
      ├── append "step_completed" event for "process"
      ├── find transition: process → notify (event: completed)
      ├── transition to "notify" step
      │
      ├── "notify" step is type "system" → execute automatically:
      │   invoke SDK handler "notifications.SendOrderApproved":
      │   input: { orderId: "ord-123", customerEmail: "bob@example.com" }
      │   response: OK (or error — non-blocking)
      │
      ├── append "step_completed" event for "notify"
      ├── find transition: notify → approved (event: completed)
      ├── transition to "approved" step
      │
      ├── "approved" step is type "terminal"
      │   → set instance Status = "completed"
      │   → append "workflow_completed" event
      │
      └── persist final state

    BFF: return WorkflowDescriptor:
      {
        "id": "wf-001",
        "workflow_id": "orders.approval",
        "name": "Order Approval",
        "status": "completed",
        "current_step": { "id": "approved", "name": "Approved", "type": "terminal", "status": "completed" },
        "steps": [
          { "id": "review", "name": "Review Order", "status": "completed" },
          { "id": "process", "name": "Process Approved Order", "status": "completed" },
          { "id": "notify", "name": "Send Notification", "status": "completed" },
          { "id": "approved", "name": "Approved", "status": "completed" }
        ],
        "history": [
          { "step_name": "Review Order", "event": "approved", "actor": "alice@acme-corp.com", "timestamp": "..." },
          { "step_name": "Process Approved Order", "event": "completed", "actor": "system", "timestamp": "..." },
          { "step_name": "Send Notification", "event": "completed", "actor": "system", "timestamp": "..." }
        ]
      }

  Flutter: render success state — "Order approved" with workflow history


Step 5: Alice returns to the orders list
─────────────────────────────────────────
  Flutter → GET /ui/pages/orders.list/data?...
    BFF: invoke listOrders
    BFF: order ORD-2024-001 now shows status: "confirmed"
    Flutter: render updated table
```

### Flow Summary

```
Alice ──────── Thesa BFF ──────── Orders Service ──── Notification Service
  │                 │                    │                     │
  │  GET /navigation│                    │                     │
  │────────────────▶│                    │                     │
  │  NavigationTree │                    │                     │
  │◀────────────────│                    │                     │
  │                 │                    │                     │
  │  GET /pages/... │                    │                     │
  │────────────────▶│                    │                     │
  │  PageDescriptor │                    │                     │
  │◀────────────────│                    │                     │
  │                 │                    │                     │
  │  GET /pages/.../data                 │                     │
  │────────────────▶│   GET listOrders   │                     │
  │                 │───────────────────▶│                     │
  │                 │   order list       │                     │
  │                 │◀───────────────────│                     │
  │  DataResponse   │                    │                     │
  │◀────────────────│                    │                     │
  │                 │                    │                     │
  │  POST /workflows/orders.approval/start                    │
  │────────────────▶│                    │                     │
  │                 │  (create instance) │                     │
  │  WorkflowDesc   │                    │                     │
  │◀────────────────│                    │                     │
  │                 │                    │                     │
  │  POST /workflows/wf-001/advance                           │
  │   event:approved│                    │                     │
  │────────────────▶│                    │                     │
  │                 │  POST confirmOrder │                     │
  │                 │───────────────────▶│                     │
  │                 │  200 OK            │                     │
  │                 │◀───────────────────│                     │
  │                 │                    │                     │
  │                 │  SDK: SendOrderApproved                  │
  │                 │─────────────────────────────────────────▶│
  │                 │  OK                                      │
  │                 │◀─────────────────────────────────────────│
  │                 │                    │                     │
  │  WorkflowDesc   │                    │                     │
  │  (completed)    │                    │                     │
  │◀────────────────│                    │                     │
```

---

## 23. Deployment and Operational Model

### 23.1 Startup Sequence

```
1.  Load configuration (config.yaml + environment overrides)
2.  Initialize telemetry (tracing, metrics, logging)
3.  Load OpenAPI specifications → build OpenAPIIndex
4.  Load UI definitions → validate against OpenAPIIndex → build DefinitionRegistry
5.  Initialize CapabilityResolver (connect to policy engine / load static policies)
6.  Initialize WorkflowStore (connect to PostgreSQL / initialize in-memory)
7.  Register SDK handlers in SDKInvokerRegistry
8.  Build InvokerRegistry (OpenAPI invoker + SDK invoker)
9.  Build providers (Menu, Page, Form, Command, Workflow, Search)
10. Build HTTP router with middleware chain
11. Start background tasks (workflow timeout processor, definition watcher)
12. Start HTTP server
13. Report healthy
```

If step 3 or 4 fails, the process exits with a non-zero status code.
This ensures that a misconfigured BFF never serves traffic.

### 23.2 Graceful Shutdown

```
1.  Receive SIGTERM/SIGINT
2.  Stop accepting new HTTP connections
3.  Wait for in-flight requests to complete (with 30s deadline)
4.  Stop background tasks
5.  Close WorkflowStore connections
6.  Flush telemetry
7.  Exit 0
```

### 23.3 Configuration Hierarchy

```
Priority (highest to lowest):
  1. Environment variables (THESA_*)
  2. Config file specified by --config flag
  3. config/config.yaml (default)
  4. Compiled defaults
```

### 23.4 Health and Readiness

```
GET /ui/health          → { "status": "ok" }     (liveness: process is running)
GET /ui/ready           → { "status": "ready" }   (readiness: all dependencies connected)
```

Readiness checks:
- Definition registry loaded (non-empty)
- OpenAPI index loaded (non-empty)
- WorkflowStore connected (if configured)
- Policy engine reachable (if external)

### 23.5 Definition Deployment Workflow

```
Domain team workflow:
  1. Author/modify definition YAML in their repo
  2. CI validates definition against OpenAPI spec (using shared validation library)
  3. CI publishes definition artifact (versioned, checksummed)
  4. BFF deployment pulls latest definition artifacts
  5. BFF starts, validates definitions, begins serving

No BFF code changes. No BFF redeployment required (if hot-reload is enabled).
```

### 23.6 Multi-Instance Considerations

Multiple BFF instances can run behind a load balancer:

- **Definition registry:** All instances load the same definitions (from shared
  filesystem or artifact store). Hot-reload must be coordinated (e.g., via
  signal or watching the same source).

- **Workflow store:** PostgreSQL provides shared state. All instances read/write
  to the same database. Optimistic locking (version column) prevents conflicts.

- **Capability cache:** Each instance maintains its own cache. Cache TTL
  ensures eventual consistency. For immediate invalidation, use a pub/sub
  notification channel.

- **Idempotency store:** Must be shared (Redis or PostgreSQL). If per-instance,
  duplicate commands on different instances won't be caught.

---

## Appendix A: Glossary

| Term              | Definition                                                           |
|-------------------|----------------------------------------------------------------------|
| Definition        | A YAML file describing how backend operations are exposed to the UI  |
| Descriptor        | A resolved, capability-filtered data structure sent to the frontend  |
| Capability        | A UI-level permission string (e.g., "orders:list:view")              |
| Command           | A named, authorized mutation operation invoked via POST              |
| Workflow          | A multi-step, stateful process managed by the BFF                    |
| Operation Binding | A reference to a backend operation (OpenAPI operationId or SDK handler) |
| Input Mapping     | Rules for transforming UI input into backend request parameters       |
| Output Mapping    | Rules for transforming backend responses into UI-safe shapes         |
| Request Context   | The resolved identity, tenancy, and tracing information for a request |
| Tenant            | A top-level organizational boundary for data isolation               |
| Partition         | A workspace or environment within a tenant                           |

## Appendix B: Definition File JSON Schema

For automated validation of definition files, a JSON Schema should be
maintained alongside this document. The schema enforces:

- Required fields on every definition type
- Valid enum values for type fields (layout, field types, action types, etc.)
- Pattern validation for capability strings
- Reference integrity (form_id references, command_id references)
- Non-empty arrays where required

The schema should be versioned and published as part of the BFF's artifacts
so that domain teams can validate their definitions in CI before deployment.
