# ADR-007: BFF Owns Pagination Standardization

**Status:** Accepted

**Date:** 2025-01-15

---

## Context

The Flutter frontend fetches paginated data from multiple backend services through
the BFF. Each backend service may use a different pagination style:

| Backend Service | Pagination Style | Parameters |
|----------------|-----------------|------------|
| Orders Service | Offset-based | `offset`, `limit` |
| Customers Service | Page-number | `page`, `size` |
| Inventory Service | Cursor-based | `cursor`, `count` |
| Legacy Service | Custom | `start_row`, `end_row` |

If the frontend had to understand each backend's pagination style, it would need
per-service pagination logic — knowing which parameters to send, how to interpret
pagination metadata in responses, and how to compute next/previous page values.
This creates tight coupling between the frontend and individual backend services.

The BFF is positioned between the frontend and backends, making it the natural place
to standardize pagination.

## Decision

The BFF exposes a single pagination interface to the frontend and translates it to
each backend's native pagination style based on per-service configuration.

**Frontend interface (standardized):**

```
Request:  GET /ui/pages/{pageId}/data?page=1&page_size=25&sort=created_at&sort_dir=desc
Response: {
  "items": [...],
  "total": 142,
  "page": 1,
  "page_size": 25
}
```

The frontend always sends `page` (1-based page number), `page_size`, `sort`, and
`sort_dir`. It always receives `items`, `total`, `page`, and `page_size`.

**Backend translation (per-service configuration):**

```yaml
services:
  orders:
    pagination:
      style: "offset"
      page_param: "offset"       # offset = (page - 1) * page_size
      size_param: "limit"
      sort_param: "sort_by"
      sort_dir_param: "order"

  customers:
    pagination:
      style: "page"
      page_param: "page"         # page number passed through
      size_param: "size"
      sort_param: "sort"
      sort_dir_param: "direction"

  inventory:
    pagination:
      style: "cursor"
      cursor_param: "after"
      size_param: "count"
```

**Translation logic:**

| Frontend | Offset Backend | Page-Number Backend | Cursor Backend |
|----------|---------------|--------------------|--------------:|
| `page=2, page_size=25` | `offset=25, limit=25` | `page=2, size=25` | `after={cached_cursor}, count=25` |

For cursor-based backends, the BFF maintains a server-side cursor cache that maps
`(tenant_id, page_id, page_number)` to cursor values, enabling the frontend to use
standard page numbers while the backend uses opaque cursors.

**Response normalization:**

Each backend may return pagination metadata differently:

- `{"data": [...], "meta": {"total": 142}}`
- `{"items": [...], "total_count": 142, "page_number": 2}`
- `{"results": [...], "next_cursor": "abc123", "has_more": true}`

The definition's `items_path` and `total_path` extract the data and total from the
backend response, normalizing it into the standard `DataResponse` format.

## Consequences

### Positive

- **Frontend simplicity:** The Flutter frontend implements one pagination pattern.
  It does not know or care which pagination style each backend uses.
- **Backend independence:** Backend teams can choose their preferred pagination style
  (or change it) without affecting the frontend. The BFF configuration absorbs the
  change.
- **Consistent UX:** All paginated data in the UI behaves identically — same
  parameter names, same response shape, same next/previous page calculation.
- **Sort standardization:** Sort parameters are also translated, so the frontend
  uses consistent `sort` and `sort_dir` parameters regardless of backend conventions
  (`sort_by`, `order`, `sortField`, etc.).

### Negative

- **Cursor-based complexity:** Supporting page-number navigation for cursor-based
  backends requires a server-side cursor cache. This cache has management concerns
  (eviction, stale entries, memory limits). See [doc 13](../13-api-mapping-and-invocation.md)
  for cursor cache management details.
- **Total count uncertainty:** Some backends (especially cursor-based ones) don't
  provide a total count. The BFF returns `total: -1` or omits it, and the frontend
  must handle the "unknown total" case (e.g., show "page 1 of ?" or use infinite
  scroll).
- **Configuration per service:** Each backend service needs a `pagination` block in
  its configuration. This is additional configuration overhead, but it's declarative
  and straightforward.
- **Implementation status:** The `PaginationConfig` structure exists in
  `internal/config/config.go` but pagination translation is not yet enforced at
  runtime. The configuration is defined but dispatch to per-style translators is
  pending implementation.

## Alternatives Considered

### Require All Backends to Use the Same Pagination Style

Mandate that all backend services implement offset-based pagination with standardized
parameter names.

**Rejected because:**
- The BFF team cannot mandate API design for independent backend teams.
- Some backends genuinely benefit from cursor-based pagination (real-time data, large
  datasets, database performance).
- Legacy backends may have pagination styles that are expensive to change.
- This approach pushes complexity onto backend teams instead of absorbing it at the
  BFF layer.

### Frontend Handles Multiple Pagination Styles

Expose each backend's native pagination style to the frontend and let it handle the
differences.

**Rejected because:**
- Violates Principle P7 (Backend Evolution Does Not Break the Frontend). If a backend
  changes its pagination style, the frontend would need to be updated.
- Increases frontend complexity — the Flutter app would need per-service pagination
  adapters.
- Creates inconsistent UX — different paginated lists would behave differently based
  on the backend's implementation choice.

### GraphQL with Relay-Style Pagination

Use GraphQL's Relay connection specification (`first`, `after`, `last`, `before`) as
the universal pagination interface.

**Rejected because:**
- Relay pagination is cursor-based, which makes page-number navigation (jump to page 5)
  awkward. Users in enterprise applications expect page-number navigation.
- GraphQL adds significant infrastructure complexity without proportional benefit in
  this architecture (see ADR-001).
- The BFF would still need to translate between Relay pagination and each backend's
  native style, so the translation layer is needed regardless.

## References

- [Principle P7: Backend Evolution Does Not Break the Frontend](../01-principles-and-invariants.md)
- [Doc 13: API Mapping and Invocation — Pagination Deep-Dive](../13-api-mapping-and-invocation.md)
- [Doc 18: PageProvider interface](../18-core-abstractions-and-interfaces.md)
