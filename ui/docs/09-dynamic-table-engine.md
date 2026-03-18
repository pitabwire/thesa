# 9. Dynamic Table Engine

## What Is the Table Engine?

The Table Engine is a subsystem of the UI Engine that builds data tables automatically from BFF descriptors. Instead of hardcoding "this table has columns A, B, C with these widths and these sort options," the BFF describes the columns, and the Table Engine renders them.

Tables are the most common component in enterprise applications. Users spend most of their time looking at lists of items — orders, customers, products, transactions, logs. The Table Engine must handle this workload efficiently.

---

## Key Capabilities

| Capability | What It Does |
|---|---|
| **Dynamic columns** | Columns are defined by the BFF, not hardcoded |
| **Server-side pagination** | Only loads one page of data at a time (e.g., 25 rows) |
| **Server-side sorting** | Sends sort parameters to the server, which returns sorted data |
| **Server-side filtering** | Sends filter parameters to the server, which returns filtered data |
| **Column prioritization** | On smaller screens, less important columns are hidden |
| **Column visibility toggle** | Users can show/hide columns |
| **Bulk selection** | Checkboxes allow selecting multiple rows for batch operations |
| **Row actions** | Each row has action buttons (View, Edit, Delete, etc.) |
| **Inline editing** | Some cells can be edited directly in the table |
| **Export** | Selected or all data can be exported (CSV, etc.) |
| **Virtualization** | Only visible rows are rendered in memory |

---

## How It Works

### The BFF Describes a Table

```json
{
  "type": "data_table",
  "id": "orders-table",
  "resource": "orders",
  "schemaRef": "schemas/order-summary",
  "columns": [
    {
      "field": "id",
      "label": "Order #",
      "sortable": true,
      "width": "auto",
      "priority": 1
    },
    {
      "field": "customer_name",
      "label": "Customer",
      "sortable": true,
      "width": "flex",
      "priority": 1
    },
    {
      "field": "total",
      "label": "Total",
      "format": "currency",
      "sortable": true,
      "width": 120,
      "priority": 2,
      "alignment": "right"
    },
    {
      "field": "status",
      "label": "Status",
      "component": "status_badge",
      "sortable": true,
      "width": 100,
      "priority": 1
    },
    {
      "field": "created_at",
      "label": "Date",
      "format": "relative_date",
      "sortable": true,
      "width": 100,
      "priority": 3
    }
  ],
  "pagination": {
    "type": "server",
    "defaultPageSize": 25,
    "pageSizes": [10, 25, 50, 100]
  },
  "defaultSort": {
    "field": "created_at",
    "direction": "desc"
  },
  "bulkActions": [ ... ],
  "rowActions": [ ... ]
}
```

### The Table Engine Processes It

1. **Column builder**: Reads each column descriptor and creates a column configuration:
   - What header label to show
   - How wide the column should be
   - How to format the cell value (currency, date, status badge, etc.)
   - Whether the column is sortable
   - What responsive priority level it has

2. **Pagination controller**: Sets up server-side pagination:
   - Tracks the current page number
   - Tracks the current page size
   - Sends `?page=1&pageSize=25` to the server
   - Shows "Showing 1-25 of 1,247" in the footer

3. **Sort controller**: Sets up server-side sorting:
   - Tracks which column is sorted and in which direction
   - When the user clicks a column header, sends `?sort=created_at&order=desc` to the server
   - Shows a sort indicator (▲ or ▼) on the active column

4. **Row builder**: For each data row returned by the server:
   - Reads each field value
   - Formats it according to the column's `format` setting
   - If the column has `"component": "status_badge"`, renders a `StatusBadge` widget instead of plain text

---

## Server-Side Pagination — How It Works

### Why Server-Side?

Enterprise tables can have millions of rows. Loading all of them into the browser would:
- Use enormous amounts of memory
- Take a very long time to download
- Make the app unresponsive

Instead, the app loads one "page" of data at a time. The server does the heavy lifting.

### The Flow

```
User opens the orders page
  │
  ▼
Table Engine sends: GET /ui/resources/orders?page=1&pageSize=25&sort=created_at&order=desc
  │
  ▼
Server responds:
{
  "data": [ ...25 order objects... ],
  "pagination": {
    "page": 1,
    "pageSize": 25,
    "totalItems": 1247,
    "totalPages": 50
  }
}
  │
  ▼
Table Engine renders 25 rows and pagination controls

User clicks "Next Page" (page 2)
  │
  ▼
Table Engine sends: GET /ui/resources/orders?page=2&pageSize=25&sort=created_at&order=desc
  │
  ▼
Server responds with the next 25 items
  │
  ▼
Table Engine replaces the rows
```

### Pagination Controls

The footer shows:

```
Showing 26-50 of 1,247 items    [10 ▾] [25 ▾] [50 ▾] [100 ▾]    [< 1 2 3 ... 50 >]
```

- "Showing 26-50 of 1,247" — tells the user where they are
- Page size buttons — let the user change how many rows to show per page
- Page navigation — lets the user jump to specific pages or go forward/back

---

## Server-Side Sorting

### How It Works

When the user clicks a column header:

1. **First click**: Sort ascending (A→Z, 1→9, oldest→newest)
2. **Second click**: Sort descending (Z→A, 9→1, newest→oldest)
3. **Third click**: Remove sort (return to default)

The table does NOT sort the data locally. Instead, it sends the sort parameters to the server:

```
GET /ui/resources/orders?sort=customer_name&order=asc&page=1&pageSize=25
```

The server returns data in the requested order.

### Visual Indicator

The sorted column shows an arrow:

```
│ Order # │ Customer ▲ │ Total │ Status │ Date │
```

The ▲ means "sorted ascending by Customer."

---

## Server-Side Filtering

### How It Works

When the user applies filters (from the filter panel above the table), the table sends filter parameters to the server:

```
GET /ui/resources/orders?status=pending,shipped&created_after=2026-01-01&page=1&pageSize=25
```

### Debouncing

Text-based filters (like search) are debounced — the app waits 300 milliseconds after the user stops typing before sending the request. This prevents sending a request for every keystroke.

```
User types: "A" → waits → "Al" → waits → "Ali" → 300ms passes → sends request for "Ali"
```

### Filter Reset

A "Clear Filters" button resets all filters and reloads the default data.

---

## Responsive Column Prioritization

### The Problem

A table with 8 columns looks great on a wide desktop monitor. On a phone, those 8 columns would be crammed together and unreadable.

### The Solution

Each column has a `priority` level (defined by the BFF):

| Priority | Meaning | Visible On |
|---|---|---|
| **1** (highest) | Essential — always visible | Phone, tablet, desktop |
| **2** | Important — visible on medium+ screens | Tablet, desktop |
| **3** | Nice to have — visible on large screens only | Desktop only |

### How It Works

The Table Engine checks the current screen width and hides columns based on their priority:

| Breakpoint | Screen Width | Visible Priorities |
|---|---|---|
| Phone | < 600px | Priority 1 only |
| Tablet | 600-1200px | Priority 1 and 2 |
| Desktop | > 1200px | Priority 1, 2, and 3 |

### What About Hidden Columns?

Hidden columns are still accessible. The user can:

1. **Column picker**: A button (typically ⚙ or "Columns") opens a panel where the user can toggle individual columns on or off
2. **Row expansion**: On phone, tapping a row can expand it to show all fields in a card-like layout

```
Phone view (priority 1 only):

│ Order # │ Customer    │ Status │
│ ORD-1234│ Alice Smith │ ● Pend │  ← tap to expand
   └─────────────────────────────┐
   │ Total: $142.50              │  ← hidden columns shown
   │ Date: Feb 6, 2026           │     in expanded view
   └─────────────────────────────┘
```

---

## Bulk Selection and Actions

### How Bulk Selection Works

Each row has a checkbox in the first column. The table header has a "select all" checkbox.

```
☑ │ Order # │ Customer      │ Status │     ← "select all" checkbox
───┼─────────┼───────────────┼────────│
☑ │ ORD-1234│ Alice Smith   │ ● Pend │     ← selected
☐ │ ORD-1235│ Bob Jones     │ ● Ship │     ← not selected
☑ │ ORD-1236│ Carol White   │ ● Pend │     ← selected
```

When rows are selected, a **bulk action toolbar** appears:

```
┌────────────────────────────────────────────────────────┐
│ 2 items selected    [Export CSV]  [Cancel Orders]       │
└────────────────────────────────────────────────────────┘
```

### Bulk Actions

Bulk actions are defined by the BFF:

```json
{
  "bulkActions": [
    {
      "actionId": "export-csv",
      "label": "Export CSV",
      "icon": "download",
      "allowed": true
    },
    {
      "actionId": "bulk-cancel",
      "label": "Cancel Orders",
      "icon": "cancel",
      "allowed": true,
      "confirmation": {
        "message": "Cancel {count} orders? This cannot be undone.",
        "style": "destructive"
      }
    }
  ]
}
```

- Actions with `allowed: false` do not appear
- Actions with `confirmation` show a dialog before executing
- `{count}` is replaced with the number of selected items

### "Select All" Behavior

The "select all" checkbox selects all rows **on the current page**, not all 1,247 items. For "select all across all pages," a separate option appears:

```
┌────────────────────────────────────────────────────────┐
│ All 25 on this page selected.                          │
│ [Select all 1,247 items across all pages]              │
└────────────────────────────────────────────────────────┘
```

If the user clicks "Select all 1,247 items," the bulk action sends a flag to the server indicating "apply to all matching items" rather than sending 1,247 individual IDs.

---

## Row Actions

Each row can have action buttons, defined by the BFF:

```json
{
  "rowActions": [
    {
      "actionId": "view-order",
      "label": "View",
      "icon": "visibility",
      "navigation": "/orders/{id}",
      "allowed": true
    },
    {
      "actionId": "edit-order",
      "label": "Edit",
      "icon": "edit",
      "navigation": "/orders/{id}/edit",
      "allowed": true
    },
    {
      "actionId": "delete-order",
      "label": "Delete",
      "icon": "delete",
      "allowed": false,
      "confirmation": {
        "message": "Delete order {id}? This cannot be undone.",
        "style": "destructive"
      }
    }
  ]
}
```

- `allowed: false` on "delete-order" means the delete button does not appear for this user
- `navigation: "/orders/{id}"` means clicking "View" navigates to that page, with `{id}` replaced by the row's ID

### Action Display

On desktop: actions appear as icon buttons in the last column:

```
│ ... │ [👁] [✏️] │
```

On tablet: actions appear in a "more" menu (three dots):

```
│ ... │ [⋮] │
         └── View
             Edit
```

On phone: actions appear in a bottom sheet when the row is tapped.

---

## Cell Formatting

The Table Engine supports several built-in cell formatters:

| Format | Input | Output |
|---|---|---|
| `currency` | `14250` | `$142.50` |
| `percentage` | `0.85` | `85%` |
| `relative_date` | `"2026-02-06T10:00:00Z"` | `2 hours ago` |
| `date` | `"2026-02-06T10:00:00Z"` | `Feb 6, 2026` |
| `datetime` | `"2026-02-06T10:00:00Z"` | `Feb 6, 2026 at 10:00 AM` |
| `boolean` | `true` | `✓` (checkmark icon) |
| `number` | `1247` | `1,247` |
| `bytes` | `1048576` | `1.0 MB` |

When a column specifies `"component": "status_badge"`, the cell does not display plain text. Instead, it renders a `StatusBadge` widget — a colored pill with the status text:

```
● Pending    (yellow)
● Shipped    (blue)
● Delivered  (green)
● Cancelled  (red)
```

The colors are defined in the BFF response or the design system based on the status value.

---

## Virtualization — Handling Large Data

### The Problem

If a page shows 100 rows, building all 100 row widgets at once wastes memory and slows down the initial render. On a phone screen, only about 10 rows are visible at a time — the other 90 are below the scroll area.

### The Solution

The Table Engine uses Flutter's **virtualized list** (via `CustomScrollView` and `SliverList.builder`). This means:

- Only the visible rows (plus a small buffer above and below) are built as widgets
- As the user scrolls, new rows are built and old rows (scrolled out of view) are destroyed
- Memory usage stays constant regardless of the total row count

### What This Means in Practice

| Total Rows | Without Virtualization | With Virtualization |
|---|---|---|
| 25 | 25 widgets built | 25 widgets built |
| 100 | 100 widgets built | ~15 widgets built |
| 1,000 | 1,000 widgets built | ~15 widgets built |
| 10,000 | 10,000 widgets built (freezes) | ~15 widgets built |

The user can scroll through thousands of rows smoothly because only 10-15 are ever in memory.

---

## Export

When the user clicks "Export CSV," the process is:

1. The table sends a request to the BFF: `POST /ui/actions/export-csv` with the current filters and sort settings
2. The BFF generates the export file server-side (important for large datasets)
3. The BFF responds with a download URL
4. The browser/app downloads the file

The export happens server-side because:
- The table might have 100,000 matching rows but only shows 25 at a time
- Generating a 100,000-row CSV on the client would freeze the app
- The server has direct access to the full dataset
