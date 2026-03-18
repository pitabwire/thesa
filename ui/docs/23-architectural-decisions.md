# 23. Architectural Decision Records (ADRs)

## What Is an ADR?

An Architectural Decision Record (ADR) documents a significant technical decision — what was decided, why, what alternatives were considered, and what the consequences are. ADRs serve as institutional memory: when a new developer asks "why do we do it this way?", the ADR has the answer.

---

## ADR-001: Riverpod Over Bloc for State Management

### Status: Accepted

### Context

Thesa UI needs a state management solution that supports:
- An unbounded number of dynamic pages (each with its own state)
- Cache-first data lifecycle with background refresh
- Compile-time safe dependency injection
- Automatic cleanup when pages are navigated away from

### Decision

Use Riverpod 3.0 with code generation.

### Rationale

1. **Family providers**: A single `pageProvider(pageId)` definition creates unlimited instances. Bloc requires manually creating, registering, and disposing a Bloc per page.
2. **Built-in cache lifecycle**: `keepAlive`, `invalidateSelf`, and auto-dispose map directly onto the cache-first pattern. Bloc has no equivalent — these must be built from scratch.
3. **Code generation**: `@riverpod` annotation generates boilerplate. Bloc requires separate Event and State classes per feature.
4. **Stream integration**: Riverpod's `StreamProvider` integrates naturally with Drift's `watch()` streams. Bloc requires wrapping streams in events.

### Consequences

- Team must learn Riverpod's mental model (providers, ref, family, keepAlive)
- Code generation adds a build step (`build_runner`)
- Provider-based architecture differs from the more event-driven Bloc pattern

### Alternatives Considered

| Alternative | Reason Rejected |
|---|---|
| Bloc | Too much boilerplate for dynamic, unbounded page count |
| GetX | Not sufficiently typed; less community trust for enterprise use |
| MobX | Weaker Flutter integration; less ecosystem support |
| Provider (raw) | Too low-level for complex async + caching scenarios |

---

## ADR-002: Drift (SQLite) Over Hive for Local Cache

### Status: Accepted

### Context

The offline-first cache needs to store relational data (pages reference schemas, schemas are shared across pages, navigation items form a tree). It also needs reactive streams that notify the UI when data changes.

### Decision

Use Drift (SQLite) for all local persistence.

### Rationale

1. **Relational queries**: "Find all pages that reference schema X" is a simple SQL join. In Hive, this requires iterating all entries.
2. **Reactive `watch()`**: Drift's `watch()` method on any query returns a `Stream` that emits whenever the underlying data changes. This integrates naturally with Riverpod's `StreamProvider`.
3. **Schema migrations**: Drift's built-in `stepByStep` migration system handles database schema evolution across app versions.
4. **Complex invalidation**: "Mark all pages stale where schema version changed" is one SQL UPDATE. In Hive, this requires reading all entries, checking each one, and updating individually.
5. **Cross-platform**: Drift works on web (via sql.js WASM), mobile, and desktop.

### Consequences

- SQL knowledge is required (though Drift's Dart API abstracts most of it)
- Slightly higher initialization cost than Hive (SQLite file must be opened)
- Code generation required for type-safe queries

### Alternatives Considered

| Alternative | Reason Rejected |
|---|---|
| Hive | Key-value store cannot express relational cache efficiently |
| Isar | No web support; less mature |
| SharedPreferences | No structured queries; not suitable for complex cache |
| In-memory only | Data lost on app restart; defeats offline-first goal |

---

## ADR-003: Stale-While-Revalidate Caching Strategy

### Status: Accepted

### Context

The app must be fast (instant page loads) and fresh (up-to-date data). These are conflicting goals — fast means using cached data, fresh means fetching from the network.

### Decision

Use the "stale-while-revalidate" pattern: always return cached data immediately (even if stale), then refresh in the background.

### Rationale

1. **Instant perceived performance**: Users see content in <100ms on warm starts. The psychological impact of eliminating loading spinners is significant for daily-use enterprise tools.
2. **Network-independent**: If the network is slow or down, users still have full functionality with cached data.
3. **Convergent freshness**: Background refreshes ensure data converges to the latest version within minutes, without blocking the user.
4. **Simplicity**: One pattern for all data types (navigation, pages, schemas, permissions). No case-by-case decisions.

### Consequences

- Users may briefly see stale data (mitigated by the stale cache banner)
- Permission changes have a propagation delay (mitigated by 5-minute permission TTL)
- Cache invalidation logic is critical — bugs here mean stale data persists

### Alternatives Considered

| Alternative | Reason Rejected |
|---|---|
| Cache-then-network | Blocks on network response before rendering |
| Network-only | No offline support; slower page loads |
| Cache-only | Data becomes permanently stale |
| SWR with optimistic lock | Overly complex for the benefits gained |

---

## ADR-004: No Client-Side Permission Evaluation

### Status: Accepted

### Context

The app must enforce access control — users should only see and do what they are authorized to. The question: should the frontend evaluate permissions locally, or rely entirely on server-provided flags?

### Decision

The frontend NEVER evaluates permissions. It reads `allowed: true/false` flags from BFF responses.

### Rationale

1. **Single source of truth**: Permission logic exists in exactly one place (the BFF). No risk of frontend/backend disagreements.
2. **No stale permission bugs**: The frontend does not cache permission rules — it caches the BFF's pre-evaluated decisions. If the BFF says "allowed: false," that is final.
3. **No bypass risk**: Since the frontend does not contain permission logic, there is nothing to bypass.
4. **Simpler frontend**: No RBAC (Role-Based Access Control) library, no permission evaluation engine, no policy language.

### Consequences

- The BFF must include `allowed` flags on EVERY item it returns (navigation, components, actions)
- Permission changes require a BFF response refresh (mitigated by 5-minute TTL)
- The frontend cannot show "you need X permission to do this" (unless the BFF includes this in the error message)

### Alternatives Considered

| Alternative | Reason Rejected |
|---|---|
| Download permissions, evaluate locally | Duplicates security logic; risk of mismatch |
| CASL or similar client-side RBAC | Adds complexity; creates a second source of truth |
| Hybrid (server + client evaluation) | Worst of both worlds; maintenance doubled |

---

## ADR-005: Component Registry with Plugin Override

### Status: Accepted

### Context

The UI Engine must render components generically from BFF descriptors. But some domains need specialized UX that generic rendering cannot provide.

### Decision

Implement a two-tier system: a Component Registry for built-in components, and a Plugin Registry that can override any component, page, or schema renderer.

### Rationale

1. **90% generic, 10% custom**: Most enterprise screens (lists, forms, dashboards) work perfectly with generic rendering. Plugins handle the 10% that needs special treatment.
2. **No core modifications**: Domain teams add plugins without touching the platform code. This prevents merge conflicts and reduces risk.
3. **Graceful fallback**: If a plugin is not registered, the generic renderer handles it. If a component type is unknown, a placeholder is shown. Nothing crashes.
4. **Incremental customization**: Teams can start with generic rendering and add plugins only where the generic UX is insufficient.

### Consequences

- Plugin API must be stable (breaking it breaks all plugins)
- Plugin developers must understand the descriptor format and provider system
- Testing must cover both generic and plugin-overridden paths

---

## ADR-006: Drift watch() Streams as Bridge to Riverpod

### Status: Accepted

### Context

The cache-first pattern requires: "when background refresh writes new data to the cache, the UI must update automatically." This needs a mechanism to connect Drift (database) writes to Riverpod (state) updates to Flutter (widgets).

### Decision

Use Drift's `watch()` method to create reactive Streams from database queries. Riverpod providers subscribe to these streams via `StreamProvider` or manual stream listening within `AsyncNotifier`.

### Rationale

1. **Automatic propagation**: Writing to Drift automatically triggers stream updates. No manual "notify" calls needed.
2. **Single write path**: The only way to update the UI is through the database. This eliminates bugs where the UI and cache disagree.
3. **Natural integration**: Drift `Stream` → Riverpod `StreamProvider` → Flutter widget rebuild is a clean, well-supported pipeline.
4. **Debugging simplicity**: If the UI shows wrong data, check the database. If the database has wrong data, check the network write. The path is unambiguous.

### Consequences

- All data mutations must go through Drift (no "quick" in-memory-only updates)
- Drift queries must be efficient (watch() re-executes the query on every table change)
- Potential for excessive rebuilds if watch() queries are too broad (mitigated by scoping queries to specific rows)

---

## ADR-007: GoRouter with Dynamic Route Generation

### Status: Accepted

### Context

Navigation in Thesa UI is dynamic — the BFF defines what pages exist. The router must create routes from data, not from code.

### Decision

Use GoRouter with `StatefulShellRoute` and programmatic route generation from BFF navigation descriptors.

### Rationale

1. **URL-based routing**: Essential for web deployment, deep linking, and browser history
2. **StatefulShellRoute**: Preserves per-section navigation state (scrolling, pagination, form input)
3. **Programmatic generation**: Routes are generated by iterating the BFF navigation tree. Adding a new page to the BFF automatically creates a new route.
4. **Route guards**: Built-in redirect mechanism for auth and permission checks

### Consequences

- Route regeneration on navigation changes requires careful handling of the current route
- `GoRouter` does not natively support dynamic reconfiguration — the router instance must be recreated
- Deep links to dynamically-created routes work only if the route exists at navigation time
