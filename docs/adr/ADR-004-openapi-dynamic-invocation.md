# ADR-004: OpenAPI Dynamic Invocation Instead of Code Generation

**Status:** Accepted

**Date:** 2025-01-15

---

## Context

The BFF needs to call backend microservices on behalf of the frontend. Each backend
service exposes an OpenAPI specification describing its HTTP endpoints, request/response
schemas, and parameter definitions.

Two approaches exist for calling these services:

1. **Code generation:** Generate typed Go client code from each OpenAPI spec at build
   time. Each generated client has methods for each operation with typed request/response
   structs.

2. **Dynamic invocation:** Parse OpenAPI specs at startup, index the operations, and
   construct HTTP requests at runtime from the spec metadata and definition bindings.

The choice affects onboarding speed, binary coupling, and the flexibility to add new
backend services without recompiling the BFF.

## Decision

The BFF dynamically constructs HTTP requests at runtime using indexed OpenAPI
specifications. The `OpenAPIOperationInvoker` reads the `IndexedOperation` (containing
method, path template, base URL, and parameter metadata from kin-openapi) and builds
the request by:

1. Substituting path parameters into the URL template.
2. Appending query parameters.
3. Setting standard headers (Authorization, tenant/partition/correlation IDs).
4. Serializing the request body as JSON.

The `OpenAPIIndex` loads and indexes specs at startup:

```go
type IndexedOperation struct {
    Method       string
    PathTemplate string
    BaseURL      string
    Operation    *openapi3.Operation  // from kin-openapi
    PathItem     *openapi3.PathItem   // from kin-openapi
}
```

At startup, every definition's `operation_id` is validated against the index
(Principle P6). Operations not referenced by any definition remain invisible
(Principle P1).

## Consequences

### Positive

- **Zero-code onboarding:** Adding a new backend service requires only providing its
  OpenAPI spec file and writing YAML definitions. No Go code changes, no
  recompilation (Principle P10).
- **Loose coupling:** The BFF binary has no compile-time knowledge of specific backend
  services. It works with any HTTP API that has a valid OpenAPI spec.
- **Spec-driven validation:** Request bodies can be validated against the OpenAPI
  schema before sending, catching errors at the BFF layer rather than waiting for
  a 400 from the backend.
- **Single invocation path:** All OpenAPI-based operations flow through the same
  invoker code, ensuring consistent error handling, circuit breaking, retry logic,
  and observability.

### Negative

- **No compile-time type safety:** Request and response bodies are `map[string]any`,
  not typed structs. Type errors (wrong field name, wrong type) are caught at
  runtime (startup validation or request time), not at compile time.
- **Runtime overhead:** Building URLs and headers at runtime has more overhead than
  calling a pre-generated method. In practice, this overhead is negligible compared
  to network latency (microseconds vs. milliseconds).
- **Spec quality dependency:** The invoker is only as good as the OpenAPI specs it
  consumes. Incomplete or incorrect specs lead to incorrect requests. Mitigated by
  startup validation and structured logging of request/response details.

## Alternatives Considered

### Code Generation (go-swagger, oapi-codegen)

Generate typed Go clients from each backend's OpenAPI spec at build time.

**Rejected because:**
- Every backend API change (new endpoint, renamed field, changed parameter) requires
  regenerating the client and recompiling the BFF.
- The BFF binary becomes coupled to specific backend API versions.
- Onboarding a new backend service requires a code change (adding the generated
  client), violating Principle P10.
- Generated code may not align with the BFF's invocation patterns (e.g., generated
  clients don't know about the BFF's circuit breaker, retry, or header propagation
  requirements).

### Static HTTP Clients (Manual Go Code)

Write a custom Go HTTP client for each backend service, with hand-coded URL
construction and response parsing.

**Rejected because:** This is the most labor-intensive approach with no advantages
over code generation. Every backend change requires manual code updates.

### SDK Handlers (Typed Go Functions)

Register typed Go handler functions for each backend operation, bypassing HTTP
entirely for some services.

**Not rejected, but scoped:** SDK handlers are the designated escape hatch for
operations that cannot be expressed through OpenAPI invocation — streaming responses,
complex multi-service orchestration, high-integrity financial operations. They
coexist with the OpenAPI invoker through the `InvokerRegistry`, which dispatches
to the appropriate invoker based on the binding type (`openapi` vs. `sdk`). SDK
handlers DO require recompilation, which is the accepted trade-off for their
additional capabilities.

## References

- [Principle P1: No Implicit Exposure](../01-principles-and-invariants.md)
- [Principle P10: No Recompilation for New APIs](../01-principles-and-invariants.md)
- [Doc 13: API Mapping and Invocation](../13-api-mapping-and-invocation.md)
- [Doc 18: OperationInvoker, InvokerRegistry, OpenAPIIndex interfaces](../18-core-abstractions-and-interfaces.md)
