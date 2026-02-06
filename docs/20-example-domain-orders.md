# 20 — Example Domain: Orders

This document provides a complete worked example for the Orders domain, showing
every definition type with real values, the associated capabilities, the referenced
OpenAPI operations, and the expected frontend behavior.

---

## Domain Overview

The Orders domain manages the lifecycle of customer orders. It provides:

- A list page showing all orders with filtering and sorting.
- A detail page showing order information with conditional actions.
- An edit form for modifying order details.
- Commands for updating and cancelling orders.
- An approval workflow for pending orders.
- A cancellation workflow with reason tracking.
- Global search integration.
- Lookup definitions for status options and customer search.

---

## Capability Namespace: `orders`

| Capability | Who Has It | Purpose |
|-----------|-----------|---------|
| `orders:nav:view` | All order-related roles | See Orders in navigation |
| `orders:list:view` | viewer, manager, admin | See the orders list page |
| `orders:detail:view` | viewer, manager, admin | See order detail pages |
| `orders:create:view` | manager, admin | See the create order page |
| `orders:edit:execute` | manager, admin | Execute the update command |
| `orders:cancel:execute` | manager, admin | Execute the cancel command |
| `orders:approve:execute` | approver, admin | Execute the approval workflow |
| `orders:export:execute` | manager, admin | Execute bulk export |
| `orders:line_items:view` | viewer, manager, admin | See the line items section |
| `orders:notes:view` | manager, admin | See the internal notes section |
| `orders:notes:edit` | manager, admin | Edit the notes field |
| `orders:sensitive:view` | admin | See sensitive financial data |
| `orders:search:execute` | viewer, manager, admin | Results appear in global search |

### Role → Capability Mapping (in policy engine)

```yaml
roles:
  order_viewer:
    capabilities:
      - "orders:nav:view"
      - "orders:list:view"
      - "orders:detail:view"
      - "orders:line_items:view"
      - "orders:search:execute"

  order_manager:
    capabilities:
      - "orders:nav:view"
      - "orders:list:view"
      - "orders:detail:view"
      - "orders:create:view"
      - "orders:edit:execute"
      - "orders:cancel:execute"
      - "orders:export:execute"
      - "orders:line_items:view"
      - "orders:notes:view"
      - "orders:notes:edit"
      - "orders:search:execute"

  order_approver:
    capabilities:
      - "orders:nav:view"
      - "orders:list:view"
      - "orders:detail:view"
      - "orders:approve:execute"
      - "orders:line_items:view"
      - "orders:search:execute"

  order_admin:
    capabilities:
      - "orders:*"
```

---

## Referenced OpenAPI Operations

These operations must exist in the `orders-svc` OpenAPI spec:

| operationId | Method | Path | Purpose |
|-------------|--------|------|---------|
| `listOrders` | GET | /api/v1/orders | Paginated order list |
| `getOrder` | GET | /api/v1/orders/{orderId} | Single order details |
| `createOrder` | POST | /api/v1/orders | Create new order |
| `updateOrder` | PATCH | /api/v1/orders/{orderId} | Update order fields |
| `cancelOrder` | POST | /api/v1/orders/{orderId}/cancel | Cancel order |
| `confirmOrder` | POST | /api/v1/orders/{orderId}/confirm | Confirm (approve) order |
| `searchOrders` | GET | /api/v1/orders/search | Full-text search |
| `getOrderStatuses` | GET | /api/v1/orders/statuses | Status enum values |
| `exportOrders` | POST | /api/v1/orders/export | Bulk export |

---

## Complete Definition File

```yaml
domain: "orders"
version: "2.0.0"

navigation:
  label: "Orders"
  icon: "shopping_cart"
  order: 10
  capabilities: ["orders:nav:view"]
  children:
    - label: "All Orders"
      icon: "list"
      route: "/orders"
      page_id: "orders.list"
      capabilities: ["orders:list:view"]
      order: 1
      badge:
        operation_id: "listOrders"
        field: "meta.pending_count"
        style: "warning"
    - label: "Create Order"
      icon: "add_circle"
      route: "/orders/new"
      page_id: "orders.create"
      capabilities: ["orders:create:view"]
      order: 2

# ── PAGES ──────────────────────────────────────────────────────────

pages:
  - id: "orders.list"
    title: "Orders"
    route: "/orders"
    layout: "list"
    capabilities: ["orders:list:view"]
    refresh_interval: 30

    table:
      data_source:
        operation_id: "listOrders"
        service_id: "orders-svc"
        mapping:
          items_path: "data.orders"
          total_path: "meta.total"
          field_map:
            order_number: "orderNumber"
            customer_name: "customer.name"
            total_amount: "totalAmount"
            created_at: "createdAt"
            updated_at: "updatedAt"

      columns:
        - field: "order_number"
          label: "Order #"
          type: "link"
          sortable: true
          width: "150px"
          link:
            route: "/orders/{id}"
            params: { id: "id" }

        - field: "customer_name"
          label: "Customer"
          type: "text"
          sortable: true

        - field: "status"
          label: "Status"
          type: "status"
          sortable: true
          filterable: true
          status_map:
            draft: "default"
            pending: "warning"
            confirmed: "info"
            processing: "info"
            shipped: "success"
            delivered: "success"
            cancelled: "danger"
            refunded: "danger"

        - field: "total_amount"
          label: "Total"
          type: "currency"
          format: "USD"
          sortable: true
          visible: "orders:sensitive:view"

        - field: "created_at"
          label: "Created"
          type: "datetime"
          sortable: true

      filters:
        - field: "status"
          label: "Status"
          type: "select"
          operator: "eq"
          options:
            lookup_id: "orders.statuses"

        - field: "created_at"
          label: "Date Range"
          type: "date-range"
          operator: "between"

        - field: "q"
          label: "Search"
          type: "text"
          operator: "contains"

      row_actions:
        - id: "orders.view_row"
          label: "View"
          icon: "visibility"
          type: "navigate"
          navigate_to: "/orders/{id}"
          capabilities: ["orders:detail:view"]

        - id: "orders.cancel_row"
          label: "Cancel"
          icon: "cancel"
          style: "danger"
          type: "command"
          command_id: "orders.cancel"
          capabilities: ["orders:cancel:execute"]
          confirmation:
            title: "Cancel Order?"
            message: "Cancel order {order_number}? This cannot be undone."
            confirm: "Yes, Cancel"
            style: "danger"
          conditions:
            - field: "status"
              operator: "in"
              value: ["pending", "confirmed"]
              effect: "show"

      bulk_actions:
        - id: "orders.bulk_export"
          label: "Export Selected"
          icon: "download"
          type: "command"
          command_id: "orders.export"
          capabilities: ["orders:export:execute"]

      default_sort: "created_at"
      sort_dir: "desc"
      page_size: 25
      selectable: true

    actions:
      - id: "orders.create_action"
        label: "New Order"
        icon: "add"
        style: "primary"
        type: "navigate"
        navigate_to: "/orders/new"
        capabilities: ["orders:create:view"]

  - id: "orders.detail"
    title: "Order Details"
    route: "/orders/{id}"
    layout: "detail"
    capabilities: ["orders:detail:view"]

    breadcrumb:
      - label: "Orders"
        route: "/orders"
      - label: "{order_number}"

    sections:
      - id: "header"
        title: "Order Information"
        layout: "grid"
        columns: 3
        fields:
          - field: "order_number"
            label: "Order Number"
            type: "text"
            read_only: "true"
          - field: "status"
            label: "Status"
            type: "status"
            read_only: "true"
          - field: "customer_name"
            label: "Customer"
            type: "text"
            read_only: "true"
          - field: "total_amount"
            label: "Total Amount"
            type: "currency"
            format: "USD"
            read_only: "true"
            visibility: "orders:sensitive:view"
          - field: "shipping_address"
            label: "Shipping Address"
            type: "textarea"
            read_only: "true"
          - field: "created_at"
            label: "Created"
            type: "datetime"
            read_only: "true"

      - id: "notes"
        title: "Internal Notes"
        layout: "card"
        capabilities: ["orders:notes:view"]
        collapsible: true
        collapsed: true
        fields:
          - field: "internal_notes"
            label: "Notes"
            type: "rich-text"
            read_only: "orders:notes:edit"

    actions:
      - id: "orders.edit_action"
        label: "Edit"
        icon: "edit"
        style: "secondary"
        type: "form"
        form_id: "orders.edit_form"
        capabilities: ["orders:edit:execute"]
        conditions:
          - field: "status"
            operator: "in"
            value: ["draft", "pending", "confirmed"]
            effect: "show"

      - id: "orders.approve_action"
        label: "Approve"
        icon: "check_circle"
        style: "primary"
        type: "workflow"
        workflow_id: "orders.approval"
        capabilities: ["orders:approve:execute"]
        conditions:
          - field: "status"
            operator: "eq"
            value: "pending"
            effect: "show"

      - id: "orders.cancel_action"
        label: "Cancel Order"
        icon: "cancel"
        style: "danger"
        type: "workflow"
        workflow_id: "orders.cancellation"
        capabilities: ["orders:cancel:execute"]
        confirmation:
          title: "Cancel Order?"
          message: "This will cancel order {order_number}. This action cannot be undone."
          confirm: "Yes, Cancel"
          cancel: "Keep Order"
          style: "danger"
        conditions:
          - field: "status"
            operator: "in"
            value: ["pending", "confirmed"]
            effect: "show"

# ── FORMS ──────────────────────────────────────────────────────────

forms:
  - id: "orders.edit_form"
    title: "Edit Order"
    capabilities: ["orders:edit:execute"]
    submit_command: "orders.update"
    load_source:
      operation_id: "getOrder"
      service_id: "orders-svc"
      mapping:
        items_path: "data"
    success_route: "/orders/{id}"
    success_message: "Order updated successfully"

    sections:
      - id: "details"
        title: "Order Details"
        layout: "grid"
        columns: 2
        fields:
          - field: "customer_id"
            label: "Customer"
            type: "reference"
            required: true
            lookup:
              lookup_id: "customers.search"
            span: 2

          - field: "shipping_address"
            label: "Shipping Address"
            type: "textarea"
            required: true
            validation:
              max_length: 500
            span: 2

          - field: "priority"
            label: "Priority"
            type: "select"
            required: true
            lookup:
              static:
                - { label: "Normal", value: "normal" }
                - { label: "High", value: "high" }
                - { label: "Urgent", value: "urgent" }

          - field: "internal_notes"
            label: "Internal Notes"
            type: "textarea"
            required: false
            visibility: "orders:notes:edit"
            span: 2

  - id: "orders.approval_form"
    title: "Review Order for Approval"
    capabilities: ["orders:approve:execute"]
    submit_command: "orders.approve_submit"
    sections:
      - id: "review"
        title: "Approval Decision"
        layout: "card"
        fields:
          - field: "approval_notes"
            label: "Approval Notes"
            type: "textarea"
            required: false
            placeholder: "Add any notes about this approval..."
            help_text: "These notes will be visible in the order's audit trail."

  - id: "orders.cancel_form"
    title: "Cancel Order"
    capabilities: ["orders:cancel:execute"]
    submit_command: "orders.cancel"
    sections:
      - id: "reason"
        title: "Cancellation Reason"
        layout: "card"
        fields:
          - field: "reason"
            label: "Reason for Cancellation"
            type: "select"
            required: true
            lookup:
              static:
                - { label: "Customer Request", value: "customer_request" }
                - { label: "Out of Stock", value: "out_of_stock" }
                - { label: "Payment Failed", value: "payment_failed" }
                - { label: "Duplicate Order", value: "duplicate" }
                - { label: "Other", value: "other" }
          - field: "notes"
            label: "Additional Notes"
            type: "textarea"
            required: false
            depends_on:
              - field: "reason"
                condition: "equals"
                value: "other"

# ── COMMANDS ───────────────────────────────────────────────────────

commands:
  - id: "orders.update"
    capabilities: ["orders:edit:execute"]
    operation:
      type: "openapi"
      operation_id: "updateOrder"
      service_id: "orders-svc"
    input:
      path_params:
        orderId: "route.id"
      body_mapping: "projection"
      field_projection:
        customerId: "input.customer_id"
        shippingAddress: "input.shipping_address"
        priority: "input.priority"
        internalNotes: "input.internal_notes"
    output:
      type: "project"
      fields:
        id: "data.id"
        order_number: "data.orderNumber"
      error_map:
        ORDER_NOT_FOUND: "This order no longer exists."
        INVALID_STATUS: "This order cannot be edited in its current status."
        VALIDATION_ERROR: "Please check your input and try again."
      success_message: "Order updated successfully"
    idempotency:
      key_source: "header:Idempotency-Key"
      ttl: "24h"

  - id: "orders.cancel"
    capabilities: ["orders:cancel:execute"]
    operation:
      type: "openapi"
      operation_id: "cancelOrder"
      service_id: "orders-svc"
    input:
      path_params:
        orderId: "input.order_id"
      body_mapping: "template"
      body_template:
        reason: "input.reason"
        notes: "input.notes"
        cancelledBy: "context.subject_id"
    output:
      type: "envelope"
      error_map:
        ORDER_NOT_FOUND: "This order no longer exists."
        ALREADY_CANCELLED: "This order has already been cancelled."
        CANNOT_CANCEL: "This order cannot be cancelled in its current status."
      success_message: "Order cancelled successfully"

  - id: "orders.export"
    capabilities: ["orders:export:execute"]
    operation:
      type: "openapi"
      operation_id: "exportOrders"
      service_id: "orders-svc"
    input:
      body_mapping: "template"
      body_template:
        orderIds: "input.selected_ids"
        format: "'csv'"
        requestedBy: "context.subject_id"
    output:
      type: "project"
      fields:
        download_url: "data.downloadUrl"
      success_message: "Export started. Download will be available shortly."

# ── WORKFLOWS ──────────────────────────────────────────────────────

workflows:
  - id: "orders.approval"
    name: "Order Approval"
    capabilities: ["orders:approve:execute"]
    initial_step: "review"
    timeout: "72h"
    on_timeout: "expired"

    steps:
      - id: "review"
        name: "Review Order"
        type: "approval"
        capabilities: ["orders:approve:execute"]
        form_id: "orders.approval_form"
        timeout: "48h"
        on_timeout: "escalated"
        assignee:
          type: "role"
          value: "order_approver"

      - id: "process"
        name: "Process Approved Order"
        type: "system"
        operation:
          type: "openapi"
          operation_id: "confirmOrder"
          service_id: "orders-svc"
        input:
          path_params:
            orderId: "workflow.order_id"
          body_mapping: "template"
          body_template:
            approvedBy: "workflow.approved_by"
            approvalNotes: "workflow.approval_notes"

      - id: "notify"
        name: "Send Notification"
        type: "notification"
        operation:
          type: "sdk"
          handler: "notifications.SendOrderApproved"
        input:
          body_mapping: "template"
          body_template:
            orderId: "workflow.order_id"
            customerEmail: "workflow.customer_email"

      - id: "approved"
        name: "Approved"
        type: "terminal"

      - id: "rejected"
        name: "Rejected"
        type: "terminal"

      - id: "escalated"
        name: "Escalated"
        type: "action"
        capabilities: ["orders:approve:execute"]
        form_id: "orders.approval_form"
        assignee:
          type: "role"
          value: "order_admin"

      - id: "expired"
        name: "Expired"
        type: "terminal"

    transitions:
      - { from: "review", to: "process", event: "approved" }
      - { from: "review", to: "rejected", event: "rejected" }
      - { from: "review", to: "escalated", event: "timeout" }
      - { from: "escalated", to: "process", event: "approved" }
      - { from: "escalated", to: "rejected", event: "rejected" }
      - { from: "escalated", to: "expired", event: "timeout" }
      - { from: "process", to: "notify", event: "completed" }
      - { from: "process", to: "rejected", event: "error" }
      - { from: "notify", to: "approved", event: "completed" }
      - { from: "notify", to: "approved", event: "error" }

  - id: "orders.cancellation"
    name: "Order Cancellation"
    capabilities: ["orders:cancel:execute"]
    initial_step: "confirm"
    timeout: "1h"

    steps:
      - id: "confirm"
        name: "Confirm Cancellation"
        type: "action"
        form_id: "orders.cancel_form"
        capabilities: ["orders:cancel:execute"]

      - id: "execute"
        name: "Execute Cancellation"
        type: "system"
        operation:
          type: "openapi"
          operation_id: "cancelOrder"
          service_id: "orders-svc"
        input:
          path_params:
            orderId: "workflow.order_id"
          body_mapping: "template"
          body_template:
            reason: "workflow.reason"
            notes: "workflow.notes"
            cancelledBy: "workflow.cancelled_by"

      - id: "cancelled"
        name: "Cancelled"
        type: "terminal"

      - id: "failed"
        name: "Failed"
        type: "terminal"

    transitions:
      - { from: "confirm", to: "execute", event: "completed" }
      - { from: "execute", to: "cancelled", event: "completed" }
      - { from: "execute", to: "failed", event: "error" }

# ── SEARCHES ───────────────────────────────────────────────────────

searches:
  - id: "orders.search"
    domain: "Orders"
    capabilities: ["orders:search:execute"]
    operation:
      type: "openapi"
      operation_id: "searchOrders"
      service_id: "orders-svc"
    result_mapping:
      items_path: "data.results"
      title_field: "orderNumber"
      subtitle_field: "customerName"
      category_field: "status"
      route: "/orders/{id}"
      id_field: "id"
    weight: 10
    max_results: 5

# ── LOOKUPS ────────────────────────────────────────────────────────

lookups:
  - id: "orders.statuses"
    operation:
      type: "openapi"
      operation_id: "getOrderStatuses"
      service_id: "orders-svc"
    label_field: "label"
    value_field: "code"
    cache:
      ttl: "5m"
      scope: "global"
```

---

## What Each Role Sees

### order_viewer

- Navigation: "All Orders" (no "Create Order")
- List page: all columns except "Total" (lacks `orders:sensitive:view`)
- Detail page: header section + line items. No notes section, no actions.
- No commands available.
- Search results include orders.

### order_manager

- Navigation: "All Orders" + "Create Order"
- List page: all columns except "Total" (lacks `orders:sensitive:view`)
- Detail page: all sections including notes (editable). Edit + Cancel actions.
- Commands: update, cancel, export.
- No approve action.

### order_approver

- Navigation: "All Orders" (no "Create Order")
- List page: all columns except "Total". No row cancel action.
- Detail page: header + line items. Approve action on pending orders.
- No edit, cancel, or export.

### order_admin

- Everything. All columns, all sections, all actions, all commands.
- "Total" column visible.
- Can approve, edit, cancel, export.
