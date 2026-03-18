# Thesa UI — Architecture Documentation

## Enterprise Capability-Driven UI Runtime for Flutter

Thesa UI is a generic, dynamic frontend that renders enterprise admin interfaces entirely from backend (BFF) descriptors. It contains zero hardcoded domain knowledge. Every screen, form, table, action, and workflow is built at runtime from server-provided contracts.

---

## Documentation Index

### Foundations

| # | Section | Description | Audience |
|---|---|---|---|
| 01 | [Executive Summary](./01-executive-summary.md) | What Thesa UI is, why it exists, and its core principles | Everyone |
| 02 | [Technology Choices](./02-technology-choices.md) | Every library used, why it was chosen, and how it fits | Developers, Architects |
| 03 | [Layered Architecture](./03-layered-architecture.md) | The six layers, their responsibilities, and communication rules | Developers, Architects |
| 04 | [Folder Structure](./04-folder-structure.md) | Every folder and file explained with navigation rules | Developers |

### Data & State

| # | Section | Description | Audience |
|---|---|---|---|
| 05 | [Offline-First Cache](./05-offline-first-cache.md) | Database schema, cache lifecycle, TTL, ETags, stale-while-revalidate | Developers, Architects |
| 06 | [State Architecture](./06-state-architecture.md) | Riverpod providers, dependency tree, cache-first pattern, permissions | Developers |

### Dynamic UI

| # | Section | Description | Audience |
|---|---|---|---|
| 07 | [Dynamic UI Engine](./07-dynamic-ui-engine.md) | Page rendering pipeline, component registry, BFF response examples | Developers |
| 08 | [Dynamic Form Engine](./08-dynamic-form-engine.md) | Schema-to-form pipeline, all field types, validation, conditional fields | Developers |
| 09 | [Dynamic Table Engine](./09-dynamic-table-engine.md) | Pagination, sorting, filtering, column priority, bulk actions, virtualization | Developers |
| 10 | [Workflow Engine](./10-workflow-engine.md) | State machine, step rendering, branching, persistence, polling | Developers |

### Navigation & Layout

| # | Section | Description | Audience |
|---|---|---|---|
| 11 | [Dynamic Navigation](./11-dynamic-navigation.md) | Route generation, sidebar, breadcrumbs, deep linking, route guards | Developers |
| 12 | [Responsive Layout](./12-responsive-layout.md) | Breakpoints, shell adaptation, content layout rules per screen size | Developers, Designers |

### Extensibility

| # | Section | Description | Audience |
|---|---|---|---|
| 13 | [Plugin System](./13-plugin-system.md) | Page, component, and schema plugins; resolution order; packaging | Developers, Domain Teams |

### Operations

| # | Section | Description | Audience |
|---|---|---|---|
| 14 | [Authentication & Session](./14-authentication-session.md) | Token flow, refresh, logout, session management, login screen | Developers, Security |
| 15 | [Error Handling](./15-error-handling.md) | Error categories, UX treatment, ErrorBoundary, graceful degradation | Developers, QA |
| 16 | [Telemetry & Observability](./16-telemetry-observability.md) | Instrumented events, OpenTelemetry export, key metrics | Developers, Ops |

### Quality & Performance

| # | Section | Description | Audience |
|---|---|---|---|
| 17 | [Performance Strategy](./17-performance.md) | Virtualization, debouncing, rebuild optimization, performance budgets | Developers |
| 18 | [Testing Strategy](./18-testing-strategy.md) | Unit, widget, golden tests; what to test; test organization; CI | Developers, QA |
| 19 | [Security Principles](./19-security.md) | No local permissions, absent (not disabled) elements, token security | Everyone |

### System Overview

| # | Section | Description | Audience |
|---|---|---|---|
| 20 | [Data Flow & Scalability](./20-data-flow-startup-scalability.md) | Complete data flow, startup sequences, scalability axes | Developers, Architects |
| 21 | [Networking Layer](./21-networking-layer.md) | BFF endpoints, interceptor chain, retries, cancellation, background refresh | Developers |
| 22 | [Design System](./22-design-system.md) | Color tokens, typography, spacing, dark mode, branding, accessibility | Developers, Designers |
| 23 | [Architectural Decisions](./23-architectural-decisions.md) | ADRs for all major technology and design decisions | Architects, Tech Leads |

---

## Reading Order

### If you are new to the project

Read in order: 01 → 02 → 03 → 04. This gives you the full context.
Then read the sections relevant to your work.

### If you are a frontend developer

Start with: 01, 03, 04, 06, 07. Then read whichever component you are working on (forms: 08, tables: 09, workflows: 10).

### If you are a backend / BFF developer

Start with: 01, 07, 08, 09, 10, 11. These sections show the exact JSON contracts the frontend expects.

### If you are a designer

Start with: 01, 12, 22. These cover the responsive strategy and design system.

### If you are a domain team building plugins

Start with: 01, 07, 13. Section 13 is the plugin guide.

### If you are reviewing security

Start with: 14, 19, 05 (cache security section).

### If you are an architect evaluating the system

Start with: 01, 03, 23 (ADRs), 20, 05.

---

## Key Concepts Quick Reference

| Concept | Meaning | Detailed In |
|---|---|---|
| **BFF** | Backend-For-Frontend — the single server the UI talks to | Section 01 |
| **Capability-driven** | UI only shows what the server declares available | Section 01 |
| **Schema-driven** | Forms and tables built from data structure definitions | Section 08, 09 |
| **Offline-first** | Always render from cache; refresh in background | Section 05 |
| **Stale-while-revalidate** | Return cached data immediately, even if stale; refresh behind the scenes | Section 05 |
| **Family provider** | A Riverpod provider that creates separate instances per parameter | Section 06 |
| **Component Registry** | Lookup table mapping BFF component types to Flutter widgets | Section 07 |
| **Plugin Registry** | Registry for custom overrides of pages, components, or schema renderers | Section 13 |
| **ErrorBoundary** | Widget wrapper that catches rendering errors, preventing page-wide crashes | Section 15 |
| **Design tokens** | Named values (colors, fonts, spacing) ensuring visual consistency | Section 22 |
| **ADR** | Architectural Decision Record — documents why a decision was made | Section 23 |
