# 18 — Core Abstractions and Interfaces

This document catalogs every major interface in Thesa, its methods, responsibilities,
dependencies, and implementation notes. This is the reference for understanding the
system's component boundaries.

---

## Interface Catalog

### RequestContext (Value Object)

Not an interface — a struct. Included here because it is the most-referenced type.

```
RequestContext
  Fields:
    SubjectID       string
    Email           string
    TenantID        string
    PartitionID     string
    Roles           []string
    Claims          map[string]any
    SessionID       string
    DeviceID        string
    CorrelationID   string
    TraceID         string
    SpanID          string
    Locale          string
    Timezone        string

  Methods:
    Validate() → error                    // Checks mandatory fields
    HasRole(role string) → bool
    Claim(key string) → any

  Context helpers:
    WithRequestContext(ctx, *RequestContext) → context.Context
    RequestContextFrom(ctx) → *RequestContext
    MustRequestContext(ctx) → *RequestContext  // panics if absent
```

**Immutable after construction. Safe for concurrent reads.**

---

### DefinitionLoader

Loads definition files from the filesystem.

```
DefinitionLoader
  ├── LoadAll(directories []string) → ([]DomainDefinition, error)
  │     Scans directories recursively for *.yaml/*.yml files.
  │     Returns all parsed definitions.
  │     Fatal on parse errors.
  │
  ├── LoadFile(path string) → (DomainDefinition, error)
  │     Loads and parses a single file.
  │
  └── Watch(directories []string, onChange func([]DomainDefinition)) → (stop func, error)
        Watches for file changes and calls onChange with new definitions.
        Uses debouncing (2s) to batch rapid changes.
```

**Dependencies:** Filesystem, YAML parser.
**Concurrency:** LoadAll is called once at startup. Watch runs in a background goroutine.

---

### DefinitionRegistry

Read-optimized, thread-safe store of all loaded definitions.

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
  ├── Replace(definitions []DomainDefinition)
  └── Checksum() → string
```

**Dependencies:** None (pure data structure).
**Concurrency:** Lock-free reads via atomic pointer swap. Replace() is atomic.
**Implementation:** `atomic.Pointer[snapshot]` where snapshot contains all maps.

---

### CapabilityResolver

Resolves the full capability set for a request context.

```
CapabilityResolver
  ├── Resolve(RequestContext) → (CapabilitySet, error)
  │     Returns all capabilities for the given subject/tenant/partition.
  │     Uses caching.
  │
  └── Invalidate(subjectId string, tenantId string)
        Clears cached capabilities.
```

**Dependencies:** PolicyEvaluator, cache (in-memory or Redis).
**Concurrency:** Must be thread-safe. Cache uses sync.Map or similar.

---

### PolicyEvaluator

The backend implementation that actually resolves capabilities.

```
PolicyEvaluator
  ├── ResolveCapabilities(RequestContext) → (CapabilitySet, error)
  │     Resolves all capabilities from roles, attributes, tenant config.
  │
  ├── Evaluate(RequestContext, capability string, resource map[string]any) → (bool, error)
  │     Fine-grained evaluation for a specific capability + resource.
  │
  ├── EvaluateAll(RequestContext, capabilities []string, resource map[string]any) → (map[string]bool, error)
  │     Batch evaluation.
  │
  └── Sync() → error
        Refreshes policy data from external source.
```

**Dependencies:** External policy engine (OPA) or static configuration.
**Implementations:** StaticPolicyEvaluator (config file), OPAPolicyEvaluator (HTTP to OPA).

---

### MenuProvider

Builds the navigation tree from definitions filtered by capabilities.

```
MenuProvider
  └── GetMenu(ctx context.Context, rctx RequestContext, caps CapabilitySet) → (NavigationTree, error)
        Iterates all domains, filters by capabilities, sorts by order.
```

**Dependencies:** DefinitionRegistry, InvokerRegistry (for badge counts).

---

### PageProvider

Resolves page descriptors and fetches page data.

```
PageProvider
  ├── GetPage(ctx, rctx, caps, pageId string) → (PageDescriptor, error)
  │     Resolves and filters the page descriptor.
  │
  └── GetPageData(ctx, rctx, caps, pageId string, params DataParams) → (DataResponse, error)
        Fetches data from backend, applies mapping, returns standardized response.
```

**Dependencies:** DefinitionRegistry, CapabilityResolver (for field-level filtering), InvokerRegistry.

---

### FormProvider

Resolves form descriptors and fetches pre-populated data.

```
FormProvider
  ├── GetForm(ctx, rctx, caps, formId string) → (FormDescriptor, error)
  │     Resolves and filters the form descriptor.
  │
  └── GetFormData(ctx, rctx, caps, formId string, params map[string]string) → (map[string]any, error)
        Fetches existing data to pre-populate the form.
```

**Dependencies:** DefinitionRegistry, CapabilityResolver, InvokerRegistry.

---

### ActionProvider

Resolves action descriptors from definitions filtered by capabilities.

```
ActionProvider
  └── ResolveActions(ctx, rctx, caps, actions []ActionDefinition, resourceData map[string]any) → []ActionDescriptor
        For each action:
          1. Check capabilities → omit if unauthorized.
          2. Evaluate static conditions (not data-dependent) → set enabled/visible.
          3. Pass through data-dependent conditions for client-side evaluation.
          4. Strip internal fields.
          5. Return ActionDescriptor.
```

**Dependencies:** CapabilityResolver.

---

### CommandExecutor

Executes commands through the full pipeline.

```
CommandExecutor
  ├── Execute(ctx, rctx, commandId string, input CommandInput) → (CommandResponse, error)
  │     Full pipeline: resolve → authorize → map → validate → invoke → translate.
  │
  └── Validate(commandId string, input CommandInput) → []FieldError
        Dry-run validation without execution.
```

**Dependencies:** DefinitionRegistry, CapabilityResolver, InvokerRegistry, OpenAPIIndex.

---

### WorkflowEngine

Manages workflow instance lifecycle.

```
WorkflowEngine
  ├── Start(ctx, rctx, workflowId string, input map[string]any) → (WorkflowInstance, error)
  ├── Advance(ctx, rctx, instanceId string, event string, input map[string]any) → (WorkflowInstance, error)
  ├── Get(ctx, rctx, instanceId string) → (WorkflowDescriptor, error)
  ├── Cancel(ctx, rctx, instanceId string, reason string) → error
  ├── List(ctx, rctx, filters WorkflowFilters) → ([]WorkflowSummary, int, error)
  └── ProcessTimeouts(ctx) → error
```

**Dependencies:** DefinitionRegistry, CapabilityResolver, InvokerRegistry, WorkflowStore.

---

### WorkflowStore

Persists workflow instances and events.

```
WorkflowStore
  ├── Create(ctx, instance WorkflowInstance) → error
  ├── Get(ctx, tenantId string, instanceId string) → (WorkflowInstance, error)
  ├── Update(ctx, instance WorkflowInstance) → error
  ├── AppendEvent(ctx, event WorkflowEvent) → error
  ├── GetEvents(ctx, tenantId string, instanceId string) → ([]WorkflowEvent, error)
  ├── FindActive(ctx, tenantId string, filters map[string]string) → ([]WorkflowInstance, error)
  ├── FindExpired(ctx, cutoff time.Time) → ([]WorkflowInstance, error)
  └── Delete(ctx, tenantId string, instanceId string) → error
```

**Dependencies:** PostgreSQL (production) or none (in-memory for testing).
**Implementations:** PostgresWorkflowStore, MemoryWorkflowStore.

---

### SearchProvider

Aggregates search results across domains.

```
SearchProvider
  └── Search(ctx, rctx, caps, query string, pagination Pagination) → (SearchResponse, error)
        Calls all eligible search providers in parallel.
        Merges, ranks, deduplicates, paginates results.
```

**Dependencies:** DefinitionRegistry, CapabilityResolver, InvokerRegistry.

---

### OperationInvoker

Unified interface for backend invocation.

```
OperationInvoker
  ├── Invoke(ctx, rctx, binding OperationBinding, input InvocationInput) → (InvocationResult, error)
  └── Supports(binding OperationBinding) → bool
```

**Implementations:** OpenAPIOperationInvoker, SDKOperationInvoker.

---

### InvokerRegistry

Holds all invoker implementations and dispatches to the appropriate one.

```
InvokerRegistry
  ├── Register(invoker OperationInvoker)
  └── Invoke(ctx, rctx, binding OperationBinding, input InvocationInput) → (InvocationResult, error)
        Iterates registered invokers, finds one where Supports(binding) is true.
```

---

### OpenAPIIndex

Indexes OpenAPI specs for fast operation lookup.

```
OpenAPIIndex
  ├── Load(specs []SpecSource) → error
  ├── GetOperation(serviceId, operationId string) → (IndexedOperation, bool)
  ├── ValidateRequest(serviceId, operationId string, body any) → []ValidationError
  └── AllOperationIDs(serviceId string) → []string
```

**Dependencies:** OpenAPI parsing library (kin-openapi).

---

## Dependency Graph

```
                              Transport (HTTP Handlers + Middleware)
                              ┌────────────┼─────────────┐
                              │            │             │
                   ┌──────────▼───┐  ┌─────▼──────┐  ┌──▼──────────┐
                   │ MenuProvider │  │PageProvider │  │FormProvider │
                   └──────┬───────┘  └──┬────┬────┘  └──┬─────┬───┘
                          │             │    │           │     │
                   ┌──────▼─────────────▼────│───────────▼─────│─────────┐
                   │              DefinitionRegistry                      │
                   └─────────────────────────────────────────────────────┘
                          │             │    │           │     │
                   ┌──────▼──┐   ┌──────▼────▼───────────▼─────▼──┐
                   │Capability│   │        InvokerRegistry         │
                   │Resolver  │   │   ┌──────────┐ ┌──────────┐   │
                   └────┬─────┘   │   │ OpenAPI  │ │   SDK    │   │
                        │         │   │ Invoker  │ │ Invoker  │   │
                   ┌────▼─────┐   │   └────┬─────┘ └────┬─────┘   │
                   │  Policy  │   └────────┼─────────────┼─────────┘
                   │Evaluator │            │             │
                   └──────────┘   ┌────────▼─────┐   ┌──▼──────────┐
                                  │ OpenAPIIndex │   │SDKHandler   │
                                  └──────────────┘   │Registry     │
                                                     └─────────────┘

        CommandExecutor ──→ DefinitionRegistry + CapabilityResolver + InvokerRegistry + OpenAPIIndex
        WorkflowEngine  ──→ DefinitionRegistry + CapabilityResolver + InvokerRegistry + WorkflowStore
        SearchProvider  ──→ DefinitionRegistry + CapabilityResolver + InvokerRegistry
```

**Key guarantee: No circular dependencies.** All arrows point downward in the
dependency hierarchy.

---

## Type Summary

### Input/Output Types

```
CommandInput
  ├── Input           map[string]any      // User-provided payload
  ├── RouteParams     map[string]string   // Current route parameters
  └── IdempotencyKey  string

InvocationInput
  ├── PathParams      map[string]string   // URL path parameter values
  ├── QueryParams     map[string]string   // URL query parameter values
  ├── Headers         map[string]string   // Additional HTTP headers
  └── Body            any                 // Request body (will be JSON-serialized)

InvocationResult
  ├── StatusCode      int                 // HTTP status code
  ├── Body            any                 // Parsed JSON response body
  └── Headers         map[string]string   // Response headers

CommandResponse
  ├── Success         bool
  ├── Message         string
  ├── Result          map[string]any      // Projected response data
  └── Errors          []FieldError        // Field-level errors (on failure)

FieldError
  ├── Field           string              // UI field name
  ├── Code            string              // Error code (REQUIRED, MIN_LENGTH, etc.)
  └── Message         string              // Human-readable message

DataParams
  ├── Page            int
  ├── PageSize        int
  ├── Sort            string
  ├── SortDir         string
  ├── Filters         map[string]string
  └── Query           string              // Free-text search

Pagination
  ├── Page            int
  ├── PageSize        int
  └── Domain          string              // Optional domain filter

WorkflowFilters
  ├── Status          string
  ├── WorkflowID      string
  ├── SubjectID       string
  ├── Page            int
  └── PageSize        int

CapabilitySet = map[string]bool
```
