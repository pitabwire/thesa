# 7. Dynamic UI Engine

## What Is the UI Engine?

The UI Engine is the heart of Thesa UI. It is the system that reads BFF descriptions (JSON data describing what a page should look like) and converts them into actual Flutter widgets that appear on screen.

Think of it this way:

- **Traditional app**: A developer writes Flutter code that says "put a table here, a form there, a button here."
- **Thesa UI**: The BFF sends a JSON message that says "put a table here, a form there, a button here." The UI Engine reads that message and builds the same widgets automatically.

The developer does not write page layouts. The BFF writes them. The UI Engine is the translator.

---

## The Restaurant Kitchen Analogy

Imagine a restaurant kitchen that works from recipe cards:

1. The **waiter** (Presentation Layer) takes an order: "Customer wants the Orders page"
2. The **kitchen manager** (Page Renderer) looks up the recipe card (Page Descriptor) from the pantry (Cache)
3. The recipe card lists ingredients: "1 search bar, 1 filter panel, 1 data table with 5 columns, 2 action buttons"
4. The **line cooks** (Component Builders) each prepare their item according to the recipe
5. The **plate** (Widget Tree) is assembled and sent to the customer

If the recipe changes tomorrow (the BFF sends a different descriptor), the kitchen produces a different dish without any retraining.

---

## Page Rendering Pipeline

When a user navigates to a page, the following steps happen in order:

### Step 1: Receive the Page Descriptor

The `PageRenderer` widget receives a `PageDescriptor` — a Dart object parsed from the BFF's JSON response. This descriptor contains everything needed to build the page:

```json
{
  "pageId": "orders-list",
  "title": "Orders",
  "layout": { "type": "stack", "direction": "vertical" },
  "components": [ ... ],
  "actions": [ ... ]
}
```

- **`pageId`**: A unique identifier for this page
- **`title`**: The text shown in the page header and breadcrumbs
- **`layout`**: How to arrange the components (vertical stack, grid, tabs, etc.)
- **`components`**: A list of UI elements to render (tables, forms, search bars, etc.)
- **`actions`**: Page-level buttons (like "Create New Order") that appear in the header

### Step 2: Resolve the Layout

The `LayoutResolver` reads the `layout` field and determines how to arrange the components:

| Layout Type | What It Does | Example |
|---|---|---|
| `stack` (vertical) | Places components one below the other | A search bar above a table |
| `stack` (horizontal) | Places components side by side | A sidebar next to a detail panel |
| `grid` | Places components in a responsive grid | Dashboard cards in 2-4 columns |
| `tabs` | Places components in tabbed sections | "Active Orders" tab, "Completed Orders" tab |
| `split` | Splits the screen into resizable panes | A list on the left, details on the right |

### Step 3: Iterate Over Components

For each component in the descriptor, the renderer asks: "What widget should I use?"

This is where the **Component Registry**, **Plugin Registry**, and **Permission Filter** work together:

```
For each component in the page descriptor:

  1. CHECK PERMISSION
     → Is component.allowed == true?
     → If NO → skip this component entirely (do not render it)
     → If YES → proceed

  2. CHECK PLUGIN REGISTRY
     → Has a plugin registered a custom widget for this component type?
     → If YES → use the plugin's widget
     → If NO → proceed to step 3

  3. CHECK COMPONENT REGISTRY
     → Is this a known built-in component type?
     → If YES → use the built-in widget builder
     → If NO → render an "Unknown Component" placeholder

  4. RESOLVE SCHEMAS
     → If the component references a schema (e.g., a form references "schemas/order-create"),
       fetch the schema from schemaProvider and pass it to the widget builder

  5. BUILD THE WIDGET
     → The widget builder receives the ComponentDescriptor and returns a Flutter widget
```

### Step 4: Assemble the Widget Tree

All built widgets are arranged according to the layout type and returned as a single widget tree. The Presentation Layer displays this tree on screen.

---

## The Component Registry

The Component Registry is a lookup table. Given a component type string, it returns a function that builds the corresponding widget.

### Built-In Components

| Type String | Widget | What It Renders |
|---|---|---|
| `"data_table"` | `DynamicTable` | A data table with columns, rows, pagination, sorting, and filters |
| `"form"` | `DynamicForm` | A form with fields generated from a schema |
| `"card"` | `DynamicCard` | A content card with a title, body, and optional actions |
| `"metric"` | `MetricWidget` | A single KPI number (like "Total Orders: 1,247") |
| `"chart"` | `DynamicChart` | A basic chart (bar, line, pie) |
| `"search_bar"` | `SearchBar` | A text input for searching resources |
| `"filter_panel"` | `FilterPanel` | A row/column of filter controls (dropdowns, date pickers) |
| `"status_badge"` | `StatusBadge` | A colored label (e.g., "Pending" in yellow, "Shipped" in green) |
| `"action_button"` | `ActionButton` | A button that triggers a BFF action |
| `"inline_editor"` | `InlineEditor` | An editable cell within a table |
| `"section"` | `Section` | A labeled container that groups other components |
| `"grid_layout"` | `GridLayout` | A responsive grid container |
| `"tab_layout"` | `TabLayout` | A tabbed container |
| `"workflow_step"` | `WorkflowRenderer` | The current step of an active workflow |

### How Registration Works

The registry is initialized at app startup with all built-in components. It is essentially a `Map<String, WidgetBuilder>`:

```
registry = {
  "data_table": (descriptor, ref) → DynamicTable(descriptor),
  "form": (descriptor, ref) → DynamicForm(descriptor),
  "card": (descriptor, ref) → DynamicCard(descriptor),
  ...
}
```

When the page renderer encounters `"type": "data_table"`, it looks up `"data_table"` in this map and calls the builder function.

### Unknown Components

If a BFF sends a component type that is not in the registry (e.g., `"type": "3d_model_viewer"`), the renderer does not crash. Instead, it renders a placeholder:

```
┌─────────────────────────────────────────┐
│  ⚠ Unknown component: "3d_model_viewer" │
│  This component type is not supported.  │
│  Consider registering a plugin.         │
└─────────────────────────────────────────┘
```

This ensures the rest of the page still renders correctly. One unsupported component does not break the entire page.

---

## Supported Page Types

The BFF can describe different types of pages. Each type has a typical component composition:

### Resource List Page

**Purpose**: Show a searchable, sortable, filterable table of items (orders, customers, products, etc.)

**Typical components**:
```
┌─────────────────────────────────────────────────────────┐
│ [Page Title]                           [+ Create New]    │  ← header + action button
├─────────────────────────────────────────────────────────┤
│ 🔍 Search...                                             │  ← search_bar
├─────────────────────────────────────────────────────────┤
│ Status: [All ▾]  Date: [Last 30 days ▾]  [Clear Filters]│  ← filter_panel
├─────────────────────────────────────────────────────────┤
│ ☐ │ ID     │ Name          │ Status  │ Date   │ Actions │  ← data_table
│ ☐ │ ORD-1  │ Alice Smith   │ ● Pend  │ 2h ago │ [View]  │
│ ☐ │ ORD-2  │ Bob Jones     │ ● Ship  │ 1d ago │ [View]  │
├─────────────────────────────────────────────────────────┤
│ [Export CSV] [Cancel Selected]  Page 1 of 50 [< >]       │  ← bulk actions + pagination
└─────────────────────────────────────────────────────────┘
```

### Resource Detail Page

**Purpose**: Show all details of a single item, with editing capabilities.

**Typical components**:
```
┌─────────────────────────────────────────────────────────┐
│ Order #ORD-1234                [Edit] [Delete] [Fulfill] │  ← header + actions
├─────────────────────────────────────────────────────────┤
│ ┌─────────────────────┐  ┌──────────────────────────┐   │
│ │ Order Information    │  │ Status: ● Pending        │   │  ← card + status_badge
│ │ Customer: Alice      │  │ Created: Feb 5, 2026     │   │
│ │ Total: $142.50       │  │ Updated: Feb 6, 2026     │   │
│ └─────────────────────┘  └──────────────────────────┘   │
├─────────────────────────────────────────────────────────┤
│ Line Items                                               │  ← section + data_table
│ │ Product       │ Qty │ Price  │ Total  │                │
│ │ Widget Pro    │ 2   │ $49.99 │ $99.98 │                │
│ │ Gadget Lite   │ 1   │ $42.52 │ $42.52 │                │
└─────────────────────────────────────────────────────────┘
```

### Dashboard Page

**Purpose**: Show an overview with key metrics, charts, and recent activity.

**Typical components**:
```
┌─────────────────────────────────────────────────────────┐
│ Dashboard                                    [Refresh]   │
├─────────────────────────────────────────────────────────┤
│ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐    │
│ │ Orders   │ │ Revenue  │ │ Customers│ │ Returns  │    │  ← metric widgets
│ │  1,247   │ │ $89,432  │ │   892    │ │    23    │    │
│ │ +12% ▲   │ │ +8% ▲    │ │ +5% ▲    │ │ -2% ▼   │    │
│ └──────────┘ └──────────┘ └──────────┘ └──────────┘    │
├─────────────────────────────────────────────────────────┤
│ ┌─────────────────────────┐ ┌──────────────────────┐    │
│ │ Revenue Trend           │ │ Orders by Status     │    │  ← chart widgets
│ │ [line chart]            │ │ [pie chart]          │    │
│ └─────────────────────────┘ └──────────────────────┘    │
├─────────────────────────────────────────────────────────┤
│ Recent Activity                                          │  ← data_table (compact)
│ │ 2 min ago │ Order ORD-1234 created by Alice           │
│ │ 5 min ago │ Customer Bob Jones updated                │
└─────────────────────────────────────────────────────────┘
```

### Embedded Workflow Page

**Purpose**: Guide the user through a multi-step process.

```
┌─────────────────────────────────────────────────────────┐
│ Refund Process                                           │
├─────────────────────────────────────────────────────────┤
│ Step 1         Step 2          Step 3        Step 4      │
│ [●]───────────[●]─────────────[○]───────────[○]         │  ← workflow_stepper
│ Select Order   Enter Reason   Review        Confirm      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  (Current step content renders here — a form,            │  ← workflow_step
│   a review screen, a selection list, etc.)               │
│                                                          │
├─────────────────────────────────────────────────────────┤
│                              [Back]  [Continue]          │
└─────────────────────────────────────────────────────────┘
```

### Custom Container Page

**Purpose**: A page whose layout is entirely defined by the BFF, with no assumed structure. The BFF provides a tree of components, and the UI Engine renders them as described.

---

## Example: BFF Response to Widget Mapping

Here is a complete example showing how a BFF JSON response becomes a visible page.

### The BFF Sends This JSON

(In response to `GET /ui/pages/orders-list`)

```json
{
  "pageId": "orders-list",
  "title": "Orders",
  "layout": {
    "type": "stack",
    "direction": "vertical",
    "spacing": 16
  },
  "components": [
    {
      "type": "search_bar",
      "id": "order-search",
      "placeholder": "Search by order ID, customer name...",
      "searchEndpoint": "/ui/resources/orders",
      "debounceMs": 300
    },
    {
      "type": "filter_panel",
      "id": "order-filters",
      "filters": [
        {
          "field": "status",
          "label": "Status",
          "type": "enum",
          "options": [
            { "value": "pending", "label": "Pending" },
            { "value": "processing", "label": "Processing" },
            { "value": "shipped", "label": "Shipped" },
            { "value": "delivered", "label": "Delivered" }
          ],
          "multi": true
        },
        {
          "field": "created_at",
          "label": "Date",
          "type": "date_range"
        },
        {
          "field": "total",
          "label": "Total",
          "type": "number_range",
          "min": 0,
          "max": 100000
        }
      ]
    },
    {
      "type": "data_table",
      "id": "orders-table",
      "resource": "orders",
      "schemaRef": "schemas/order-summary",
      "columns": [
        { "field": "id", "label": "Order #", "sortable": true, "priority": 1 },
        { "field": "customer_name", "label": "Customer", "sortable": true, "priority": 1 },
        { "field": "total", "label": "Total", "format": "currency", "sortable": true, "priority": 2 },
        { "field": "status", "label": "Status", "component": "status_badge", "priority": 1 },
        { "field": "created_at", "label": "Date", "format": "relative_date", "priority": 3 }
      ],
      "pagination": {
        "type": "server",
        "defaultPageSize": 25,
        "pageSizes": [10, 25, 50, 100]
      },
      "bulkActions": [
        {
          "actionId": "export-csv",
          "label": "Export CSV",
          "icon": "download",
          "allowed": true
        }
      ],
      "rowActions": [
        {
          "actionId": "view-order",
          "label": "View",
          "navigation": "/orders/{id}",
          "allowed": true
        }
      ]
    }
  ],
  "actions": [
    {
      "actionId": "create-order",
      "label": "New Order",
      "icon": "add",
      "position": "header",
      "inputSchema": "schemas/order-create",
      "allowed": true
    }
  ]
}
```

### The UI Engine Processes It

```
1. PageRenderer receives this descriptor

2. Layout: stack/vertical with 16px spacing
   → Use a Column widget with 16px SizedBox between children

3. Component: search_bar (allowed: not specified, defaults to true)
   → Registry lookup: "search_bar" → SearchBar builder
   → Build SearchBar with:
     - placeholder: "Search by order ID, customer name..."
     - debounce: 300ms
     - endpoint: /ui/resources/orders

4. Component: filter_panel (allowed: not specified, defaults to true)
   → Registry lookup: "filter_panel" → FilterPanel builder
   → Build FilterPanel with 3 filters:
     - Multi-select enum dropdown for "Status"
     - Date range picker for "Date"
     - Number range slider for "Total"

5. Component: data_table (allowed: not specified, defaults to true)
   → Registry lookup: "data_table" → DynamicTable builder
   → Resolve schema: schemaProvider("order-summary")
   → Build DynamicTable with:
     - 5 columns (with responsive priority levels)
     - Server-side pagination (25 items per page)
     - One bulk action: "Export CSV"
     - One row action: "View" (navigates to order detail)

6. Page action: "New Order" (allowed: true)
   → Render as a button in the page header
   → On tap: open a form dialog using schema "order-create"

7. Assemble all widgets in a Column → return the widget tree
```

### The User Sees This

```
┌─────────────────────────────────────────────────────────┐
│ Orders                                    [+ New Order]  │
├─────────────────────────────────────────────────────────┤
│ 🔍 Search by order ID, customer name...                  │
├─────────────────────────────────────────────────────────┤
│ Status: [All ▾]   Date: [Any ▾]   Total: [Any ▾]        │
├─────────────────────────────────────────────────────────┤
│ ☐ │ Order #  │ Customer      │ Total    │ Status  │ Date│
│───┼──────────┼───────────────┼──────────┼─────────┼─────│
│ ☐ │ ORD-1234 │ Alice Smith   │ $142.50  │ ● Pend  │ 2h  │
│ ☐ │ ORD-1235 │ Bob Jones     │ $89.00   │ ● Ship  │ 1d  │
│ ☐ │ ORD-1236 │ Carol White   │ $213.75  │ ● Dlvd  │ 3d  │
│   │          │               │          │         │     │
├─────────────────────────────────────────────────────────┤
│ [Export CSV]                      Showing 1-25 of 1,247  │
│                                   [< 1 2 3 ... 50 >]    │
└─────────────────────────────────────────────────────────┘
```

---

## Error Handling in the Engine

The UI Engine wraps every component in an `ErrorBoundary`. This is a safety net:

- If a single component fails to render (bad data, missing schema, rendering bug), only that component shows an error
- All other components on the page continue to work normally
- The error boundary displays a helpful message:

```
┌─────────────────────────────────────────┐
│ ⚠ Failed to render "order-filters"      │
│ Error: Schema "filter-schema" not found │
│ [Retry]                                 │
└─────────────────────────────────────────┘
```

This is critical for an enterprise application. A single bad server response should not take down an entire page.

---

## Performance Considerations

The UI Engine is designed for pages with potentially hundreds of components:

1. **Lazy rendering**: Components below the visible scroll area are not built until the user scrolls to them
2. **Memoization**: Resolved schemas are cached in memory — the same schema is not re-parsed on every rebuild
3. **Const constructors**: Stateless components use `const` constructors so Flutter can skip rebuilding them entirely
4. **RepaintBoundary**: Heavy components (charts, large tables) are wrapped in `RepaintBoundary` so their repainting does not affect surrounding components
