# 11. Dynamic Navigation and Routing

## What Is Dynamic Navigation?

In most apps, the navigation menu (sidebar, tabs, breadcrumbs) is hardcoded by developers. Adding a new page requires changing the frontend code, rebuilding the app, and redeploying.

In Thesa UI, the navigation is **dynamic** — it comes from the BFF server. The sidebar menu, the available routes, and the breadcrumb trail are all built at runtime from data the server provides. Adding a new page to the menu requires only a server-side change. The frontend automatically picks it up.

---

## How Navigation Data Flows

```
BFF Server
    │
    │  GET /ui/navigation
    │
    ▼
Navigation Response (JSON)
    │
    ▼
navigationProvider (Riverpod)
    │
    ├──▶ Sidebar Widget (renders menu items)
    │
    ├──▶ DynamicRouteBuilder (generates GoRouter routes)
    │     │
    │     ▼
    │    routerProvider (GoRouter instance)
    │     │
    │     ▼
    │    App Shell (StatefulShellRoute)
    │
    └──▶ Breadcrumb Bar (derives path from current route)
```

---

## The Navigation Response

The BFF sends a tree of navigation items:

```json
{
  "version": 42,
  "items": [
    {
      "id": "dashboard",
      "label": "Dashboard",
      "icon": "dashboard",
      "path": "/dashboard",
      "pageId": "main-dashboard",
      "allowed": true,
      "badge": null,
      "position": "top"
    },
    {
      "id": "orders",
      "label": "Orders",
      "icon": "shopping_cart",
      "path": "/orders",
      "allowed": true,
      "position": "top",
      "children": [
        {
          "id": "orders-list",
          "label": "All Orders",
          "path": "/orders/list",
          "pageId": "orders-list",
          "allowed": true,
          "badge": { "count": 12, "color": "warning" }
        },
        {
          "id": "orders-returns",
          "label": "Returns",
          "path": "/orders/returns",
          "pageId": "returns-list",
          "allowed": true
        },
        {
          "id": "orders-analytics",
          "label": "Analytics",
          "path": "/orders/analytics",
          "pageId": "orders-analytics",
          "allowed": false
        }
      ]
    },
    {
      "id": "products",
      "label": "Products",
      "icon": "inventory",
      "path": "/products",
      "allowed": false
    },
    {
      "id": "settings",
      "label": "Settings",
      "icon": "settings",
      "path": "/settings",
      "pageId": "settings-page",
      "allowed": true,
      "position": "bottom"
    }
  ]
}
```

### Field-by-Field Explanation

| Field | What It Means | Example |
|---|---|---|
| `id` | Unique identifier for this nav item | `"orders-list"` |
| `label` | The text displayed in the sidebar | `"All Orders"` |
| `icon` | The Material icon name displayed next to the label | `"shopping_cart"` |
| `path` | The URL path this item navigates to | `"/orders/list"` |
| `pageId` | Which page descriptor to load for this route | `"orders-list"` |
| `allowed` | Whether the current user can see this item | `true` or `false` |
| `badge` | An optional notification badge (count + color) | `{ "count": 12, "color": "warning" }` |
| `position` | Where to place this item: top (main area) or bottom (footer area) | `"top"` or `"bottom"` |
| `children` | Sub-items that appear when the parent is expanded | Array of nav items |
| `version` | A number that changes when the nav structure changes | `42` |

### Permission Filtering

Items with `allowed: false` are **completely removed** from the sidebar. They are not greyed out or disabled — they do not exist in the rendered widget tree. In the example above:

- "Products" (`allowed: false`) → not visible to this user
- "Orders > Analytics" (`allowed: false`) → not visible

The user has no way to know these items exist.

---

## Dynamic Route Generation

### What Is a Route?

A route is a mapping between a URL path and a screen. When the user navigates to `/orders/list`, the router looks up which widget to show for that path.

### How Routes Are Generated

The `DynamicRouteBuilder` converts the BFF navigation tree into GoRouter routes:

```
For each navigation item where allowed == true:

  IF item has a pageId:
    Create a GoRoute:
      path: item.path
      builder: PageRenderer(pageId: item.pageId)

  IF item has children:
    For each child where allowed == true:
      Create a child GoRoute:
        path: child.path
        builder: PageRenderer(pageId: child.pageId)
```

From the example above, the generated routes would be:

```
/dashboard            → PageRenderer(pageId: "main-dashboard")
/orders/list          → PageRenderer(pageId: "orders-list")
/orders/returns       → PageRenderer(pageId: "returns-list")
/settings             → PageRenderer(pageId: "settings-page")
```

Note: `/orders/analytics` is NOT generated because `allowed: false`. Even if a user manually types the URL, the router has no route for it and shows a "Page Not Found" screen.

### Route Wrapping with StatefulShellRoute

All generated routes are wrapped in a `StatefulShellRoute`. This is the layout container that provides the sidebar and content area:

```
StatefulShellRoute (provides sidebar + content area)
├── /dashboard            → content area shows dashboard
├── /orders/list          → content area shows orders table
├── /orders/returns       → content area shows returns table
└── /settings             → content area shows settings
```

When the user switches between routes, the sidebar stays visible and the content area changes. The sidebar does NOT rebuild — it persists across route changes.

### Dynamic Detail Routes

Some routes need parameters. For example, viewing a specific order: `/orders/ORD-1234`. These are not always in the navigation tree (you do not list every order in the sidebar). Instead, the route builder creates pattern-matched routes:

```
/orders/:id             → PageRenderer(pageId: "order-detail", params: { id: "ORD-1234" })
```

The BFF can define parameterized routes in the navigation response:

```json
{
  "id": "order-detail",
  "path": "/orders/:id",
  "pageId": "order-detail",
  "allowed": true,
  "hidden": true
}
```

`hidden: true` means this route exists (the user can navigate to it) but it does NOT appear as a sidebar item.

---

## The Sidebar

### Structure

```
┌────────────────────────┐
│  [Logo / App Name]     │
│                        │
│  ● Dashboard           │  ← top items
│                        │
│  ▼ Orders         (12) │  ← expandable, with badge
│    ├ All Orders    (12) │
│    └ Returns            │
│                        │
│                        │
│  (spacer)              │
│                        │
│  ⚙ Settings            │  ← bottom items
│  [Collapse ◀]          │
└────────────────────────┘
```

### Behavior

| Action | Result |
|---|---|
| Click a leaf item | Navigate to that item's page. Highlight the item. |
| Click a parent item (e.g., "Orders") | Toggle expand/collapse of children. If the parent has its own `pageId`, also navigate to it. |
| Click the collapse button | Sidebar collapses to icon-only mode (rail). |
| Hover over a collapsed icon | Shows a tooltip with the label. |
| Click a collapsed icon | If leaf: navigate. If parent: show a flyout menu with children. |

### Active State Highlighting

The sidebar highlights the item matching the current route:

```
│  ○ Dashboard           │  ← not active (dimmer)
│  ▼ Orders              │
│    ● All Orders        │  ← active (highlighted, bold)
│    ○ Returns           │  ← not active
```

The highlighting is computed by comparing the current URL path with each item's `path`.

### Badges

Badges show counts on navigation items (e.g., "12 pending orders"):

```
│  Orders           (12) │
```

Badge data comes from the BFF. The `badge` object specifies:
- `count`: The number to display
- `color`: Semantic color — `"info"` (blue), `"warning"` (yellow), `"error"` (red)

Badges are refreshed whenever the navigation data is refreshed (every 15 minutes by default, or on demand).

---

## Navigation State Persistence

### Cross-Session Persistence

The navigation tree is cached in Drift. When the app starts:

1. Read cached navigation from Drift → render sidebar immediately
2. Background-refresh navigation from BFF
3. If the BFF returns a different `version` number → update sidebar smoothly

### In-Session State

The sidebar's expand/collapse state (which groups are open, whether the sidebar is in rail mode) is held in `sidebarState` (a simple Riverpod provider). This state:
- Persists during the session (navigating between pages does not reset it)
- Resets on app restart (it is not persisted to Drift — sidebar expand state is not important enough to cache)

### Navigation Version Changes

When the BFF updates the navigation (adds a new page, removes an item, changes permissions):

1. `navigationProvider` detects the version change during background refresh
2. The provider updates with the new navigation tree
3. The sidebar widget rebuilds with new items
4. The `DynamicRouteBuilder` regenerates routes
5. The `routerProvider` rebuilds `GoRouter` with the new routes

**If the user is on a page that was removed**: The route guard detects that the current path no longer matches any route and redirects to the first available route (usually the dashboard).

**If a new page was added**: It appears in the sidebar on the next render. No action needed from the user.

---

## Breadcrumbs

### What Are Breadcrumbs?

Breadcrumbs show the user where they are in the navigation hierarchy:

```
Home > Orders > All Orders > Order #ORD-1234
```

They allow the user to click any ancestor to navigate back up the hierarchy.

### How Breadcrumbs Are Generated

The breadcrumb bar reads the current URL path and the navigation tree to build the trail:

```
Current path: /orders/ORD-1234

Navigation tree:
  Dashboard → /dashboard
  Orders → /orders
    All Orders → /orders/list
    Returns → /orders/returns

Breadcrumb computation:
  1. Start with "Home" (always present, links to /)
  2. Match /orders → "Orders" (from nav tree)
  3. Match /orders/ORD-1234 → "Order #ORD-1234" (from page title)

Result: Home > Orders > Order #ORD-1234
```

### Breadcrumb Behavior

| Element | Clickable? | Action |
|---|---|---|
| Home | Yes | Navigate to root (/) |
| Orders | Yes | Navigate to /orders |
| Order #ORD-1234 | No | Current page (not clickable) |

---

## Route Guards

### What Is a Route Guard?

A route guard is a function that runs before navigation completes. It can allow the navigation, redirect to a different page, or block it entirely.

### Guards in Thesa UI

| Guard | What It Checks | If It Fails |
|---|---|---|
| **Auth guard** | Is the user authenticated? | Redirect to `/login` |
| **Permission guard** | Does the current route exist in the generated routes? (Routes are only generated for allowed items.) | Redirect to `/` (or a 403 page) |
| **Session guard** | Is the session still valid? (Token not expired?) | Redirect to `/login` with a "session expired" message |

### Guard Execution Order

```
User navigates to /orders/list
        │
        ▼
1. Auth guard: Is user authenticated?
   ├── NO → Redirect to /login?returnUrl=/orders/list
   └── YES → Continue
        │
        ▼
2. Session guard: Is token still valid?
   ├── EXPIRED → Attempt token refresh
   │   ├── Refresh success → Continue
   │   └── Refresh failure → Redirect to /login
   └── VALID → Continue
        │
        ▼
3. Permission guard: Does /orders/list exist in generated routes?
   ├── NO → Redirect to / with "Access denied" message
   └── YES → Continue
        │
        ▼
Navigation completes → PageRenderer loads page descriptor
```

### Return URL

When the auth guard redirects to login, it saves the original URL the user was trying to visit. After successful login, the user is automatically redirected back to that URL:

```
User tries to visit /orders/list → not authenticated → /login?returnUrl=/orders/list
User logs in → redirected to /orders/list (not to the default dashboard)
```

---

## Deep Linking

### What Is Deep Linking?

Deep linking means the user can type a URL directly into the browser (or click a shared link) and arrive at a specific page:

```
https://app.company.com/orders/ORD-1234
```

### How It Works in Thesa UI

1. GoRouter parses the URL
2. Route guards execute (auth check, permission check)
3. If guards pass → the matching route's PageRenderer loads
4. PageRenderer fetches the page descriptor → renders the page
5. The sidebar highlights the matching nav item

### On Web vs. Mobile

| Platform | Deep Link Format | How It Is Triggered |
|---|---|---|
| Web | Browser URL bar / shared link | User types URL or clicks a link |
| Android | Custom URL scheme or App Links | User clicks a link that opens the app |
| iOS | Universal Links | User clicks a link that opens the app |
| Desktop | Command line argument or protocol handler | User clicks a link or opens from file association |

GoRouter handles all of these uniformly.

---

## Tab Views

Some pages use tabs to organize content. Tabs can be defined at two levels:

### Page-Level Tabs

The BFF page descriptor uses a `tab_layout` component:

```json
{
  "type": "tab_layout",
  "tabs": [
    { "label": "Active", "components": [ ... ] },
    { "label": "Completed", "components": [ ... ] },
    { "label": "Cancelled", "components": [ ... ] }
  ]
}
```

These tabs exist within a single page (single URL). Switching tabs does not change the URL.

### Navigation-Level Tabs

Multiple navigation items can share a parent route, appearing as tabs:

```json
{
  "id": "orders",
  "path": "/orders",
  "tabMode": true,
  "children": [
    { "id": "orders-active", "label": "Active", "path": "/orders/active", "pageId": "orders-active" },
    { "id": "orders-completed", "label": "Completed", "path": "/orders/completed", "pageId": "orders-completed" }
  ]
}
```

These tabs correspond to separate routes (separate URLs). Each tab loads a different page descriptor. The `StatefulShellRoute` preserves the state of each tab independently — scrolling on the Active tab is preserved when you switch to Completed and back.
