# 14 — Schema and Contract Stability

This document describes how Thesa shields the frontend from backend API evolution,
maintaining stable contracts despite independent backend deployments.

---

## The Two-Contract Problem

In a multi-service architecture, two independent contracts exist:

```
Frontend Contract              Backend Contract
(what the UI expects)          (what the API provides)

Stable, versioned              Evolves with each service release
Changed by BFF/UI teams        Changed by domain teams
UI field names (snake_case)    API field names (camelCase, etc.)
Standard pagination            Service-specific pagination
Standard error format          Service-specific errors
```

Without an adapter layer, changes to the backend contract break the frontend.
With Thesa, the definition file acts as the adapter, absorbing backend changes
while keeping the frontend contract stable.

---

## The Adapter Layer

The adapter layer consists of three components in each definition:

### 1. ResponseMapping (for data fetching)

```yaml
mapping:
  items_path: "data.orders"        # Where to find items in the response
  total_path: "meta.total"          # Where to find total count
  field_map:                        # Backend → UI field renaming
    order_number: "orderNumber"
    created_at: "createdAt"
    customer_name: "customer.name"  # Nested field extraction
```

### 2. InputMapping (for commands)

```yaml
input:
  path_params:
    orderId: "route.id"
  field_projection:
    customerId: "input.customer_id"     # UI field → backend field
    shippingAddress: "input.shipping_address"
```

### 3. OutputMapping (for command responses)

```yaml
output:
  fields:
    id: "data.id"
    order_number: "data.orderNumber"
  error_map:
    ORDER_NOT_FOUND: "This order no longer exists"
```

---

## Absorbing Backend Changes

### Scenario 1: Field Renamed

Backend renames `orderNumber` to `order_num`.

**Change in definition:**
```yaml
# Before:
field_map:
  order_number: "orderNumber"

# After:
field_map:
  order_number: "order_num"
```

**Frontend impact:** None. Frontend still receives `order_number`.

### Scenario 2: Response Structure Changed

Backend moves pagination metadata from `meta.total` to `pagination.totalItems`.

**Change in definition:**
```yaml
# Before:
total_path: "meta.total"

# After:
total_path: "pagination.totalItems"
```

**Frontend impact:** None. Frontend still receives `total_count` in DataResponse.

### Scenario 3: New API Version

Backend releases v2 API with new endpoints and response format.

**Change in definition:**
```yaml
# Before:
operation:
  operation_id: "listOrders"
  service_id: "orders-svc"

# After:
operation:
  operation_id: "listOrdersV2"
  service_id: "orders-svc-v2"
```

Plus updated `field_map` and `items_path` to match the new response format.

**Frontend impact:** None. Same descriptors and data responses.

### Scenario 4: Error Code Changed

Backend changes error code from `ORDER_NOT_FOUND` to `RESOURCE_NOT_FOUND`.

**Change in definition:**
```yaml
# Before:
error_map:
  ORDER_NOT_FOUND: "This order no longer exists"

# After:
error_map:
  RESOURCE_NOT_FOUND: "This order no longer exists"
```

**Frontend impact:** None. Same error message displayed.

### Scenario 5: New Field Added (Backend)

Backend adds a new field `estimatedDelivery` to the order response.

**No definition change needed** if the field is not used in the UI.
The field is simply ignored by the ResponseMapping.

To expose the new field, add a new column:
```yaml
columns:
  - field: "estimated_delivery"
    label: "Est. Delivery"
    type: "date"
```
And update the field_map:
```yaml
field_map:
  estimated_delivery: "estimatedDelivery"
```

**Frontend impact:** New column appears (frontend adapts to new columns).

---

## Versioned Descriptor Contracts

### API Versioning Strategy

The BFF supports version negotiation via the Accept header:

```
Accept: application/vnd.thesa.v1+json    → v1 descriptors
Accept: application/vnd.thesa.v2+json    → v2 descriptors
Accept: application/json                 → latest version
```

### When to Version

Version the descriptor contract when making **breaking changes** to the descriptor
schema. These are rare because the design allows for additive changes:

| Change | Breaking? | Action |
|--------|----------|--------|
| Add new field to descriptor | No | Add field, old clients ignore it |
| Add new action type | No | Old clients ignore unknown types |
| Remove field from descriptor | Yes | New version needed |
| Change field type | Yes | New version needed |
| Change field semantics | Yes | New version needed |
| Rename field | Yes | New version needed |

### Dual-Version Serving

During migration, the BFF can serve both v1 and v2:

```go
router.Route("/ui", func(r chi.Router) {
    r.Use(versionNegotiationMiddleware)

    r.Get("/pages/{pageId}", func(w http.ResponseWriter, r *http.Request) {
        version := r.Context().Value("api_version").(int)
        switch version {
        case 1:
            servePageV1(w, r)
        case 2:
            servePageV2(w, r)
        }
    })
})
```

### Deprecation Process

1. Announce deprecation of v1 (e.g., 6-month notice).
2. Add `Deprecation: true` header to v1 responses.
3. Add `Sunset: <date>` header.
4. Monitor v1 usage metrics.
5. Remove v1 after sunset date.

---

## Field Naming Conventions

### UI Field Names (Frontend Contract)

- **snake_case**: `order_number`, `created_at`, `shipping_address`
- Consistent across all domains.
- Defined in the definition's `field_map` values (left side) and field definitions.

### Backend Field Names (Backend Contract)

- **Varies by service**: `orderNumber` (camelCase), `OrderNumber` (PascalCase), etc.
- Defined in the OpenAPI spec and `field_map` values (right side).

### The field_map Direction

```yaml
field_map:
  {ui_field_name}: "{backend_field_path}"
```

Left side: the name the UI uses.
Right side: the path in the backend response.

Example:
```yaml
field_map:
  order_number: "orderNumber"          # Simple rename
  customer_name: "customer.name"       # Nested field extraction
  total: "financials.totalAmount"      # Deep path
  status: "status"                     # Same name (explicit for clarity)
```

---

## Nested Field Extraction

The `field_map` and `items_path` support dot-notation for nested fields:

### Example: Nested Response

Backend response:
```json
{
  "data": {
    "order": {
      "id": "ord-123",
      "header": {
        "number": "ORD-2024-001",
        "status": "pending"
      },
      "customer": {
        "profile": {
          "name": "ACME Corp"
        }
      },
      "financials": {
        "total": { "amount": 1500.00, "currency": "USD" }
      }
    }
  }
}
```

Definition:
```yaml
mapping:
  items_path: "data.order"               # Single item (for detail pages)
  field_map:
    id: "id"
    order_number: "header.number"
    status: "header.status"
    customer_name: "customer.profile.name"
    total_amount: "financials.total.amount"
    currency: "financials.total.currency"
```

Resulting UI data:
```json
{
  "id": "ord-123",
  "order_number": "ORD-2024-001",
  "status": "pending",
  "customer_name": "ACME Corp",
  "total_amount": 1500.00,
  "currency": "USD"
}
```

---

## Compatibility Rules for Definitions

| Change Type | Compatibility | Frontend Impact |
|-------------|---------------|-----------------|
| **Additions** | | |
| Add new page | Compatible | Frontend may not render it yet |
| Add new column to table | Compatible | Frontend renders new column |
| Add new section to page | Compatible | Frontend renders new section |
| Add new action | Compatible | Frontend renders new button |
| Add new command | Compatible | Frontend may not use it yet |
| Add new workflow | Compatible | Frontend may not use it yet |
| **Modifications** | | |
| Change field_map | Compatible | Absorbs backend change, no frontend change |
| Change operation_id | Compatible | Transparent to frontend |
| Change page title | Compatible | Frontend shows new title |
| Change default_sort | Compatible | Frontend uses new default |
| Change page_size | Compatible | Frontend uses new size |
| Change column type | **Potentially breaking** | If frontend has type-specific rendering |
| **Removals** | | |
| Remove column | Compatible | Frontend adapts (column disappears) |
| Remove action | Compatible | Frontend adapts (button disappears) |
| Remove section | Compatible | Frontend adapts |
| **ID Changes** | | |
| Rename page ID | **Breaking** | Frontend routes break |
| Rename command ID | **Breaking** | Frontend action references break |
| Rename form ID | **Breaking** | Frontend form references break |
| Rename workflow ID | **Breaking** | Frontend workflow references break |
| **Semantic Changes** | | |
| Change field semantics | **Breaking** | Frontend displays wrong data |
| Change command behavior | **Potentially breaking** | Frontend expectations may not match |

### Coordinating Breaking Changes

When a breaking change is necessary:

1. **Create new ID alongside old**: e.g., `orders.list_v2` alongside `orders.list`.
2. **Update frontend** to use the new ID.
3. **Deploy frontend** with new references.
4. **Remove old definition** once no frontend version uses it.

This is a blue-green approach at the definition level.
