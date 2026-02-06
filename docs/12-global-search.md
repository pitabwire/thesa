# 12 — Global Search

This document describes the global search system — a federated search that aggregates
results from multiple backend services into a single, unified search response for the
frontend.

---

## Overview

Global search allows users to find resources across all domains from a single search
box. When a user types "ACME" into the search bar, they might see results from:

- **Orders:** "ORD-2024-001 — ACME Corp"
- **Customers:** "ACME Corp — Enterprise Customer"
- **Inventory:** "ACME Widget — SKU-12345"

Each domain contributes search results through a `SearchDefinition` in its definition
file. The BFF calls all eligible search providers in parallel, merges results, and
returns a ranked, paginated response.

---

## Search Flow

```
GET /ui/search?q=ACME&page=1&page_size=20
  │
  ├── 1. Extract RequestContext and CapabilitySet.
  │
  ├── 2. Validate query:
  │      - q must be at least 2 characters → 400 if shorter.
  │      - page_size capped at 50.
  │
  ├── 3. Get all SearchDefinitions from registry.
  │
  ├── 4. Filter by capabilities:
  │      For each search definition:
  │        If user lacks any search.capabilities → exclude.
  │
  ├── 5. Execute eligible providers in parallel:
  │      For each eligible provider (concurrently):
  │      ┌──────────────────────────────────────────────┐
  │      │  a. Map "q" to the provider's search field.  │
  │      │  b. Build invocation input.                  │
  │      │  c. Invoke backend via OperationInvoker.     │
  │      │  d. Apply ResultMapping.                     │
  │      │  e. Score results (provider weight × relevance). │
  │      │  f. Return results or error.                 │
  │      └──────────────────────────────────────────────┘
  │      Timeout per provider: 3 seconds (configurable).
  │      If provider fails/times out: skip it (don't fail the search).
  │
  ├── 6. Merge results from all providers.
  │
  ├── 7. Deduplicate:
  │      If two providers return the same route + id, keep highest score.
  │
  ├── 8. Sort by score (descending).
  │
  ├── 9. Apply pagination.
  │
  ├── 10. Build SearchResponse.
  │
  └── 11. Return response with metadata about which providers responded.
```

---

## SearchProvider Interface

```
SearchProvider
  ├── Search(ctx, rctx, query string, pagination Pagination) → (SearchResponse, error)
  │     Orchestrates the full search across all eligible providers.
  │
  └── (internally manages provider execution and result merging)
```

### Pagination

```
Pagination
  ├── Page      int    // 1-based page number
  ├── PageSize  int    // items per page (max 50)
  └── Domain    string // optional: filter to a specific domain
```

---

## SearchDefinition Configuration

Each domain registers its search capability:

```yaml
searches:
  - id: "orders.search"
    domain: "orders"                      # Category label in results
    capabilities: ["orders:search:execute"]
    operation:
      type: "openapi"
      operation_id: "searchOrders"
      service_id: "orders-svc"
    result_mapping:
      items_path: "data.results"          # JSON path to results array
      title_field: "orderNumber"          # Field for result title
      subtitle_field: "customerName"      # Field for subtitle
      category_field: "status"            # Field for category tag
      icon_field: ""                      # Field for icon (optional)
      route: "/orders/{id}"              # Navigation target template
      id_field: "id"                      # Field for resource ID
    weight: 10                            # Ranking weight (higher = prioritized)
    max_results: 5                        # Max results from this provider
```

### Result Mapping

The `result_mapping` transforms backend search results into a standardized format:

```
Backend response:
{
  "data": {
    "results": [
      { "id": "ord-001", "orderNumber": "ORD-2024-001", "customerName": "ACME Corp", "status": "pending" },
      { "id": "ord-002", "orderNumber": "ORD-2024-002", "customerName": "ACME Industries", "status": "shipped" }
    ]
  }
}

After mapping:
[
  { "id": "ord-001", "title": "ORD-2024-001", "subtitle": "ACME Corp", "category": "Orders", "route": "/orders/ord-001" },
  { "id": "ord-002", "title": "ORD-2024-002", "subtitle": "ACME Industries", "category": "Orders", "route": "/orders/ord-002" }
]
```

### Route Template Resolution

The `route` field in result_mapping supports parameter substitution:

```yaml
route: "/orders/{id}"
id_field: "id"
```

For each result, `{id}` is replaced with the value of the `id` field from the result.

More complex templates:
```yaml
route: "/orders/{orderId}/items/{itemId}"
# Would need multiple ID fields — not typical for search results.
```

---

## Scoring and Ranking

### Score Computation

Each search result receives a score:

```
score = provider_weight × position_score

Where:
  provider_weight = SearchDefinition.weight (default: 1)
  position_score = 1.0 - (index / total_results_from_provider * 0.5)
```

This gives higher scores to:
1. Results from higher-weight providers (e.g., orders weighted 10 vs. audit logs weighted 2).
2. Results that appear earlier in the provider's response (backends typically return
   most relevant results first).

### Cross-Provider Ranking

After scoring, results from all providers are merged into a single list and sorted
by score descending. This means a high-relevance result from a low-weight provider
can still outrank a low-relevance result from a high-weight provider.

### Domain Filtering

When the user selects a domain filter (`?domain=orders`), only that domain's search
provider is queried. This is faster and returns more focused results.

---

## Error Handling

### Provider Failure

If a search provider fails (backend error, timeout, circuit breaker open), the
search continues with remaining providers. The response includes metadata about
which providers responded:

```json
{
  "data": {
    "results": [...],
    "total_count": 8,
    "query": "ACME"
  },
  "meta": {
    "trace_id": "...",
    "providers": {
      "orders.search": "ok",
      "customers.search": "ok",
      "inventory.search": "timeout"
    }
  }
}
```

The frontend can show a notice: "Some results may be missing (Inventory search unavailable)."

### All Providers Fail

If ALL providers fail, return an empty result set with the provider error metadata.
Do NOT return a 500 error — partial results are better than no results.

### Query Too Short

If `q` is less than 2 characters, return 400:
```json
{ "error": { "code": "BAD_REQUEST", "message": "Search query must be at least 2 characters" } }
```

---

## Performance Considerations

### Parallel Execution

All eligible providers are called concurrently using goroutines. The BFF waits
for all providers to respond or timeout:

```go
// Pseudo-code
results := make(chan providerResult, len(providers))
for _, p := range providers {
    go func(provider SearchDef) {
        ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
        defer cancel()
        res, err := invoker.Invoke(ctx, ...)
        results <- providerResult{provider: provider, result: res, err: err}
    }(p)
}

// Collect results
for range providers {
    r := <-results
    if r.err != nil { log warning; continue }
    merged = append(merged, mapResults(r)...)
}
```

### Provider Timeout

Each provider has a 3-second timeout (configurable). If a backend is slow, its
results are excluded rather than delaying the entire search response.

### Result Limiting

Each provider is limited to `max_results` (default: 5). This prevents a single
provider from dominating the results and keeps total result processing fast.

### Caching

Search results are NOT cached by default (searches are dynamic and query-dependent).
However, the underlying lookup data (e.g., customer names) may be cached by the
backend services.

---

## Use Cases

### Use Case 1: Quick Navigation

User types an order number: "ORD-2024-001"

→ Orders search provider returns exact match.
→ Single result displayed.
→ User clicks to navigate to `/orders/ord-001`.

### Use Case 2: Cross-Domain Discovery

User types a customer name: "Globex"

→ Orders: "ORD-2024-015 — Globex Inc" (order for this customer)
→ Customers: "Globex Inc — Enterprise" (customer profile)
→ User sees results from multiple domains, picks the one they need.

### Use Case 3: Filtered Search

User types "pending" and filters by Orders domain:

→ Only the Orders search provider is queried.
→ Results show all pending orders matching "pending" in the text.

### Use Case 4: Large-Scale Search

Organization with 20 domains, each with a search provider:

→ 20 parallel backend calls (all within 3s timeout).
→ ~100 results (5 per provider).
→ Merged, ranked, paginated.
→ Frontend shows top 20.
