# ADR-003: Atomic Pointer Swap for Definition Registry

**Status:** Accepted

**Date:** 2025-01-15

---

## Context

The `DefinitionRegistry` is the most frequently accessed component in the BFF.
Every request — page load, form fetch, command execution, navigation build, search,
lookup — reads from the registry to resolve definitions. In a production deployment
handling thousands of concurrent requests, the registry is on the critical path of
every single request.

The registry also needs to support atomic replacement of all definitions (for
hot-reload and startup). This creates a classic readers-writer concurrency problem:
many goroutines read concurrently, and occasionally one goroutine replaces the
entire dataset.

The choice of concurrency mechanism directly impacts request latency under load.

## Decision

The `DefinitionRegistry` uses `atomic.Pointer[snapshot]` where `snapshot` is an
immutable struct containing all pre-built lookup maps:

```go
type snapshot struct {
    domains   map[string]DomainDefinition
    pages     map[string]PageDefinition
    forms     map[string]FormDefinition
    commands  map[string]CommandDefinition
    workflows map[string]WorkflowDefinition
    searches  map[string]SearchDefinition
    lookups   map[string]LookupDefinition
    checksum  string
}
```

**Read path (hot path):**

```go
func (r *Registry) GetPage(pageId string) (PageDefinition, bool) {
    snap := r.current.Load()  // atomic load, no lock
    p, ok := snap.pages[pageId]
    return p, ok
}
```

**Write path (rare):**

```go
func (r *Registry) Replace(definitions []DomainDefinition) {
    snap := buildSnapshot(definitions)  // build new snapshot
    r.current.Store(snap)               // atomic store, replaces pointer
}
```

Reads are completely lock-free. Writers build a new snapshot and atomically swap the
pointer. In-flight reads that loaded the old pointer continue using the old snapshot
safely (Go's garbage collector reclaims it when no references remain).

## Consequences

### Positive

- **Zero read contention:** No mutex, no RWMutex, no lock of any kind on the read
  path. This is critical because the registry is read on every request.
- **No reader starvation:** With `sync.RWMutex`, a pending writer blocks new readers,
  causing latency spikes during hot-reload. With atomic pointer swap, readers are
  never blocked.
- **Simplicity:** The implementation is straightforward — `Load()` to read,
  `Store()` to write. No lock ordering concerns, no deadlock risk.
- **Consistent snapshots:** A reader that loads the pointer sees a consistent view
  of all definitions. There is no risk of seeing a partially updated registry
  (e.g., a command definition that references a form not yet loaded).

### Negative

- **Memory overhead during swap:** During a replacement, both the old and new
  snapshots exist in memory simultaneously until the garbage collector reclaims the
  old one. For typical definition sets (hundreds of definitions), this overhead is
  negligible (kilobytes to low megabytes).
- **Full rebuild on any change:** Even a single definition change requires building
  a complete new snapshot. This is acceptable because replacements are rare (startup
  and hot-reload with 2-second debounce) and building a snapshot from parsed
  definitions is fast (sub-millisecond for typical sets).
- **No incremental updates:** Cannot add or remove a single definition without
  rebuilding the entire snapshot. This is by design — atomic replacement ensures
  consistency and eliminates partial-update bugs.

## Alternatives Considered

### sync.RWMutex

Protect the registry maps with `sync.RWMutex`. Readers acquire a read lock;
the writer acquires a write lock during replacement.

**Rejected because:** `RWMutex` has measurable overhead on the read path due to
atomic counter increments/decrements for reader tracking. Under high concurrency,
this creates cache line contention across CPU cores. Additionally, a pending writer
starves new readers, causing latency spikes during hot-reload.

### Copy-on-Write with sync.Map

Use `sync.Map` for each definition type, allowing concurrent reads and writes
without explicit locking.

**Rejected because:** `sync.Map` does not provide consistent snapshots across
multiple map reads. A request that reads a command definition and then its referenced
form definition might see them from different versions if a replacement happens
between the two reads. The atomic pointer swap guarantees that all reads within a
request see the same snapshot.

### Channel-Based Updates

Send definition updates through a channel to a single goroutine that owns the maps.
Reads are served by the owning goroutine via request-response channels.

**Rejected because:** This serializes all reads through a single goroutine,
creating a severe bottleneck. Every request would need to send a message, wait
for a response, and handle the reply — orders of magnitude slower than a direct
map lookup.

## References

- [Doc 18: DefinitionRegistry interface](../18-core-abstractions-and-interfaces.md)
- Go documentation: [`sync/atomic.Pointer`](https://pkg.go.dev/sync/atomic#Pointer)
