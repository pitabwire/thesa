# 08 — UI Descriptor Model

This document describes the descriptor types — the resolved, capability-filtered data
structures that the BFF sends to the frontend. Descriptors are the **external contract**
between the BFF and the UI rendering engine.

---

## Definitions vs. Descriptors

This distinction is fundamental to Thesa's architecture.

| Aspect | Definition (Internal) | Descriptor (External) |
|--------|----------------------|----------------------|
| **Audience** | BFF internals, definition authors | Frontend rendering engine |
| **Contains backend details?** | Yes: operation IDs, service IDs, handlers | No: only BFF endpoint URLs |
| **Contains capability strings?** | Yes: used for filtering | No: already evaluated and filtered |
| **Contains field mappings?** | Yes: backend → UI field names | No: only UI field names |
| **Contains all elements?** | Yes: all defined elements | No: only elements the user can see |
| **Stored where?** | YAML files, DefinitionRegistry | Generated per-request, never stored |
| **Who creates them?** | Domain teams | BFF providers (automatic) |

### The Resolution Process

```
Definition + RequestContext + CapabilitySet → Descriptor

Steps:
  1. Load definition from registry.
  2. Evaluate capabilities against every element.
  3. Remove elements the user cannot access.
  4. Resolve expressions and references.
  5. Replace internal references with BFF endpoints.
  6. Strip all internal metadata.
  7. Return descriptor.
```

---

## Descriptor Type Catalog

### NavigationTree

Returned by `GET /ui/navigation`.

```json
{
  "items": [
    {
      "id": "orders",
      "label": "Orders",
      "icon": "shopping_cart",
      "route": null,
      "children": [
        {
          "id": "orders.list",
          "label": "All Orders",
          "icon": "list",
          "route": "/orders",
          "children": [],
          "badge": {
            "count": 12,
            "style": "warning"
          }
        },
        {
          "id": "orders.create",
          "label": "Create Order",
          "icon": "add",
          "route": "/orders/new",
          "children": [],
          "badge": null
        }
      ],
      "badge": null
    },
    {
      "id": "inventory",
      "label": "Inventory",
      "icon": "warehouse",
      "route": null,
      "children": [...]
    }
  ]
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique node ID |
| `label` | string | Display text |
| `icon` | string | Icon identifier (Material Icons or custom) |
| `route` | string or null | Navigation route (null for parent nodes) |
| `children` | NavigationNode[] | Child nodes (empty for leaf nodes) |
| `badge` | BadgeDescriptor or null | Optional count indicator |

**BadgeDescriptor:**

| Field | Type | Description |
|-------|------|-------------|
| `count` | int | The count to display |
| `style` | string | Visual style: "info", "warning", "danger" |

**What's filtered out:** Entire domains or navigation items where the user lacks
the required capabilities. Items are sorted by their `order` field.

### PageDescriptor

Returned by `GET /ui/pages/{pageId}`.

```json
{
  "id": "orders.list",
  "title": "Orders",
  "route": "/orders",
  "layout": "list",
  "refresh_interval": 30,
  "breadcrumb": [
    { "label": "Home", "route": "/" },
    { "label": "Orders" }
  ],
  "table": {
    "columns": [...],
    "filters": [...],
    "row_actions": [...],
    "bulk_actions": [...],
    "data_endpoint": "/ui/pages/orders.list/data",
    "default_sort": "created_at",
    "sort_dir": "desc",
    "page_size": 25,
    "selectable": true
  },
  "sections": [],
  "actions": [
    {
      "id": "orders.create_action",
      "label": "New Order",
      "icon": "add",
      "style": "primary",
      "type": "navigate",
      "enabled": true,
      "visible": true,
      "navigate_to": "/orders/new"
    }
  ]
}
```

**Key transformations from definition to descriptor:**
- `data_source.operation_id` → `data_endpoint: "/ui/pages/orders.list/data"` (BFF URL)
- Columns filtered by visibility capabilities
- Actions filtered by action capabilities
- `capabilities` field stripped entirely

### TableDescriptor

Nested within PageDescriptor for `layout: "list"` pages.

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `columns` | ColumnDescriptor[] | Visible columns |
| `filters` | FilterDescriptor[] | Available filters |
| `row_actions` | ActionDescriptor[] | Per-row actions |
| `bulk_actions` | ActionDescriptor[] | Multi-select actions |
| `data_endpoint` | string | BFF URL for fetching data |
| `default_sort` | string | Default sort field |
| `sort_dir` | string | Default sort direction |
| `page_size` | int | Default page size |
| `selectable` | bool | Whether rows have checkboxes |

### ColumnDescriptor

```json
{
  "field": "order_number",
  "label": "Order #",
  "type": "link",
  "sortable": true,
  "format": "",
  "width": "150px",
  "link": {
    "route": "/orders/{id}",
    "params": { "id": "id" }
  },
  "status_map": null
}
```

| Field | Type | Description |
|-------|------|-------------|
| `field` | string | Data field key (matches keys in DataResponse items) |
| `label` | string | Column header text |
| `type` | string | Column type (text, number, currency, date, datetime, status, link, boolean) |
| `sortable` | bool | Whether column supports sorting |
| `format` | string | Format pattern (e.g., "USD" for currency) |
| `width` | string | Width hint (e.g., "150px", "20%") |
| `link` | LinkDescriptor or null | Navigation link from cell |
| `status_map` | map or null | Status value → style mapping |

### FilterDescriptor

```json
{
  "field": "status",
  "label": "Status",
  "type": "select",
  "operator": "eq",
  "options": [
    { "label": "Pending", "value": "pending", "icon": "" },
    { "label": "Confirmed", "value": "confirmed", "icon": "" }
  ],
  "default": null
}
```

| Field | Type | Description |
|-------|------|-------------|
| `field` | string | Query parameter name for this filter |
| `label` | string | Filter label |
| `type` | string | Input type: text, select, multi-select, date-range, number-range, boolean |
| `operator` | string | Filter operator: eq, neq, contains, gte, lte, between, in |
| `options` | OptionDescriptor[] | Available options (for select types). Resolved from lookups. |
| `default` | any | Default filter value |

**Options resolution:** If the definition uses a `lookup_id`, the options are resolved
at descriptor generation time by calling the lookup's backend operation or cache.
If the definition uses `static` options, they are passed through directly.

### FormDescriptor

Returned by `GET /ui/forms/{formId}`.

```json
{
  "id": "orders.edit_form",
  "title": "Edit Order",
  "sections": [
    {
      "id": "basics",
      "title": "Order Details",
      "layout": "grid",
      "columns": 2,
      "collapsible": false,
      "collapsed": false,
      "fields": [
        {
          "field": "customer_id",
          "label": "Customer",
          "type": "reference",
          "read_only": false,
          "required": true,
          "validation": null,
          "options": [],
          "format": "",
          "placeholder": "Search customers...",
          "help_text": "",
          "span": 2,
          "value": null,
          "depends_on": []
        },
        {
          "field": "shipping_address",
          "label": "Shipping Address",
          "type": "textarea",
          "read_only": false,
          "required": true,
          "validation": { "max_length": 500 },
          "options": [],
          "format": "",
          "placeholder": "Enter shipping address",
          "help_text": "",
          "span": 1,
          "value": null,
          "depends_on": []
        }
      ]
    }
  ],
  "actions": [
    {
      "id": "submit",
      "label": "Save",
      "style": "primary",
      "type": "command",
      "enabled": true,
      "visible": true,
      "command_id": "orders.update"
    },
    {
      "id": "cancel",
      "label": "Cancel",
      "style": "secondary",
      "type": "navigate",
      "enabled": true,
      "visible": true,
      "navigate_to": "/orders/{id}"
    }
  ],
  "submit_endpoint": "/ui/commands/orders.update",
  "success_route": "/orders/{id}",
  "success_message": "Order updated successfully"
}
```

**Key transformations:**
- `submit_command: "orders.update"` → `submit_endpoint: "/ui/commands/orders.update"`
- `load_source` → data fetched at descriptor generation time (pre-populated `value` fields)
- `read_only: "orders:notes:edit"` → `read_only: false` (if user has capability) or `true`
- `visibility: "orders:notes:view"` → field omitted (if user lacks capability)

### SectionDescriptor

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Section identifier |
| `title` | string | Section title |
| `layout` | string | Layout type: card, grid, tabs, accordion |
| `columns` | int | Number of grid columns (for grid layout) |
| `fields` | FieldDescriptor[] | Fields in this section |
| `collapsible` | bool | Whether the section can be collapsed |
| `collapsed` | bool | Whether initially collapsed |

### FieldDescriptor

| Field | Type | Description |
|-------|------|-------------|
| `field` | string | Field key (for data binding and form submission) |
| `label` | string | Display label |
| `type` | string | Field type (see [06](06-definition-schema-reference.md)) |
| `read_only` | bool | Whether the field is read-only (resolved from capability expression) |
| `required` | bool | Whether the field is required |
| `validation` | ValidationDescriptor or null | Client-side validation rules |
| `options` | OptionDescriptor[] | Resolved options for select/multi-select types |
| `format` | string | Display format |
| `placeholder` | string | Input placeholder text |
| `help_text` | string | Help text below the field |
| `span` | int | Grid column span |
| `value` | any | Pre-populated value (from load_source or workflow state) |
| `depends_on` | FieldDependencyDescriptor[] | Client-side field dependencies |

### ValidationDescriptor

```json
{
  "min_length": 1,
  "max_length": 500,
  "min": 0.01,
  "max": 999999.99,
  "pattern": "^[A-Z]{2}-\\d{6}$",
  "message": "Must be a valid order number"
}
```

All fields are optional. Only non-null fields are enforced.

### ActionDescriptor

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Action identifier |
| `label` | string | Button/menu text |
| `icon` | string | Icon identifier |
| `style` | string | Visual style: primary, secondary, danger, warning |
| `type` | string | Action type: command, navigate, workflow, form, confirm |
| `enabled` | bool | Whether the action is currently enabled |
| `visible` | bool | Whether the action is currently visible |
| `command_id` | string | Command to execute (for type: command/confirm) |
| `navigate_to` | string | Route to navigate to (for type: navigate) |
| `workflow_id` | string | Workflow to start (for type: workflow) |
| `form_id` | string | Form to open (for type: form) |
| `confirmation` | ConfirmationDescriptor or null | Confirmation dialog |
| `conditions` | ConditionDescriptor[] | Client-side conditions |
| `params` | map | Parameters to pass to the action |

**Note on `enabled` and `visible`:** These are resolved server-side from static
analysis. `conditions` are sent to the frontend for data-dependent evaluation
(e.g., show "Cancel" only when `status == "pending"`).

### ConditionDescriptor

```json
{
  "field": "status",
  "operator": "in",
  "value": ["pending", "confirmed"],
  "effect": "show"
}
```

The frontend evaluates these against the current resource data:
- If effect is "show" and condition is false → hide the action.
- If effect is "hide" and condition is true → hide the action.
- If effect is "enable" and condition is false → disable the action.
- If effect is "disable" and condition is true → disable the action.

### WorkflowDescriptor

Returned by `GET /ui/workflows/{instanceId}`.

```json
{
  "id": "wf-a1b2c3",
  "workflow_id": "orders.approval",
  "name": "Order Approval",
  "status": "active",
  "current_step": {
    "id": "review",
    "name": "Review Order",
    "type": "approval",
    "status": "active",
    "form": {
      "id": "orders.approval_form",
      "title": "Review Order",
      "sections": [...],
      "actions": [
        { "id": "approve", "label": "Approve", "style": "primary", "type": "command", "enabled": true, "visible": true },
        { "id": "reject", "label": "Reject", "style": "danger", "type": "command", "enabled": true, "visible": true }
      ]
    },
    "actions": []
  },
  "steps": [
    { "id": "review", "name": "Review Order", "type": "approval", "status": "active" },
    { "id": "process", "name": "Process Order", "type": "system", "status": "pending" },
    { "id": "notify", "name": "Send Notification", "type": "notification", "status": "pending" },
    { "id": "approved", "name": "Approved", "type": "terminal", "status": "pending" }
  ],
  "history": [
    {
      "step_name": "Review Order",
      "event": "step_entered",
      "actor": "alice@acme-corp.com",
      "timestamp": "2025-01-15T10:30:00Z",
      "comment": ""
    }
  ]
}
```

The workflow descriptor shows:
- The current step with its form (if applicable) and available actions.
- All steps with their current status (pending, active, completed, failed, skipped).
- History of events for audit trail.

---

## DataResponse

Returned by `GET /ui/pages/{pageId}/data` and similar data-fetching endpoints.

```json
{
  "data": {
    "items": [
      {
        "id": "ord-001",
        "order_number": "ORD-2024-001",
        "customer_name": "Acme Corp",
        "status": "pending",
        "total_amount": 1500.00,
        "created_at": "2025-01-15T10:30:00Z"
      },
      {
        "id": "ord-002",
        "order_number": "ORD-2024-002",
        "customer_name": "Globex Inc",
        "status": "confirmed",
        "total_amount": 3200.50,
        "created_at": "2025-01-14T14:20:00Z"
      }
    ],
    "total_count": 142,
    "page": 1,
    "page_size": 25
  },
  "meta": {
    "trace_id": "abc123"
  }
}
```

**Key points:**
- Field names in `items` use UI field names (after field_map renaming).
- Backend field names never appear.
- `total_count` is resolved from the `total_path` in ResponseMapping.
- Pagination metadata is standardized regardless of backend pagination style.

---

## CommandResponse

Returned by `POST /ui/commands/{commandId}`.

**Success:**
```json
{
  "data": {
    "success": true,
    "message": "Order updated successfully",
    "result": {
      "id": "ord-001",
      "order_number": "ORD-2024-001"
    }
  },
  "meta": { "trace_id": "abc123" }
}
```

**Validation Error:**
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "One or more fields are invalid",
    "details": [
      { "field": "shipping_address", "code": "REQUIRED", "message": "Shipping address is required" },
      { "field": "priority", "code": "INVALID_VALUE", "message": "Must be one of: normal, high, urgent" }
    ],
    "trace_id": "abc123"
  }
}
```

---

## SearchResponse

Returned by `GET /ui/search?q=...`.

```json
{
  "data": {
    "results": [
      {
        "id": "ord-001",
        "title": "ORD-2024-001",
        "subtitle": "Acme Corp - $1,500.00",
        "category": "Orders",
        "icon": "shopping_cart",
        "route": "/orders/ord-001",
        "score": 0.95
      },
      {
        "id": "cust-001",
        "title": "Acme Corp",
        "subtitle": "Enterprise Customer",
        "category": "Customers",
        "icon": "business",
        "route": "/customers/cust-001",
        "score": 0.82
      }
    ],
    "total_count": 2,
    "query": "acme"
  },
  "meta": { "trace_id": "abc123" }
}
```

---

## LookupResponse

Returned by `GET /ui/lookups/{lookupId}`.

```json
{
  "data": {
    "options": [
      { "label": "Pending", "value": "pending", "icon": "hourglass" },
      { "label": "Confirmed", "value": "confirmed", "icon": "check" },
      { "label": "Shipped", "value": "shipped", "icon": "local_shipping" },
      { "label": "Cancelled", "value": "cancelled", "icon": "cancel" }
    ]
  },
  "meta": { "trace_id": "abc123" }
}
```

---

## Frontend Contract Guarantees

The frontend can rely on these guarantees:

1. **All descriptor fields documented here will always be present** (possibly null/empty).
   The frontend never needs to check for undefined fields.

2. **Unknown fields in descriptors should be ignored.** The BFF may add new fields
   in minor versions. The frontend should use lenient JSON parsing.

3. **Data field names are stable.** Once a field name appears in a descriptor (e.g.,
   `order_number` in a column), it won't change without a major version bump.

4. **Actions in descriptors are authorized.** If an action is present, the user
   has the capability to execute it. The frontend doesn't need to second-guess.

5. **Page-size and pagination are standardized.** All data endpoints use the same
   pagination model regardless of the underlying backend service.

6. **Error responses use the same envelope.** The frontend has a single error
   handling path for all BFF endpoints.
