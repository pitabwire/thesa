# 00 — Architectural Overview

## What Is Thesa?

Thesa is a **metadata-driven Backend-For-Frontend (BFF)** written in Go. It sits between
a client application (Flutter, React, or any other UI) and an ecosystem of backend domain
services. It is the **sole backend entry point** for the frontend — the frontend never
communicates with domain services directly.

Thesa dynamically assembles UI metadata — menus, pages, tables, forms, actions, workflows —
by combining three inputs at runtime:

1. **UI Exposure Definitions** — Declarative YAML files authored by domain teams describing
   which backend operations are available and how they map to UI constructs.
2. **Capabilities** — The resolved set of permissions for the current user, tenant, and
   partition, evaluated against a policy engine.
3. **OpenAPI Specifications** — The machine-readable contracts of backend services, used to
   validate definitions at startup and dynamically invoke operations at runtime.

The frontend receives **descriptors** — fully resolved, capability-filtered, UI-safe data
structures that describe exactly what to render and what actions are available. The frontend
renders these descriptors using a metadata-driven rendering engine. The frontend never needs
to know about backend URLs, operation IDs, or internal data schemas.

---

## Why Does Thesa Exist?

### The Problem

In a multi-tenant enterprise platform with many backend services and domain teams:

- **Frontend teams** need a stable, UI-oriented API that doesn't break when backend teams
  refactor or evolve their services.
- **Backend teams** need the freedom to evolve APIs without coordinating every change with
  the frontend team.
- **Security teams** need a single chokepoint where authentication, authorization, tenant
  isolation, and audit logging are enforced uniformly.
- **Platform teams** need to onboard new domains and services without rewriting or redeploying
  a central orchestration layer.
- **Compliance teams** need every mutation to be traceable, authorized, and idempotent for
  financial and operational integrity.

Without a BFF like Thesa, organizations face:

- Tight coupling between frontend and backend contracts, requiring synchronized deployments.
- Inconsistent authorization enforcement — some UIs check permissions client-side, some don't.
- Frontend applications that embed knowledge of dozens of backend service URLs and schemas.
- Inability to add new services or change existing ones without frontend code changes.
- No centralized place to enforce tenant isolation, rate limiting, or circuit breaking.

### The Solution

Thesa solves these problems by:

- **Decoupling frontend and backend contracts** through an adapter layer (definitions).
- **Centralizing authorization** through a capability-based permission system.
- **Centralizing observability** through uniform logging, tracing, and metrics.
- **Enabling zero-code-change service onboarding** through declarative definitions and
  OpenAPI-driven dynamic invocation.
- **Providing workflow orchestration** for multi-step processes without embedding business
  logic in the frontend.
- **Enforcing tenant isolation structurally** — tenant ID comes from the verified token,
  never from user input.

---

## What Thesa Is NOT

Thesa is explicitly not:

- **A general-purpose API gateway.** It serves only the frontend and exposes only UI-oriented
  contracts. It is not a replacement for Kong, Envoy, or similar infrastructure.
- **A domain service.** It does not contain business logic. It does not validate whether an
  order total is correct or whether inventory is sufficient. Those rules belong in backend
  services.
- **A data store.** It does not own domain data. The only state it manages is workflow
  instances and ephemeral caches. All domain data lives in backend services.
- **An API auto-discovery system.** Loading a new OpenAPI spec does NOT automatically expose
  its operations. Every exposed operation must appear in an explicit definition.
- **A code generator.** It does not generate client SDKs or server stubs. It invokes backend
  APIs dynamically at runtime using indexed OpenAPI specifications.

---

## High-Level Architecture

```
┌────────────────┐
│   Flutter UI   │
│  (or React)    │
└───────┬────────┘
        │ HTTPS / JSON
        │ Fixed BFF endpoints only
        ▼
┌──────────────────────────────────────────────────────────────────────┐
│                          T H E S A  (BFF)                            │
│                                                                      │
│   Inbound                                                            │
│   ┌──────────────────────────────────────────────────────────────┐   │
│   │  Transport Layer (HTTP)                                      │   │
│   │  ┌───────┐ ┌───────┐ ┌────────┐ ┌──────────┐ ┌──────────┐  │   │
│   │  │ Auth  │→│Context│→│Capabil.│→│ Handlers │→│ Response │  │   │
│   │  │ Mware │ │ Mware │ │ Mware  │ │          │ │ Renderer │  │   │
│   │  └───────┘ └───────┘ └────────┘ └──────────┘ └──────────┘  │   │
│   └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│   Core Engine                                                        │
│   ┌────────────────┐  ┌────────────────┐  ┌───────────────────┐     │
│   │  Definition    │  │  Capability    │  │  Metadata         │     │
│   │  Registry      │  │  Resolver      │  │  Providers        │     │
│   │  (from YAML)   │  │  (from policy) │  │  (Menu,Page,Form) │     │
│   └────────────────┘  └────────────────┘  └───────────────────┘     │
│                                                                      │
│   ┌────────────────┐  ┌────────────────┐  ┌───────────────────┐     │
│   │  Command       │  │  Workflow      │  │  Search           │     │
│   │  Executor      │  │  Engine        │  │  Provider         │     │
│   └────────────────┘  └────────────────┘  └───────────────────┘     │
│                                                                      │
│   Outbound                                                           │
│   ┌──────────────────────────────────────────────────────────────┐   │
│   │  Invocation Layer                                            │   │
│   │  ┌─────────────────────┐  ┌──────────────────────────────┐  │   │
│   │  │ OpenAPI Invoker     │  │ SDK Invoker                  │  │   │
│   │  │ (dynamic HTTP)      │  │ (typed Go clients)           │  │   │
│   │  │                     │  │                              │  │   │
│   │  │ Uses: kin-openapi   │  │ Uses: Connect RPC, gRPC,    │  │   │
│   │  │ to build requests   │  │ native SDKs                  │  │   │
│   │  └─────────┬───────────┘  └─────────────┬────────────────┘  │   │
│   └────────────┼────────────────────────────┼────────────────────┘   │
└────────────────┼────────────────────────────┼────────────────────────┘
                 │                            │
                 ▼                            ▼
    ┌────────────────────┐      ┌────────────────────┐
    │  Orders Service    │      │  Ledger Service    │
    │  (REST / OpenAPI)  │      │  (Connect RPC)     │
    └────────────────────┘      └────────────────────┘
    ┌────────────────────┐      ┌────────────────────┐
    │  Inventory Service │      │  Notification Svc  │
    │  (REST / OpenAPI)  │      │  (gRPC / SDK)      │
    └────────────────────┘      └────────────────────┘
```

---

## Core Data Flows

### Flow 1: Metadata Request (e.g., "render a page")

```
1. Flutter requests page descriptor: GET /ui/pages/orders.list
2. Transport middleware: verify JWT, build RequestContext, resolve capabilities
3. PageProvider:
   a. Look up PageDefinition "orders.list" in DefinitionRegistry
   b. Verify user has required capabilities
   c. Filter table columns by column-level capabilities
   d. Filter actions by action-level capabilities
   e. Evaluate action conditions (is the "cancel" button relevant?)
   f. Replace operation references with BFF data endpoint URLs
   g. Resolve lookup references to option lists or endpoint URLs
   h. Return PageDescriptor (fully resolved, safe for frontend)
4. Transport: serialize to JSON, return to Flutter
```

### Flow 2: Data Request (e.g., "fetch table data")

```
1. Flutter requests table data: GET /ui/pages/orders.list/data?page=2&sort=created_at
2. Transport middleware: verify JWT, build RequestContext, resolve capabilities
3. PageProvider:
   a. Look up PageDefinition to find DataSourceDefinition
   b. Verify user has required capabilities
   c. Apply InputMapping: translate BFF query params to backend query params
   d. Call InvokerRegistry.Invoke() with the OperationBinding
   e. OpenAPI Invoker: build HTTP request from indexed OpenAPI operation
   f. Execute HTTP request to backend service (with circuit breaker, timeout)
   g. Apply ResponseMapping: extract items, total, rename fields
   h. Return DataResponse
4. Transport: serialize to JSON, return to Flutter
```

### Flow 3: Command Execution (e.g., "update an order")

```
1. Flutter sends command: POST /ui/commands/orders.update { input: {...} }
2. Transport middleware: verify JWT, build RequestContext, resolve capabilities
3. CommandExecutor:
   a. Look up CommandDefinition "orders.update" in DefinitionRegistry
   b. Verify user has required capabilities
   c. Check idempotency key (if configured)
   d. Check rate limit (if configured)
   e. Apply InputMapping: build backend request from UI input
   f. Validate constructed request body against OpenAPI schema
   g. Call InvokerRegistry.Invoke()
   h. On success: apply OutputMapping, return CommandResponse
   i. On error: translate backend error via error_map, return error envelope
4. Transport: serialize to JSON, return to Flutter
```

### Flow 4: Workflow (e.g., "approve an order")

```
1. Flutter starts workflow: POST /ui/workflows/orders.approval/start { order_id: "..." }
2. WorkflowEngine:
   a. Create WorkflowInstance, persist to WorkflowStore
   b. Enter initial step
   c. Return WorkflowDescriptor
3. Flutter advances workflow: POST /ui/workflows/{instanceId}/advance { event: "approved" }
4. WorkflowEngine:
   a. Load instance from store
   b. Verify tenant isolation, status, capabilities
   c. Find matching transition
   d. Execute system steps automatically (invoke backend operations)
   e. Persist updated state
   f. Return updated WorkflowDescriptor
```

---

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Single entry point | BFF is the only backend the UI talks to | Centralizes auth, tenant isolation, observability |
| Declarative definitions | YAML files, not code | Domain teams contribute without modifying BFF binary |
| Dynamic OpenAPI invocation | Runtime HTTP request construction | No recompilation for new APIs |
| Typed SDK escape hatch | Compiled Go handlers for specific cases | Streaming, complex orchestration, type safety |
| Capability-based authorization | UI permission strings, not backend permissions | Decouples UI concerns from backend security model |
| Server-driven UI | BFF sends descriptors, frontend renders them | Backend controls what the UI shows; frontend is generic |
| Workflow state in BFF | Persistent workflow instances | Multi-step processes need server-side state |
| Explicit exposure only | No auto-discovery | Security: prevents accidental exposure |

---

## Who Interacts With Thesa?

### Frontend Team
- Consumes the fixed BFF API endpoints
- Builds a metadata-driven rendering engine that interprets descriptors
- Never needs to know about backend service URLs or schemas
- Coordinates with definition authors on descriptor contract changes

### Domain Teams (e.g., Orders team, Inventory team)
- Author and maintain their domain's definition files (YAML)
- Own their OpenAPI specifications
- Register SDK handlers for specialized use cases
- Own their capability namespace

### Platform Team
- Maintains the Thesa codebase and deployment pipeline
- Manages service configuration (URLs, timeouts, circuit breaker settings)
- Manages the policy engine configuration
- Monitors observability dashboards

### Security Team
- Reviews capability definitions and policy mappings
- Audits command execution logs
- Validates tenant isolation guarantees

---

## Document Map

This documentation is organized into the following sections. Each section is a
standalone reference document — you can read them in order for a complete understanding,
or jump to any section for targeted reference.

| # | Document | What It Covers |
|---|----------|----------------|
| 00 | [Overview](00-overview.md) | This document. Architecture summary and data flows. |
| 01 | [Principles & Invariants](01-principles-and-invariants.md) | Non-negotiable rules. Read this first. |
| 02 | [System Topology](02-system-topology.md) | Tenancy model, service landscape, network topology. |
| 03 | [Transport & Invocation](03-transport-and-invocation.md) | Inbound HTTP API, outbound invocation models. |
| 04 | [Request Context & Identity](04-request-context-and-identity.md) | Authentication, context construction, propagation. |
| 05 | [UI Exposure Definitions](05-ui-exposure-definitions.md) | Definition system: loading, validation, hot-reload. |
| 06 | [Definition Schema Reference](06-definition-schema-reference.md) | Complete YAML schema for all definition types. |
| 07 | [Capabilities & Permissions](07-capabilities-and-permissions.md) | Capability model, resolution, policy evaluation. |
| 08 | [UI Descriptor Model](08-ui-descriptor-model.md) | Descriptor types, resolution process, frontend contract. |
| 09 | [Server-Driven UI APIs](09-server-driven-ui-apis.md) | All BFF HTTP endpoints, request/response formats. |
| 10 | [Command & Action Model](10-command-and-action-model.md) | Command pipeline, input mapping, output mapping. |
| 11 | [Workflow Engine](11-workflow-engine.md) | Workflow lifecycle, step types, state management. |
| 12 | [Global Search](12-global-search.md) | Search aggregation, ranking, pagination. |
| 13 | [API Mapping & Invocation](13-api-mapping-and-invocation.md) | OpenAPI index, dynamic invocation, SDK handlers. |
| 14 | [Schema & Contract Stability](14-schema-and-contract-stability.md) | Adapter layer, versioning, compatibility rules. |
| 15 | [Error Handling & Validation](15-error-handling-and-validation.md) | Error envelope, validation layers, error translation. |
| 16 | [Observability & Reliability](16-observability-and-reliability.md) | Logging, tracing, metrics, circuit breakers, retries. |
| 17 | [Security Model](17-security-model.md) | Threat model, tenant isolation, definition integrity. |
| 18 | [Core Abstractions & Interfaces](18-core-abstractions-and-interfaces.md) | Complete interface catalog and dependency graph. |
| 19 | [Go Package Structure](19-go-package-structure.md) | Package layout, responsibilities, conventions. |
| 20 | [Example Domain: Orders](20-example-domain-orders.md) | Complete worked example with all definition types. |
| 21 | [Example End-to-End Flows](21-example-end-to-end-flows.md) | Step-by-step request flows for common scenarios. |
| 22 | [Deployment & Operations](22-deployment-and-operations.md) | Startup, shutdown, configuration, multi-instance. |
| 23 | [Glossary & Appendices](23-glossary-and-appendices.md) | Terminology, JSON Schema reference, FAQ. |
