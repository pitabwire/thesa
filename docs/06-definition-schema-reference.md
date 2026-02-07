# 06 — Definition Schema Reference

This document provides the complete YAML schema for every definition type. Each type
is documented with all fields, their types, whether they are required, and examples.

For context on how definitions are authored, loaded, and validated, see
[05 — UI Exposure Definitions](05-ui-exposure-definitions.md). For the formal
JSON Schema representation and glossary, see
[23 — Glossary and Appendices](23-glossary-and-appendices.md).

---

## DomainDefinition (Top Level)

The root structure of a definition file.

```yaml
domain: "orders"                    # REQUIRED. Unique domain identifier.
version: "1.0.0"                    # REQUIRED. Semver for change tracking.

navigation:                          # REQUIRED. Menu entry for this domain.
  label: "Orders"                    # REQUIRED. Display label.
  icon: "shopping_cart"              # REQUIRED. Icon identifier.
  order: 10                          # REQUIRED. Sort order in menu (ascending).
  capabilities:                      # REQUIRED. Caps needed to see this domain in menu.
    - "orders:nav:view"
  children:                          # REQUIRED. At least one child navigation item.
    - label: "All Orders"            # REQUIRED.
      icon: "list"                   # Optional.
      route: "/orders"               # REQUIRED. Frontend route path.
      page_id: "orders.list"         # REQUIRED. References a PageDefinition.id.
      capabilities:                  # REQUIRED.
        - "orders:list:view"
      order: 1                       # REQUIRED. Sort order within domain.
      badge:                         # Optional. Count badge on nav item.
        operation_id: "getOrderCount"
        field: "count"
        style: "warning"             # "info", "warning", "danger"

pages: [...]                         # List of PageDefinition.
forms: [...]                         # List of FormDefinition.
commands: [...]                      # List of CommandDefinition.
workflows: [...]                     # List of WorkflowDefinition.
searches: [...]                      # List of SearchDefinition.
lookups: [...]                       # List of LookupDefinition.
```

---

## PageDefinition

Describes a page visible in the UI.

```yaml
id: "orders.list"                    # REQUIRED. Globally unique page ID.
title: "Orders"                      # REQUIRED. Page title shown in header/breadcrumb.
route: "/orders"                     # REQUIRED. Frontend route path.
layout: "list"                       # REQUIRED. One of: "list", "detail", "dashboard", "custom".
capabilities:                        # REQUIRED. Caps needed to access this page.
  - "orders:list:view"
refresh_interval: 30                 # Optional. Auto-refresh in seconds. 0 = disabled.

breadcrumb:                          # Optional. Breadcrumb trail.
  - label: "Home"
    route: "/"
  - label: "Orders"                  # Last item has no route (current page).

table:                               # Required if layout == "list". TableDefinition.
  ...

sections:                            # Optional. List of SectionDefinition.
  ...                                # Used for detail and dashboard layouts.

actions:                             # Optional. Page-level actions (toolbar buttons).
  ...                                # List of ActionDefinition.
```

### Layout Types

| Layout | Description | Typical Content |
|--------|-------------|-----------------|
| `list` | Paginated data table with filters | `table` is required |
| `detail` | Resource detail view | `sections` with fields |
| `dashboard` | Multi-section overview | Multiple `sections` with mixed content |
| `custom` | Freeform layout | `sections` arranged by frontend |

---

## TableDefinition

Describes a data table within a list page.

```yaml
table:
  data_source:                       # REQUIRED. How to fetch the data.
    operation_id: "listOrders"       # Required if type is openapi.
    service_id: "orders-svc"         # Optional. Defaults to domain default service.
    handler: "custom.ListOrders"     # Required if type is sdk (mutually exclusive with operation_id).
    mapping:                         # REQUIRED. Response transformation rules.
      items_path: "data.orders"      # REQUIRED. JSON path to the items array in the backend response.
      total_path: "data.total"       # Optional. JSON path to total count for pagination.
      field_map:                     # Optional. Backend field name → UI field name renaming.
        order_number: "orderNumber"
        created_at: "createdAt"
        customer_name: "customer.name"  # Supports nested paths.

  columns:                           # REQUIRED. At least one column.
    - field: "order_number"          # REQUIRED. Maps to a field in the (renamed) response.
      label: "Order #"              # REQUIRED. Column header label.
      type: "text"                   # REQUIRED. See Column Types below.
      sortable: true                 # Optional. Default: false.
      filterable: false              # Optional. Default: false.
      visible: "always"             # Optional. "always", "desktop", "hidden", or a capability string.
      format: ""                     # Optional. Format string (varies by type).
      width: "150px"                 # Optional. CSS width hint.
      link:                          # Optional. Makes the cell a clickable link.
        route: "/orders/{id}"        #   Route template.
        params:                      #   Map of route params to row field names.
          id: "id"
      status_map:                    # Optional. For type "status": value → style mapping.
        pending: "warning"
        confirmed: "info"
        shipped: "success"
        cancelled: "danger"

  filters:                           # Optional. Filter controls above the table.
    - field: "status"                # REQUIRED. Query parameter name.
      label: "Status"               # REQUIRED. Filter label.
      type: "select"                 # REQUIRED. See Filter Types below.
      operator: "eq"                 # Optional. See Filter Operators below. Default: inferred from type.
      options:                       # Required for select/multi-select types.
        lookup_id: "orders.statuses" # Reference to a LookupDefinition.
        # OR
        static:                      # Inline options.
          - { label: "Pending", value: "pending" }
          - { label: "Confirmed", value: "confirmed" }
      default: "pending"             # Optional. Default filter value.

  row_actions:                       # Optional. Actions per row (dropdown or icon buttons).
    - ... (ActionDefinition)

  bulk_actions:                      # Optional. Actions for selected rows.
    - ... (ActionDefinition)

  default_sort: "created_at"         # Optional. Default sort field.
  sort_dir: "desc"                   # Optional. Default sort direction: "asc" or "desc".
  page_size: 25                      # Optional. Default page size. Range: 1-200.
  selectable: false                  # Optional. Whether rows have checkboxes.
```

### Column Types

| Type | Renders As | Format Example |
|------|-----------|----------------|
| `text` | Plain text | — |
| `number` | Formatted number | `"#,##0"` |
| `currency` | Currency with symbol | `"USD"`, `"EUR"` |
| `date` | Date only | `"yyyy-MM-dd"` |
| `datetime` | Date and time | `"yyyy-MM-dd HH:mm"` |
| `boolean` | Checkmark or icon | — |
| `status` | Badge with color | Uses `status_map` |
| `badge` | Colored badge | Uses `status_map` |
| `link` | Clickable text | Uses `link` definition |
| `image` | Thumbnail | — |
| `progress` | Progress bar | — |

### Filter Types

| Type | Renders As |
|------|-----------|
| `text` | Text input |
| `select` | Single-select dropdown |
| `multi_select` | Multi-select dropdown |
| `date_range` | Two date pickers (from/to) |
| `number_range` | Two number inputs (min/max) |
| `boolean` | Toggle/checkbox |

### Filter Operators

| Operator | Meaning | Applied As |
|----------|---------|------------|
| `eq` | Equals | `?field=value` |
| `neq` | Not equals | `?field_neq=value` |
| `contains` | Contains substring | `?field_contains=value` |
| `gte` | Greater than or equal | `?field_gte=value` |
| `lte` | Less than or equal | `?field_lte=value` |
| `between` | Range (two values) | `?field_from=a&field_to=b` |
| `in` | One of values | `?field=a,b,c` |

---

## SectionDefinition

Describes a section within a page or form.

```yaml
sections:
  - id: "header"                     # REQUIRED. Section ID (unique within page/form).
    title: "Order Information"       # REQUIRED. Section title.
    layout: "grid"                   # REQUIRED. One of: card, grid, tabs, accordion.
    columns: 3                       # Optional. Grid columns (for layout "grid"). Default: 1.
    capabilities:                    # Optional. Caps needed to see this section.
      - "orders:detail:view"
    collapsible: false               # Optional. Whether the section can be collapsed.
    collapsed: false                 # Optional. Whether initially collapsed.
    fields:                          # REQUIRED. List of FieldDefinition.
      - ...
```

### Layout Types for Sections

| Layout | Description |
|--------|-------------|
| `card` | Standard card container with title and border |
| `grid` | Multi-column grid layout (uses `columns` field) |
| `tabs` | Each section becomes a tab |
| `accordion` | Collapsible panels |

---

## FieldDefinition

Describes a single field in a section or form.

```yaml
fields:
  - field: "order_number"            # REQUIRED. Field key (maps to data and input).
    label: "Order Number"            # REQUIRED. Display label.
    type: "text"                     # REQUIRED. See Field Types below.
    read_only: "true"                # Optional. "true", "false", or capability string.
                                     #   "true" — always read-only.
                                     #   "false" — always editable (default).
                                     #   "orders:edit:execute" — read-only UNLESS user has this cap.
    required: false                  # Optional. Whether field is required for form submission.
    validation:                      # Optional. Validation rules.
      min_length: 1
      max_length: 500
      min: 0
      max: 999999.99
      pattern: "^[A-Z]{2}-\\d{6}$"
      message: "Must be a valid order number (e.g., ORD-123456)"
    lookup:                          # Optional. Reference data for select/reference types.
      lookup_id: "customers.search"
      # OR
      static:
        - { label: "Normal", value: "normal" }
        - { label: "High", value: "high" }
    visibility: "orders:notes:view"  # Optional. Capability required to see this field.
                                     #   If absent, field is always visible (subject to section caps).
    format: "USD"                    # Optional. Display format (for read-only display).
    placeholder: "Enter address..."  # Optional. Placeholder text for input fields.
    help_text: "Enter the full..."   # Optional. Help text shown below the field.
    span: 2                          # Optional. Grid column span (for grid layouts).
    depends_on:                      # Optional. Field dependencies.
      - field: "country"
        condition: "equals"          # "equals", "not_empty", "in"
        value: "US"                  # Required for "equals" and "in"
```

### Field Types

| Type | Input Renders As | Display Renders As |
|------|-----------------|-------------------|
| `text` | Text input | Plain text |
| `number` | Number input | Formatted number |
| `currency` | Number input with currency symbol | Formatted currency |
| `date` | Date picker | Formatted date |
| `datetime` | Date+time picker | Formatted datetime |
| `select` | Dropdown | Selected label |
| `multi-select` | Multi-select dropdown | Comma-separated labels |
| `textarea` | Multi-line text area | Multi-line text |
| `rich-text` | Rich text editor | Rendered rich text |
| `boolean` | Toggle/checkbox | Check/cross icon |
| `reference` | Autocomplete/search with lookup | Linked text |
| `file` | File upload | File link/preview |
| `hidden` | Not rendered | Not rendered |
| `status` | (Read-only) Status badge | Status badge |

### Read-Only Resolution

The `read_only` field uses a special resolution logic:

| Value | Meaning |
|-------|---------|
| `"true"` | Always read-only |
| `"false"` or absent | Editable (in forms) |
| `"{capability}"` | Read-only UNLESS the user has the given capability |

This allows fine-grained field-level editability. Example:

```yaml
- field: "internal_notes"
  read_only: "orders:notes:edit"
  # Users with orders:notes:edit can edit this field.
  # Users without it see the field as read-only.
  # Users without orders:notes:view don't see the field at all (controlled by visibility).
```

### Visibility Resolution

The `visibility` field controls whether the field appears at all:

| Value | Meaning |
|-------|---------|
| Absent | Always visible (subject to section-level capabilities) |
| `"{capability}"` | Visible only if the user has the given capability |

### Field Dependencies

Dependencies control when a field is shown or enabled based on another field's value:

```yaml
- field: "state"
  type: "select"
  lookup:
    lookup_id: "us_states"
  depends_on:
    - field: "country"
      condition: "equals"
      value: "US"
```

When `country` is not "US", the `state` field is hidden or disabled.

---

## FormDefinition

Describes an input form.

```yaml
forms:
  - id: "orders.edit_form"           # REQUIRED. Globally unique form ID.
    title: "Edit Order"              # REQUIRED. Form title.
    capabilities:                    # REQUIRED. Caps needed to access this form.
      - "orders:edit:execute"
    submit_command: "orders.update"  # REQUIRED. Command ID to invoke on form submission.
    load_source:                     # Optional. Data source to pre-populate the form.
      operation_id: "getOrder"
      service_id: "orders-svc"
      mapping:
        items_path: "data"           # JSON path to the resource in the response.
    success_route: "/orders/{id}"    # Optional. Route to navigate to after successful submit.
    success_message: "Order updated" # Optional. Toast message after successful submit.
    sections:                        # REQUIRED. At least one section with fields.
      - ... (SectionDefinition)
```

---

## ActionDefinition

Describes a UI action (button, menu item, icon button).

```yaml
actions:
  - id: "orders.edit_action"         # REQUIRED. Action ID.
    label: "Edit"                    # REQUIRED. Button/menu label.
    icon: "edit"                     # Optional. Icon identifier.
    style: "primary"                 # Optional. One of: primary, secondary, danger, warning.
    capabilities:                    # REQUIRED. Caps needed to see this action.
      - "orders:edit:execute"
    type: "form"                     # REQUIRED. See Action Types below.
    command_id: "orders.update"      # Required if type == "command".
    navigate_to: "/orders/{id}"      # Required if type == "navigate".
    workflow_id: "orders.approval"   # Required if type == "workflow".
    form_id: "orders.edit_form"      # Required if type == "form".
    confirmation:                    # Optional. Show confirmation dialog before executing.
      title: "Cancel Order?"
      message: "This action cannot be undone."
      confirm: "Yes, Cancel"
      cancel: "Keep Order"           # Optional. Default: "Cancel".
      style: "danger"                # Optional. Dialog style: "danger", "warning".
    conditions:                      # Optional. Data-dependent visibility/enablement.
      - field: "status"
        operator: "in"               # "eq", "neq", "in", "not_in", "empty", "not_empty"
        value: "pending,confirmed"   # For "in"/"not_in": comma-separated values. For "eq"/"neq": single value.
        effect: "show"               # "show", "hide", "enable", "disable"
    params:                          # Optional. Static or dynamic params passed to the action.
      id: "{id}"                     # Interpolated from current row/resource data.
```

### Action Types

| Type | Behavior |
|------|----------|
| `command` | Execute a command (POST /ui/commands/{commandId}). Uses `command_id`. |
| `navigate` | Navigate to a route. Uses `navigate_to`. |
| `workflow` | Start a workflow (POST /ui/workflows/{workflowId}/start). Uses `workflow_id`. |
| `form` | Open a form dialog/page. Uses `form_id`. |
| `confirm` | Show a confirmation dialog, then execute a command. Uses `command_id` + `confirmation`. |

### Conditions

Conditions make actions data-dependent. They are evaluated client-side by the
frontend based on the current resource data.

```yaml
conditions:
  - field: "status"
    operator: "eq"
    value: "pending"
    effect: "show"             # Show only when status == "pending"

  - field: "total_amount"
    operator: "gte"
    value: 10000
    effect: "disable"          # Disable when total >= 10000 (requires approval workflow instead)
```

When multiple conditions are present, all must be satisfied (AND logic).

---

## CommandDefinition

Describes a mutable operation. See [10 — Command & Action Model](10-command-and-action-model.md)
for execution details.

```yaml
commands:
  - id: "orders.update"             # REQUIRED. Globally unique command ID.
    capabilities:                    # REQUIRED.
      - "orders:edit:execute"
    operation:                       # REQUIRED. Backend binding.
      type: "openapi"               # REQUIRED. "openapi" or "sdk".
      operation_id: "updateOrder"    # Required if type == "openapi".
      service_id: "orders-svc"      # Optional.
      handler: ""                    # Required if type == "sdk".
    input:                           # REQUIRED. Input mapping rules.
      path_params:                   # Optional. Path parameter sources.
        orderId: "route.id"
      query_params:                  # Optional. Query parameter sources.
        expand: "'items,customer'"   # Literal values use single quotes.
      header_params:                 # Optional. Custom header sources.
        X-Custom: "context.tenant_id"
      body_mapping: "projection"     # REQUIRED. "passthrough", "template", or "projection".
      body_template:                 # Required if body_mapping == "template".
        customerId: "input.customer_id"
        reason: "input.reason"
        updatedBy: "context.subject_id"
      field_projection:              # Required if body_mapping == "projection".
        customerId: "input.customer_id"
        shippingAddress: "input.shipping_address"
    output:                          # REQUIRED. Output mapping rules.
      type: "full"                   # "passthrough", "full", "project". See Output Types below.
      fields:                        # Required if type == "project".
        id: "data.id"
        order_number: "data.orderNumber"
      error_map:                     # Optional. Backend error code → UI message.
        ORDER_NOT_FOUND: "This order no longer exists"
        INVALID_STATUS: "Cannot edit in current status"
      success_message: "Order updated" # Optional.
    idempotency:                     # Optional.
      key_source: "header"           # Source for idempotency key. "header" reads Idempotency-Key header.
      ttl: 3600                      # Time-to-live in seconds for idempotency records.
    rate_limit:                      # Optional.
      max_requests: 10
      window: "1m"
      scope: "user"                  # "user", "tenant", "global"
```

### Body Mapping Strategies

| Strategy | Description | When to Use |
|----------|-------------|-------------|
| `passthrough` | Send the frontend's `input` as-is to the backend body | Backend accepts the same shape as the frontend sends |
| `template` | Construct body from `body_template`, substituting expressions | Backend expects a different structure; need to inject context values |
| `projection` | Map selected frontend fields to backend fields via `field_projection` | Field-by-field renaming/selection |

### Output Types

| Type | Description | When to Use |
|------|-------------|-------------|
| `passthrough` | Return the backend response body as-is | Backend response already matches the frontend contract |
| `full` | Return the full backend response (equivalent to `passthrough`) | Commonly used when the response shape is already suitable |
| `project` | Extract and rename specific fields via `fields` map | Frontend needs a subset of backend fields with different names |

### Source Expression Reference

Expressions used in path_params, query_params, body_template, and field_projection:

| Expression | Resolves To |
|------------|-------------|
| `input.{field}` | Field from the user's input payload |
| `route.{param}` | Route parameter from the current page URL |
| `context.subject_id` | Authenticated user's ID |
| `context.tenant_id` | Current tenant ID |
| `context.partition_id` | Current partition ID |
| `context.email` | User's email |
| `workflow.{field}` | Field from workflow state (only in workflow context) |
| `'{literal}'` | A literal string value |
| `{number}` | A literal numeric value |

---

## WorkflowDefinition

Describes a multi-step process. See [11 — Workflow Engine](11-workflow-engine.md)
for execution details.

```yaml
workflows:
  - id: "orders.approval"           # REQUIRED.
    name: "Order Approval"          # REQUIRED. Human-readable name.
    capabilities:                    # REQUIRED. Caps needed to start this workflow.
      - "orders:approve:execute"
    initial_step: "review"           # REQUIRED. ID of the first step.
    timeout: "72h"                   # Optional. Overall workflow timeout (Go duration: "24h", "30m", "72h").
    on_timeout: "expired"            # Optional. Step ID to transition to on timeout.

    steps:                           # REQUIRED. At least two steps (one initial + one terminal).
      - id: "review"                 # REQUIRED. Step ID (unique within workflow).
        name: "Review Order"         # REQUIRED. Human-readable step name.
        type: "approval"             # REQUIRED. See Step Types below.
        capabilities:                # Optional. Caps needed to act on this step.
          - "orders:approve:execute"
        form_id: "orders.approval_form"  # Optional. Form to display for this step.
        operation:                   # Optional. Backend operation for system steps.
          type: "openapi"
          operation_id: "confirmOrder"
          service_id: "orders-svc"
        input:                       # Optional. Input mapping for the operation.
          path_params:
            orderId: "workflow.order_id"
          body_mapping: "template"
          body_template:
            approvedBy: "workflow.approved_by"
        output:                      # Optional. Output mapping.
          type: "project"
          fields:
            confirmed_at: "data.confirmedAt"
        timeout: "24h"              # Optional. Per-step timeout (Go duration format).
        on_timeout: "expired"        # Optional. Step ID to transition to on step timeout.
        assignee:                    # Optional. Who is responsible for this step.
          type: "role"               # "role", "user", "group", "dynamic"
          value: "order_approver"

    transitions:                     # REQUIRED. At least one transition.
      - from: "review"              # REQUIRED. Source step ID.
        to: "process"               # REQUIRED. Target step ID.
        event: "approved"            # REQUIRED. See Transition Events below.
        condition: ""                # Optional. Expression evaluated against workflow state.
        guard: ""                    # Optional. Capability required to trigger this transition.
```

### Step Types

| Type | User Interaction | Backend Invocation |
|------|-----------------|-------------------|
| `human` | Required (user provides input via form) | Optional |
| `action` | Required (user performs action) | Optional |
| `approval` | Required (approve/reject) | Optional |
| `system` | None (automatic) | Required |
| `wait` | None (waits for event/time) | None |
| `notification` | None (automatic, best-effort) | Required |
| `terminal` | None (end state) | None |

> **Note:** The workflow engine treats `system` and `notification` as auto-executable
> steps (they run without user interaction). All other types (`human`, `action`,
> `approval`, `wait`) require an explicit user-initiated advance event.

### Transition Events

| Event | Meaning |
|-------|---------|
| `completed` | Step finished successfully |
| `approved` | Approval step: approved |
| `rejected` | Approval step: rejected |
| `timeout` | Step or workflow timed out |
| `error` | Step execution failed |
| `cancelled` | Workflow cancelled by user |

---

## SearchDefinition

```yaml
searches:
  - id: "orders.search"             # REQUIRED.
    domain: "orders"                 # REQUIRED. Category label in search results.
    capabilities:                    # REQUIRED.
      - "orders:search:execute"
    operation:                       # REQUIRED.
      type: "openapi"
      operation_id: "searchOrders"
      service_id: "orders-svc"
    result_mapping:                  # REQUIRED.
      items_path: "data.results"     # REQUIRED. JSON path to results array.
      title_field: "orderNumber"     # REQUIRED. Field to use as result title.
      subtitle_field: "customerName" # Optional. Field for subtitle.
      category_field: "status"       # Optional. Field for category tag.
      icon_field: ""                 # Optional. Field for result icon.
      route: "/orders/{id}"          # REQUIRED. Navigation target template.
      id_field: "id"                 # REQUIRED. Field for resource ID.
    weight: 10                       # Optional. Ranking weight (higher = ranked higher).
    max_results: 5                   # Optional. Max results from this provider.
```

---

## LookupDefinition

```yaml
lookups:
  - id: "customers.search"          # REQUIRED.
    operation:                       # Required for dynamic lookups (backend-fetched).
      type: "openapi"               #   Can be omitted for static-only lookups
      operation_id: "searchCustomers"#   whose options are defined inline in forms/filters.
      service_id: "customers-svc"
    label_field: "name"              # REQUIRED. Field to display as option label.
    value_field: "id"                # REQUIRED. Field to use as option value.
    search_field: "query"            # Optional. Query parameter name for search-as-you-type.
    cache:                           # Optional.
      ttl: 300                       # Cache time-to-live in seconds.
      scope: "global"                # "tenant" or "global".
```
