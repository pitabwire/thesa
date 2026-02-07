# ADR-001: Single Command Endpoint for All Mutations

**Status:** Accepted

**Date:** 2025-01-15

---

## Context

Thesa is a Backend-For-Frontend (BFF) that mediates between a Flutter frontend and
multiple backend microservices. The BFF must provide a consistent, secure, and
observable interface for all data mutations originating from the frontend.

Traditional BFF designs expose resource-specific endpoints (e.g., `PUT /orders/{id}`,
`DELETE /orders/{id}`, `POST /payments`). Each endpoint implements its own
authorization, validation, audit logging, rate limiting, and error translation logic.
As the number of backend services and operations grows, this leads to:

- **Inconsistent enforcement:** Each handler must independently implement security
  controls. A single omission creates a vulnerability.
- **Observability gaps:** Metrics, tracing, and audit logging require per-handler
  instrumentation. Missing instrumentation in one handler means blind spots.
- **High onboarding cost:** Adding a new mutation path requires writing a new handler,
  new route registration, new tests — even when the operation is a straightforward
  CRUD proxy.

The BFF needs a uniform mechanism that applies cross-cutting concerns exactly once,
regardless of which backend service or operation is being invoked.

## Decision

All frontend-initiated mutations go through a single endpoint:

```
POST /ui/commands/{commandId}
```

The `commandId` identifies a command definition (YAML) that specifies the backend
operation, input mapping, validation rules, capability requirements, and output
mapping. The `CommandExecutor` processes every command through a uniform 10-step
pipeline:

1. Resolve command definition
2. Authorize (capability check)
3. Check rate limit
4. Check idempotency
5. Map input (frontend → backend shape)
6. Validate input (against OpenAPI schema + definition rules)
7. Invoke backend operation
8. Detect errors in response
9. Map output (backend → frontend shape)
10. Notify observers (audit, metrics)

This pipeline is implemented once in `internal/command/executor.go` and applied
universally. New mutations are added by writing YAML definition files, not Go code.

## Consequences

### Positive

- **Uniform security:** Authorization, rate limiting, and idempotency are applied to
  every mutation without exception. A new command definition automatically inherits
  all controls.
- **Uniform observability:** Every mutation produces the same structured audit event,
  the same metrics labels (`command_id`, `tenant_id`, `status_code`, `duration_ms`),
  and the same trace span hierarchy.
- **Zero-code onboarding:** Domain teams add new mutations by writing YAML definitions.
  No Go code, no route registration, no recompilation (Principle P10).
- **Consistent error handling:** All mutations return the same `CommandResponse` envelope
  with `success`, `message`, `result`, and `errors` fields.
- **Idempotency for free:** Any command can opt into idempotency by adding an
  `idempotency` block to its definition. No per-command implementation needed.

### Negative

- **No HTTP verb semantics:** All mutations are POST, losing the semantic distinction
  between create, update, and delete at the HTTP layer. This makes standard HTTP
  caching and intermediary behavior inapplicable (acceptable because the BFF returns
  `Cache-Control: no-store` for all authenticated responses).
- **Command ID leaks operation intent:** The `commandId` in the URL reveals the
  operation name (e.g., `orders.cancel`). This is acceptable because the frontend
  already knows the command ID from the descriptor, and the ID itself does not reveal
  backend implementation details.
- **Indirection cost:** Developers must look up the command definition to understand
  what a mutation does, rather than reading a self-contained handler function. Mitigated
  by clear naming conventions (`{domain}.{verb}`) and documentation.

## Alternatives Considered

### Resource-Specific Endpoints

```
PUT /ui/orders/{id}
DELETE /ui/orders/{id}
POST /ui/payments
```

**Rejected because:** Cross-cutting concerns must be implemented per-handler. With
dozens of backend services and hundreds of operations, this creates an unsustainable
maintenance burden and guarantees inconsistency. Every new domain requires new Go code.

### GraphQL Mutations

```graphql
mutation { cancelOrder(id: "ord-123") { success message } }
```

**Rejected because:** GraphQL adds schema complexity without corresponding benefit
in this architecture. The BFF already has operation definitions (YAML) that serve
the same purpose as a GraphQL schema. GraphQL's field-level resolution is unnecessary
because the BFF's output mapping already handles response projection. GraphQL also
requires a schema build step and makes it harder to apply per-mutation rate limiting
and idempotency.

### Command Bus with Typed Commands

Define each command as a Go struct implementing a `Command` interface, dispatched
through a command bus.

**Rejected because:** This requires recompilation for every new command, violating
Principle P10. The YAML-driven approach provides the same dispatch pattern without
compile-time coupling to specific commands.

## References

- [Principle P4: Commands Are the Only Mutation Path](../01-principles-and-invariants.md)
- [Doc 10: Command and Action Model](../10-command-and-action-model.md)
- [Doc 18: CommandExecutor interface](../18-core-abstractions-and-interfaces.md)
