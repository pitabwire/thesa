# ADR-006: PostgreSQL for Workflow Persistence

**Status:** Accepted

**Date:** 2025-01-15

---

## Context

The Thesa workflow engine manages multi-step business processes (approvals, reviews,
onboarding) that can span hours or days and involve multiple users. Workflow instances
must survive:

- BFF process restarts (deployments, crashes).
- User session changes (logout, device switch, token refresh).
- Load balancer routing changes (request hits a different BFF instance).

The workflow store must support:

- **ACID transactions:** State transitions must be atomic — a workflow cannot be left
  in an inconsistent state if the process crashes mid-transition.
- **Optimistic locking:** Concurrent advances to the same workflow instance must be
  detected and rejected to prevent state corruption.
- **Tenant isolation:** Every query must be scoped by tenant ID to prevent cross-tenant
  data access.
- **Append-only audit trail:** Workflow events must be immutable — no updates or
  deletes — to serve as a compliance-grade audit trail.
- **Query flexibility:** List active workflows with filters (status, workflow ID,
  subject ID) and pagination.
- **Timeout detection:** Find workflows with expired step timeouts for automatic
  processing.

## Decision

Workflow instances and events are persisted in PostgreSQL with the following schema:

```sql
CREATE TABLE workflow_instances (
    id            UUID PRIMARY KEY,
    tenant_id     VARCHAR(255) NOT NULL,
    workflow_id   VARCHAR(255) NOT NULL,
    status        VARCHAR(50) NOT NULL,
    current_step  VARCHAR(255) NOT NULL,
    state         JSONB NOT NULL DEFAULT '{}',
    subject_id    VARCHAR(255) NOT NULL,
    partition_id  VARCHAR(255) NOT NULL,
    version       INTEGER NOT NULL DEFAULT 1,
    timeout_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE workflow_events (
    id            UUID PRIMARY KEY,
    tenant_id     VARCHAR(255) NOT NULL,
    instance_id   UUID NOT NULL REFERENCES workflow_instances(id),
    step_id       VARCHAR(255) NOT NULL,
    event         VARCHAR(255) NOT NULL,
    actor_id      VARCHAR(255) NOT NULL,
    data          JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Key design choices:

- **Optimistic locking:** The `version` column is incremented on every update.
  `UPDATE ... WHERE id = $1 AND tenant_id = $2 AND version = $3` returns 0 rows
  affected if another request advanced the workflow first, which the engine detects
  and returns as an error.
- **Tenant-scoped queries:** Every query includes `tenant_id` in the WHERE clause.
  There is no method to query across tenants.
- **Append-only events:** The `workflow_events` table has INSERT-only access. No
  UPDATE or DELETE operations exist in the `WorkflowStore` interface (only
  `AppendEvent`, no update/delete methods for events).
- **JSONB state:** The `state` column stores accumulated workflow state as JSONB,
  allowing flexible step-to-step data passing without schema migrations.

The `WorkflowStore` interface has two implementations:
- `PostgresWorkflowStore` for production.
- `MemoryWorkflowStore` for testing (fast but not durable).

## Consequences

### Positive

- **ACID guarantees:** PostgreSQL transactions ensure that state transitions are
  atomic. A crash between updating the instance and appending the event is impossible
  (both happen in the same transaction).
- **Proven reliability:** PostgreSQL is battle-tested for transactional workloads.
  Operations teams have existing expertise, monitoring, and backup procedures.
- **Rich querying:** SQL supports the workflow engine's query needs — filtering by
  status, workflow ID, subject, and timeout expiry — without additional indexing
  infrastructure.
- **Optimistic locking built-in:** The `version` column + conditional UPDATE pattern
  is a well-understood PostgreSQL idiom with predictable performance.
- **Compliance-ready audit trail:** The append-only `workflow_events` table combined
  with PostgreSQL's WAL (Write-Ahead Log) provides a durable, tamper-evident record
  of all workflow activity.

### Negative

- **Operational overhead:** Requires a PostgreSQL instance with backup, monitoring,
  and failover configuration. Adds infrastructure complexity compared to embedded
  solutions.
- **Connection management:** The BFF must manage a connection pool (configured via
  `max_open_conns`, `max_idle_conns`, `conn_max_lifetime`). Misconfiguration can
  lead to connection exhaustion under load.
- **Latency:** PostgreSQL adds network round-trip latency compared to in-process
  storage. Acceptable because workflow operations are already network-bound (they
  invoke backend services via HTTP).
- **Schema migrations:** Future changes to the workflow schema require migration
  scripts and careful rollout. Mitigated by the JSONB `state` column, which
  absorbs most data model changes without schema migration.

## Alternatives Considered

### Event Sourcing (Dedicated Event Store)

Store workflow state as a sequence of events. Reconstruct current state by replaying
events.

**Rejected because:**
- Event sourcing adds significant complexity (event versioning, snapshot optimization,
  projection rebuilds) without proportional benefit for this use case.
- The workflow engine already records events in `workflow_events` for audit purposes.
  The current state is maintained separately in `workflow_instances` for efficient
  reads, which is simpler than replaying events.
- Most workflow queries need current state (list active workflows, get current step),
  not historical event streams. Event sourcing optimizes for the less common query
  pattern.

### Redis

Store workflow state in Redis with Lua scripts for atomic transitions.

**Rejected because:**
- Redis is primarily an in-memory store. Workflow instances that span days require
  durable persistence, which Redis provides via AOF/RDB but with weaker guarantees
  than PostgreSQL's WAL.
- Redis's query capabilities are limited compared to SQL. Filtering active workflows
  by status, workflow ID, and subject would require secondary indexes or sorted sets,
  adding complexity.
- Redis Lua scripts can implement optimistic locking, but the implementation is more
  complex and less well-understood than PostgreSQL's conditional UPDATE pattern.
- Redis is better suited for the idempotency store (short-lived, key-value, high
  throughput) than for durable workflow state.

### Embedded Database (SQLite, BoltDB)

Use an embedded database that runs within the BFF process.

**Rejected because:**
- Embedded databases are single-process. In a multi-instance deployment (standard
  for production BFFs behind a load balancer), each instance would have its own
  database, and a workflow started on instance A could not be resumed on instance B.
- Horizontal scaling would require an external synchronization mechanism, negating
  the simplicity advantage of an embedded database.

## References

- [Principle P8: Workflows Are Persistent and Resumable](../01-principles-and-invariants.md)
- [Doc 11: Workflow Engine](../11-workflow-engine.md)
- [Doc 18: WorkflowStore interface](../18-core-abstractions-and-interfaces.md)
