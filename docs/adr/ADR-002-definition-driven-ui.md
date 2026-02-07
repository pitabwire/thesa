# ADR-002: Definition-Driven UI Instead of Coded Handlers

**Status:** Accepted

**Date:** 2025-01-15

---

## Context

The BFF needs to serve UI metadata (page structures, form layouts, navigation trees,
action availability) to the Flutter frontend. This metadata determines what the
frontend renders, what actions it offers, and how it presents data.

Two fundamental approaches exist:

1. **Coded handlers:** Write Go handler functions for each page, form, and navigation
   node. Each handler constructs the descriptor programmatically based on the user's
   capabilities and the requested resource.

2. **Definition-driven:** Declare UI structure in YAML files. The BFF reads these
   definitions at startup and resolves them into descriptors at runtime by applying
   capability filtering, field mapping, and condition evaluation.

The choice affects how quickly new backend services can be exposed to the frontend,
how much code must be written and tested per domain, and how domain teams interact
with the BFF team.

## Decision

UI structure is declared in YAML definition files, not in Go handler code. Each
domain (e.g., orders, customers, inventory) has a definition file that declares:

- Pages (list, detail) with columns, sections, filters, and data bindings.
- Forms with fields, validation rules, and submit commands.
- Commands with input mapping, backend operation bindings, and output projection.
- Workflows with steps, transitions, and automation.
- Search providers and lookup sources.
- Navigation nodes with ordering and badge counts.

The BFF resolves definitions into descriptors at runtime:

```
Definition + RequestContext + CapabilitySet → Descriptor
```

This resolution happens in generic provider components (`PageProvider`, `FormProvider`,
`MenuProvider`, `ActionProvider`) that work identically for all domains.

## Consequences

### Positive

- **Domain team autonomy:** Domain teams onboard new services by writing YAML
  definitions without needing BFF team involvement or Go expertise. This eliminates
  the BFF as a deployment bottleneck.
- **Separation of concerns:** The BFF codebase contains domain-agnostic resolution
  logic. Domain-specific knowledge lives entirely in definition files that are
  versioned alongside the backend service they expose.
- **No recompilation:** Adding pages, forms, commands, or workflows requires no Go
  code changes and no BFF binary rebuild (Principle P10).
- **Auditable configuration:** Definition files are declarative, diffable, and
  reviewable. A security audit can inspect exactly which operations are exposed and
  what capabilities are required by reading YAML files.
- **Hot-reload capability:** In development, definition changes take effect without
  restarting the BFF (when hot-reload is enabled).

### Negative

- **Learning curve:** Domain teams must learn the definition schema. The schema has
  significant depth (pages, forms, commands, workflows, input mapping, output mapping,
  conditions, capabilities). Mitigated by comprehensive schema documentation
  ([doc 06](../06-definition-schema-reference.md)) and example domains
  ([doc 20](../20-example-domain-orders.md)).
- **Limited expressiveness:** The definition schema handles common patterns well
  (CRUD pages, standard forms, linear workflows) but cannot express arbitrary logic.
  Complex orchestration requires SDK handlers, which DO require Go code. This is an
  intentional escape hatch, not a limitation.
- **Debugging indirection:** When a page renders incorrectly, the developer must
  trace from the descriptor response back to the definition file, then to the backend
  response, then to the mapping rules. Structured logging with correlation IDs
  mitigates this.
- **Startup validation overhead:** All definitions are validated against OpenAPI specs
  at startup (Principle P6). Large definition sets increase startup time. Acceptable
  because startup is infrequent and correctness is paramount.

## Alternatives Considered

### Code Generation from OpenAPI

Auto-generate Go handlers from OpenAPI specifications, creating per-operation
endpoints with generated validation and mapping code.

**Rejected because:** Code generation couples the BFF binary to specific backend
APIs. Every backend API change requires regeneration and redeployment of the BFF.
It also violates Principle P1 (No Implicit Exposure) — all operations in a spec
would be generated, requiring an additional allow-list mechanism.

### Custom DSL

Design a custom domain-specific language (not YAML) for UI definitions, with
purpose-built syntax for pages, forms, and workflows.

**Rejected because:** A custom DSL requires a custom parser, custom tooling (editor
support, linting, formatting), and custom documentation. YAML is a widely understood
format with mature tooling (schema validation, IDE support, diff tools). The
additional expressiveness of a custom DSL does not justify the ecosystem cost.

### JSON Configuration

Use JSON instead of YAML for definition files.

**Rejected because:** JSON lacks comments, is more verbose for nested structures, and
is harder to read for configuration files. YAML's comment support is valuable for
documenting capability requirements and mapping decisions inline.

## References

- [Principle P1: No Implicit Exposure](../01-principles-and-invariants.md)
- [Principle P2: Frontend Receives Descriptors, Not Definitions](../01-principles-and-invariants.md)
- [Principle P10: No Recompilation for New APIs](../01-principles-and-invariants.md)
- [Doc 06: Definition Schema Reference](../06-definition-schema-reference.md)
