# 19 — Go Package Structure

This document describes the Go package layout, the responsibilities of each package,
naming conventions, and dependency rules.

---

## Directory Layout

```
github.com/pitabwire/thesa/
│
├── cmd/
│   └── bff/
│       └── main.go                       # Application entry point, dependency wiring
│
├── model/                                # PUBLIC: shared domain types
│   ├── context.go                        # RequestContext, context helpers
│   ├── capability.go                     # Capability, CapabilitySet, resolver/evaluator interfaces
│   ├── definition.go                     # All definition types
│   ├── descriptor.go                     # All descriptor types
│   ├── workflow.go                       # WorkflowInstance, WorkflowEvent
│   ├── invoker.go                        # OperationInvoker interface, InvocationInput/Result
│   └── errors.go                         # ErrorEnvelope, FieldError, error codes
│
├── internal/                             # PRIVATE: all implementations
│   ├── config/
│   │   └── config.go                     # Application configuration
│   │
│   ├── definition/
│   │   ├── loader.go                     # YAML file loading, checksumming
│   │   ├── registry.go                   # In-memory definition store (atomic pointer)
│   │   └── validator.go                  # Validation against OpenAPI + cross-references
│   │
│   ├── capability/
│   │   ├── resolver.go                   # CapabilityResolver with caching
│   │   ├── static_policy.go              # StaticPolicyEvaluator (config-file-based)
│   │   └── opa_policy.go                 # OPAPolicyEvaluator (external OPA)
│   │
│   ├── metadata/
│   │   ├── menu.go                       # MenuProvider
│   │   ├── page.go                       # PageProvider
│   │   ├── form.go                       # FormProvider
│   │   └── action.go                     # ActionProvider (shared action resolution)
│   │
│   ├── command/
│   │   └── executor.go                   # CommandExecutor
│   │
│   ├── workflow/
│   │   ├── engine.go                     # WorkflowEngine
│   │   ├── store.go                      # WorkflowStore interface
│   │   ├── pgstore.go                    # PostgreSQL implementation
│   │   └── memstore.go                   # In-memory implementation (testing)
│   │
│   ├── search/
│   │   └── provider.go                   # SearchProvider
│   │
│   ├── invoker/
│   │   ├── registry.go                   # InvokerRegistry
│   │   ├── openapi.go                    # OpenAPIOperationInvoker
│   │   └── sdk.go                        # SDKOperationInvoker + handler registry
│   │
│   ├── openapi/
│   │   └── index.go                      # OpenAPI spec loading and indexing
│   │
│   ├── transport/
│   │   ├── router.go                     # HTTP router setup (chi)
│   │   ├── middleware.go                  # Auth, context, recovery, CORS, metrics
│   │   ├── handler_navigation.go         # GET /ui/navigation
│   │   ├── handler_page.go              # GET /ui/pages/{pageId}, /data
│   │   ├── handler_form.go              # GET /ui/forms/{formId}, /data
│   │   ├── handler_command.go           # POST /ui/commands/{commandId}
│   │   ├── handler_workflow.go          # Workflow endpoints
│   │   ├── handler_search.go            # GET /ui/search
│   │   ├── handler_lookup.go            # GET /ui/lookups/{lookupId}
│   │   └── response.go                  # Response helpers, error rendering
│   │
│   └── observability/
│       └── telemetry.go                  # Tracing, metrics, logging setup
│
├── definitions/                          # Domain definition YAML files
│   ├── orders/
│   │   └── definition.yaml
│   ├── inventory/
│   │   └── definition.yaml
│   └── customers/
│       └── definition.yaml
│
├── specs/                                # OpenAPI specification files
│   ├── orders-svc.yaml
│   ├── inventory-svc.yaml
│   └── customers-svc.yaml
│
├── config/                               # Runtime configuration files
│   ├── config.yaml
│   └── config.production.yaml
│
├── go.mod
├── go.sum
├── ARCHITECTURE.md
└── docs/                                 # This documentation
```

---

## Package Responsibilities

### `model/` — Public Domain Types

**Import path:** `github.com/pitabwire/thesa/model`

Contains all shared types and interfaces that define the system's contracts.
This is the ONLY public package. Everything else is internal.

**Why public:** External code may need these types:
- SDK handler implementations in separate modules.
- Definition validation tools.
- Test utilities.
- Integration test suites.

**Contents:**
- `context.go`: RequestContext struct and context helpers.
- `capability.go`: Capability, CapabilitySet, CapabilityResolver interface, PolicyEvaluator interface.
- `definition.go`: All definition types (DomainDefinition, PageDefinition, etc.).
- `descriptor.go`: All descriptor types (PageDescriptor, FormDescriptor, etc.).
- `workflow.go`: WorkflowInstance, WorkflowEvent (runtime state types).
- `invoker.go`: OperationInvoker interface, InvocationInput, InvocationResult.
- `errors.go`: ErrorEnvelope, FieldError, standard error codes.

**Rules:**
- No dependencies on `internal/` packages.
- No dependencies on third-party libraries (pure Go + standard library only).
- Types are stable — changes here affect all consumers.

### `internal/config/` — Configuration

**Import path:** `github.com/pitabwire/thesa/internal/config`

Loads and validates application configuration from YAML files and environment
variables.

**Contents:**
- Server configuration (ports, timeouts, CORS).
- Service registry (backend service URLs, auth strategies, circuit breaker settings).
- Identity provider configuration (JWKS URL, issuer, audience, claim paths).
- Cache configuration (TTLs, max entries).
- Feature flags.

### `internal/definition/` — Definition Loading and Registry

**Import path:** `github.com/pitabwire/thesa/internal/definition`

Loads YAML definitions, validates them, and provides a fast lookup registry.

**Files:**
- `loader.go`: File scanning, YAML parsing, checksumming, file watching.
- `registry.go`: Atomic-pointer-based immutable snapshot store.
- `validator.go`: Structural and referential validation.

### `internal/capability/` — Authorization

**Import path:** `github.com/pitabwire/thesa/internal/capability`

Resolves and caches capabilities.

**Files:**
- `resolver.go`: CapabilityResolver with in-memory cache.
- `static_policy.go`: PolicyEvaluator that reads role→capability mappings from config.
- `opa_policy.go`: PolicyEvaluator that calls an OPA endpoint.

### `internal/metadata/` — Descriptor Providers

**Import path:** `github.com/pitabwire/thesa/internal/metadata`

Resolves definitions into descriptors, filtering by capabilities.

**Files:**
- `menu.go`: MenuProvider — builds NavigationTree.
- `page.go`: PageProvider — resolves PageDescriptor and fetches data.
- `form.go`: FormProvider — resolves FormDescriptor and fetches pre-populated data.
- `action.go`: ActionProvider — resolves ActionDescriptor lists (shared by page and form).

### `internal/command/` — Command Execution

**Import path:** `github.com/pitabwire/thesa/internal/command`

Implements the full command execution pipeline.

**Files:**
- `executor.go`: CommandExecutor with all pipeline stages.

### `internal/workflow/` — Workflow Engine

**Import path:** `github.com/pitabwire/thesa/internal/workflow`

Manages workflow state machine and persistence.

**Files:**
- `engine.go`: WorkflowEngine implementation.
- `store.go`: WorkflowStore interface (technically could be in model/, but kept here for cohesion).
- `pgstore.go`: PostgreSQL WorkflowStore implementation.
- `memstore.go`: In-memory WorkflowStore for testing.

### `internal/search/` — Global Search

**Import path:** `github.com/pitabwire/thesa/internal/search`

Aggregates search across domains.

### `internal/invoker/` — Backend Invocation

**Import path:** `github.com/pitabwire/thesa/internal/invoker`

Implements the invocation abstraction.

**Files:**
- `registry.go`: InvokerRegistry that dispatches to appropriate invoker.
- `openapi.go`: OpenAPIOperationInvoker — dynamic HTTP invocation.
- `sdk.go`: SDKOperationInvoker + SDKHandlerRegistry.

### `internal/openapi/` — OpenAPI Spec Management

**Import path:** `github.com/pitabwire/thesa/internal/openapi`

Loads and indexes OpenAPI specifications.

### `internal/transport/` — HTTP Layer

**Import path:** `github.com/pitabwire/thesa/internal/transport`

HTTP handlers, middleware, routing. This is the outer shell of the application.

**Files split by resource** for maintainability. Each handler file is focused
and testable independently.

### `internal/observability/` — Telemetry

**Import path:** `github.com/pitabwire/thesa/internal/observability`

Sets up structured logging, distributed tracing, and Prometheus metrics.

### `cmd/bff/` — Application Entry Point

**Import path:** `github.com/pitabwire/thesa/cmd/bff`

The `main.go` file wires all dependencies together and starts the server.
This is the composition root — the only place where concrete implementations
are instantiated and connected.

---

## Dependency Rules

### Rule 1: `model/` depends on nothing

```
model/ → (standard library only)
```

### Rule 2: `internal/*` depends on `model/` and other `internal/*` packages

```
internal/config     → model
internal/openapi    → model
internal/definition → model, internal/openapi
internal/capability → model
internal/invoker    → model, internal/openapi
internal/metadata   → model, internal/definition, internal/capability, internal/invoker
internal/command    → model, internal/definition, internal/capability, internal/invoker, internal/openapi
internal/workflow   → model, internal/definition, internal/capability, internal/invoker
internal/search     → model, internal/definition, internal/capability, internal/invoker
internal/transport  → model, internal/* (all)
internal/observability → (standard library + telemetry libraries)
```

### Rule 3: No circular dependencies

If package A imports package B, then package B must NOT import package A (directly
or transitively). This is enforced by the Go compiler.

### Rule 4: `cmd/bff/` is the composition root

Only `cmd/bff/main.go` creates concrete instances and wires them together.
Internal packages accept interfaces, not concrete types.

---

## Naming Conventions

### Package Names

- Lowercase, single-word: `model`, `definition`, `capability`, `metadata`.
- No underscores or hyphens.
- Package name should describe what it contains, not what it does.

### File Names

- snake_case: `static_policy.go`, `handler_command.go`.
- One primary type per file (but related helpers in the same file are fine).
- Test files: `*_test.go` in the same package.

### Type Names

- Interfaces: verb-noun (`CapabilityResolver`, `PolicyEvaluator`, `WorkflowStore`).
- Structs: noun (`RequestContext`, `PageDescriptor`, `WorkflowInstance`).
- No `I` prefix on interfaces (Go convention).
- No stutter: `definition.Loader` not `definition.DefinitionLoader`.

### Method Names

- Getters without "Get" prefix: `ctx.TenantID` not `ctx.GetTenantID()`.
- Constructors: `NewExecutor(...)`, `NewResolver(...)`.
- Boolean methods: `Has(...)`, `Supports(...)`, `IsActive()`.

---

## Testing Strategy

### Unit Tests

Each package has `*_test.go` files. Unit tests use mock implementations
of interfaces.

```
internal/command/executor_test.go     → tests CommandExecutor with mock invoker, mock registry
internal/workflow/engine_test.go      → tests WorkflowEngine with MemoryWorkflowStore
internal/metadata/page_test.go        → tests PageProvider with fixture definitions
```

### Integration Tests

Integration tests live in a separate `test/` directory or in `*_integration_test.go`
files (build-tagged to skip in normal `go test`).

```
test/integration/
  ├── command_test.go       → tests full command pipeline with HTTP backend mock
  ├── workflow_test.go      → tests workflow with PostgreSQL
  └── search_test.go        → tests search aggregation with multiple mock backends
```

### Fixture Definitions

Test fixtures (YAML definition files for testing) live in `testdata/` directories
within each package:

```
internal/definition/testdata/
  ├── valid_orders.yaml
  ├── invalid_missing_operation.yaml
  └── invalid_duplicate_id.yaml
```

---

## Key Dependencies

| Dependency | Purpose | Package |
|-----------|---------|---------|
| `github.com/go-chi/chi/v5` | HTTP router | `internal/transport` |
| `github.com/getkin/kin-openapi` | OpenAPI parsing and validation | `internal/openapi` |
| `github.com/google/uuid` | UUID generation | `internal/workflow`, `internal/transport` |
| `gopkg.in/yaml.v3` | YAML parsing | `internal/definition` |
| `go.uber.org/zap` | Structured logging | `internal/observability` |
| `go.opentelemetry.io/otel` | Distributed tracing | `internal/observability`, `internal/transport` |
| `github.com/prometheus/client_golang` | Prometheus metrics | `internal/observability`, `internal/transport` |
| `github.com/jackc/pgx/v5` | PostgreSQL driver | `internal/workflow` |
