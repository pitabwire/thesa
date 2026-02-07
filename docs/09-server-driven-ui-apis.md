# 09 — Server-Driven UI Metadata APIs

This document provides the complete specification for every BFF endpoint, including
request format, response format, query parameters, error responses, and behavioral
details.

---

## API Design Philosophy

The BFF exposes a **generic, resource-agnostic API**. Adding a new domain (e.g., "shipments")
does not add new endpoints. The existing endpoints serve all domains through parameterized
IDs (`pageId`, `formId`, `commandId`, etc.).

The frontend interacts with Thesa through a small vocabulary:
1. **Discover** what's available → `GET /ui/navigation`
2. **Describe** what a page/form looks like → `GET /ui/pages/{id}`, `GET /ui/forms/{id}`
3. **Load** data for display → `GET /ui/pages/{id}/data`, `GET /ui/forms/{id}/data`
4. **Execute** mutations → `POST /ui/commands/{id}`
5. **Orchestrate** workflows → `POST /ui/workflows/{id}/start`, `POST /ui/workflows/{id}/advance`
6. **Search** across domains → `GET /ui/search`
7. **Resolve** reference data → `GET /ui/lookups/{id}`

---

## GET /ui/navigation

Returns the navigation tree for the authenticated user.

### Request

```
GET /ui/navigation
Authorization: Bearer {token}
X-Partition-Id: {partition}
```

No query parameters.

### Response (200 OK)

```json
{
  "items": [
    {
      "id": "orders",
      "label": "Orders",
      "icon": "shopping_cart",
      "children": [
        {
          "id": "orders.list",
          "label": "All Orders",
          "icon": "list",
          "route": "/orders",
          "children": [],
          "badge": { "count": 12, "style": "warning" }
        }
      ]
    }
  ]
}
```

> **Note:** The navigation response is a bare `NavigationTree` object, not wrapped in
> a `data`/`meta` envelope. Parent nodes without a route omit the `route` field.

### Resolution Process

1. Iterate all domains in the DefinitionRegistry.
2. For each domain, check if the user has the navigation-level capabilities.
3. For visible domains, iterate child navigation items and filter by capabilities.
4. Sort domains and children by their `order` field.
5. Optionally resolve badges by invoking badge operations (asynchronously, with timeout).
6. If a badge operation fails, omit the badge (don't fail the whole response).

### Caching

Navigation trees change infrequently. The BFF may cache the resolved navigation
per (capabilities_hash, partition) with a short TTL (30s-60s). Badge counts should
not be cached or cached with a very short TTL (5-10s).

The frontend may cache the navigation tree locally and refresh on:
- Explicit user action (e.g., clicking a refresh button).
- After executing a command that might change badge counts.
- On a timer (every 60 seconds).

---

## GET /ui/pages/{pageId}

Returns the page descriptor for the given page.

### Request

```
GET /ui/pages/orders.list
Authorization: Bearer {token}
X-Partition-Id: {partition}
```

### Response (200 OK)

See [PageDescriptor](08-ui-descriptor-model.md#pagedescriptor) for full schema.

### Error Responses

| Status | When |
|--------|------|
| 401 | Missing or invalid token |
| 403 | User lacks required page capabilities |
| 404 | Page ID not found in DefinitionRegistry |

### Resolution Process

1. Look up PageDefinition by pageId.
2. Check page-level capabilities → 403 if insufficient.
3. Resolve table:
   a. Filter columns by visibility capabilities.
   b. Resolve filter options (inline or via lookups).
   c. Filter row_actions and bulk_actions by capabilities.
4. Resolve sections:
   a. Filter sections by section-level capabilities.
   b. Filter fields by visibility capabilities.
   c. Resolve read_only expressions to boolean values.
5. Resolve page-level actions by capabilities.
6. Replace `data_source.operation_id` with BFF data endpoint URL.
7. Assemble and return PageDescriptor.

---

## GET /ui/pages/{pageId}/data

Returns the data for a page's table or sections.

### Request

```
GET /ui/pages/orders.list/data?page=1&page_size=25&sort=created_at&sort_dir=desc&status=pending
Authorization: Bearer {token}
X-Partition-Id: {partition}
```

### Query Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `page` | int | No | 1 | Page number (1-based) |
| `page_size` | int | No | From definition | Items per page (max 200) |
| `sort` | string | No | From definition | Sort field (must be a sortable column) |
| `sort_dir` | string | No | From definition | "asc" or "desc" |
| `q` | string | No | — | Free-text search within the page's data |
| `{filter_field}` | string | No | — | Filter value (field must be defined in filters) |
| `{filter_field}_gte` | string | No | — | Greater-than-or-equal filter |
| `{filter_field}_lte` | string | No | — | Less-than-or-equal filter |
| `{filter_field}_from` | string | No | — | Date range start |
| `{filter_field}_to` | string | No | — | Date range end |

### Response (200 OK)

```json
{
  "data": {
    "items": [
      { "id": "ord-001", "order_number": "ORD-2024-001", ... },
      { "id": "ord-002", "order_number": "ORD-2024-002", ... }
    ],
    "total_count": 142,
    "page": 1,
    "page_size": 25
  },
  "meta": { "trace_id": "..." }
}
```

### Resolution Process

1. Look up PageDefinition.
2. Check page-level capabilities.
3. Extract DataSourceDefinition.
4. Validate query parameters:
   a. `sort` field must match a sortable column → 400 if invalid.
   b. Filter fields must match defined filters → ignored if unknown.
   c. `page_size` capped at 200 → silently reduced if exceeded.
5. Map BFF query parameters to backend parameters using:
   a. Service pagination configuration (offset vs. cursor vs. page).
   b. Filter field names translated via field_map (reverse direction).
   c. Sort field translated via field_map.
6. Invoke backend via OperationInvoker.
7. Apply ResponseMapping:
   a. Extract items from `items_path`.
   b. Extract total count from `total_path`.
   c. Rename fields using `field_map`.
8. Return DataResponse.

### Pagination Normalization

The BFF normalizes all backend pagination styles to a consistent interface:

| Backend Style | How BFF Translates |
|--------------|-------------------|
| Offset/Limit (`?offset=25&limit=25`) | `page=2, page_size=25` → `offset=25, limit=25` |
| Page/PerPage (`?page=2&per_page=25`) | Pass through directly |
| Cursor (`?cursor=abc&limit=25`) | BFF manages cursor state; frontend uses page numbers |

For cursor-based backends, the BFF maintains a short-lived cursor cache per
(user, page, filters) so the frontend can use simple page numbers.

---

## GET /ui/forms/{formId}

Returns the form descriptor.

### Request

```
GET /ui/forms/orders.edit_form
Authorization: Bearer {token}
X-Partition-Id: {partition}
```

### Response (200 OK)

See [FormDescriptor](08-ui-descriptor-model.md#formdescriptor) for full schema.

### Error Responses

| Status | When |
|--------|------|
| 401 | Missing or invalid token |
| 403 | User lacks required form capabilities |
| 404 | Form ID not found |

---

## GET /ui/forms/{formId}/data

Returns pre-populated data for a form (e.g., when editing an existing resource).

### Request

```
GET /ui/forms/orders.edit_form/data?id=ord-123
Authorization: Bearer {token}
X-Partition-Id: {partition}
```

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | Conditional | Resource ID (required when the form has `load_source`) |
| Other params | string | No | Additional parameters needed by the load operation |

### Response (200 OK)

```json
{
  "customer_id": "cust-001",
  "customer_name": "Acme Corp",
  "shipping_address": "123 Main St, Springfield",
  "notes": "Handle with care",
  "priority": "normal"
}
```

> **Note:** The form data response is a flat field-value map, not wrapped in an envelope.
> Field names use UI-facing names (after field_map renaming).

### Resolution Process

1. Look up FormDefinition.
2. Check form capabilities.
3. If `load_source` is defined:
   a. Map query parameters to backend operation parameters.
   b. Invoke backend to fetch the resource.
   c. Apply ResponseMapping to extract field values.
   d. Filter fields by visibility capabilities.
   e. Return field values.
4. If `load_source` is NOT defined:
   a. Return empty field values (new resource form).

---

## POST /ui/commands/{commandId}

Executes a command. See [10 — Command & Action Model](10-command-and-action-model.md)
for the complete execution pipeline.

### Request

```
POST /ui/commands/orders.update
Content-Type: application/json
Authorization: Bearer {token}
X-Partition-Id: {partition}
Idempotency-Key: abc-123          (optional)

{
  "input": {
    "customer_id": "cust-002",
    "shipping_address": "456 Oak Ave",
    "priority": "high"
  },
  "route_params": {
    "id": "ord-123"
  }
}
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `input` | object | Yes | User-provided field values |
| `route_params` | object | No | Current route parameters (e.g., resource ID) |
| `idempotency_key` | string | No | Idempotency key (alternative to header) |

### Response (200 OK)

```json
{
  "success": true,
  "message": "Order updated successfully",
  "result": { "id": "ord-123", "order_number": "ORD-2024-001" }
}
```

> **Note:** The command response is a bare `CommandResponse` object, not wrapped in a
> `data`/`meta` envelope. See [08](08-ui-descriptor-model.md#commandresponse).

### Error Responses

| Status | Code | When |
|--------|------|------|
| 400 | BAD_REQUEST | Malformed request body |
| 401 | UNAUTHORIZED | Invalid token |
| 403 | FORBIDDEN | Missing capabilities |
| 404 | NOT_FOUND | Unknown command ID |
| 409 | CONFLICT | Idempotency conflict (different input, same key) |
| 422 | VALIDATION_ERROR | Input validation failed |
| 429 | RATE_LIMITED | Rate limit exceeded |
| 500 | INTERNAL_ERROR | Unexpected error |
| 502 | BACKEND_UNAVAILABLE | Backend service unreachable |
| 504 | BACKEND_TIMEOUT | Backend service timed out |

---

## POST /ui/workflows/{workflowId}/start

Starts a new workflow instance.

### Request

```
POST /ui/workflows/orders.approval/start
Content-Type: application/json
Authorization: Bearer {token}
X-Partition-Id: {partition}

{
  "input": {
    "order_id": "ord-123",
    "customer_email": "customer@example.com"
  }
}
```

### Response (201 Created)

Returns a WorkflowDescriptor. See [08](08-ui-descriptor-model.md#workflowdescriptor).

---

## POST /ui/workflows/{instanceId}/advance

Advances a workflow by one step.

### Request

```
POST /ui/workflows/wf-a1b2c3/advance
Content-Type: application/json
Authorization: Bearer {token}
X-Partition-Id: {partition}

{
  "event": "approved",
  "input": {
    "approval_notes": "Looks good",
    "approved_by": "alice@acme-corp.com"
  }
}
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `event` | string | Yes | Transition event: "completed", "approved", "rejected" |
| `input` | object | No | User-provided data for this step |
| `comment` | string | No | Human comment for audit trail |

### Response (200 OK)

Returns updated WorkflowDescriptor.

### Error Responses

| Status | Code | When |
|--------|------|------|
| 403 | FORBIDDEN | User lacks step capabilities |
| 404 | WORKFLOW_NOT_FOUND | Instance not found (or wrong tenant) |
| 409 | WORKFLOW_NOT_ACTIVE | Workflow is not in active status |
| 422 | INVALID_TRANSITION | Event is not valid for current step |

---

## GET /ui/workflows/{instanceId}

Returns the current state of a workflow instance.

### Request

```
GET /ui/workflows/wf-a1b2c3
Authorization: Bearer {token}
X-Partition-Id: {partition}
```

### Response (200 OK)

Returns WorkflowDescriptor.

---

## GET /ui/workflows

Lists the user's workflow instances.

### Request

```
GET /ui/workflows?status=active&workflow_id=orders.approval&page=1&page_size=10
Authorization: Bearer {token}
X-Partition-Id: {partition}
```

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `status` | string | No | Filter by status: active, completed, failed, cancelled |
| `workflow_id` | string | No | Filter by workflow definition ID |
| `page` | int | No | Page number |
| `page_size` | int | No | Items per page |

### Response (200 OK)

```json
{
  "data": {
    "items": [
      {
        "id": "wf-a1b2c3",
        "workflow_id": "orders.approval",
        "name": "Order Approval",
        "status": "active",
        "current_step": "review",
        "created_at": "2025-01-15T10:30:00Z",
        "updated_at": "2025-01-15T10:30:00Z"
      }
    ],
    "total_count": 3,
    "page": 1,
    "page_size": 10
  },
  "meta": { "trace_id": "..." }
}
```

---

## POST /ui/workflows/{instanceId}/cancel

Cancels an active workflow.

### Request

```
POST /ui/workflows/wf-a1b2c3/cancel
Content-Type: application/json
Authorization: Bearer {token}
X-Partition-Id: {partition}

{
  "reason": "Customer requested cancellation"
}
```

### Response (200 OK)

Returns updated WorkflowDescriptor with status "cancelled".

---

## GET /ui/search

Global search across all domains.

### Request

```
GET /ui/search?q=acme&page=1&page_size=20
Authorization: Bearer {token}
X-Partition-Id: {partition}
```

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `q` | string | Yes | Search query (min 2 characters) |
| `page` | int | No | Page number (default: 1) |
| `page_size` | int | No | Results per page (default: 20, max: 50) |
| `domain` | string | No | Filter to a specific domain |

### Response (200 OK)

See [SearchResponse](08-ui-descriptor-model.md#searchresponse).

---

## GET /ui/lookups/{lookupId}

Returns reference data options for select fields and autocomplete.

### Request

```
GET /ui/lookups/customers.search?q=acme
Authorization: Bearer {token}
X-Partition-Id: {partition}
```

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `q` | string | No | Search term for search-as-you-type lookups |

### Response (200 OK)

```json
{
  "data": {
    "options": [
      { "label": "Acme Corp", "value": "cust-001", "icon": "" },
      { "label": "Acme Industries", "value": "cust-002", "icon": "" }
    ]
  },
  "meta": { "trace_id": "..." }
}
```

### Caching

Lookups with `cache` configured are cached server-side:
- `scope: "global"` — one cache entry for all tenants (for truly static data).
- `scope: "tenant"` — separate cache per tenant.

The frontend may also cache lookup responses locally for performance.

---

## GET /ui/health

Health check (no authentication required).

```json
{
  "status": "ok",
  "version": "1.0.0",
  "commit": "abc1234"
}
```

## GET /ui/ready

Readiness check (no authentication required).

```json
{
  "status": "ready",
  "checks": {
    "definitions": { "status": "ok", "latency_ms": 2 },
    "openapi_index": { "status": "ok", "latency_ms": 1 },
    "workflow_store": { "status": "ok", "latency_ms": 5 },
    "policy_engine": { "status": "ok", "latency_ms": 1 }
  }
}
```

If any check fails:
```json
{
  "status": "not_ready",
  "checks": {
    "definitions": { "status": "ok", "latency_ms": 2 },
    "openapi_index": { "status": "ok", "latency_ms": 1 },
    "workflow_store": { "status": "error", "latency_ms": 0, "error": "connection refused" },
    "policy_engine": { "status": "ok", "latency_ms": 1 }
  }
}
```
Status code: 503 Service Unavailable.
