# 6. State Architecture (Riverpod)

## What Is "State" in an Application?

State is everything your application "knows" at any given moment. Think of it like the app's memory:

- Who is logged in? → **Auth state**
- What menu items should the sidebar show? → **Navigation state**
- What does the current page look like? → **Page state**
- What data structure does a form use? → **Schema state**
- Is the user online or offline? → **Connectivity state**
- What step is the user on in a multi-step workflow? → **Workflow state**

When state changes, the screen needs to update. State management is the system that makes this happen reliably.

---

## How Riverpod Works — A Simple Explanation

### Providers

A **provider** is a named container that holds a piece of state. Think of it like a labeled box:

```
Box labeled "auth":        Contains → { user: "Alice", token: "abc123" }
Box labeled "navigation":  Contains → { items: [Dashboard, Orders, Settings] }
Box labeled "page:orders": Contains → { title: "Orders", components: [...] }
```

### Watching

A **widget** can "watch" a provider. This means: "Whenever the data in this box changes, redraw me."

```
The sidebar watches the "navigation" box.
→ When the navigation data changes, the sidebar redraws itself with new menu items.
→ When the navigation data does NOT change, the sidebar does nothing (efficient).
```

### Families

A **family** is a provider that creates separate instances based on a parameter. Instead of one box, it is a row of boxes, each labeled with a parameter:

```
Box "page:dashboard"     → Contains dashboard page data
Box "page:orders-list"   → Contains orders list page data
Box "page:settings"      → Contains settings page data
```

One provider definition creates all of these. You just say `pageProvider('dashboard')` and Riverpod either returns the existing box or creates a new one.

### Auto-Dispose

When nothing is watching a provider anymore (e.g., the user navigated away from a page), Riverpod automatically destroys that provider's instance and frees the memory. This prevents memory leaks.

### Keep Alive

Some providers should NEVER be destroyed, even if nothing is watching them temporarily. For example:
- The navigation tree should persist even when the sidebar is briefly hidden
- Schemas should persist because they are shared across many pages

These providers use `keepAlive: true`.

---

## The Provider Hierarchy

Providers in Thesa UI form a dependency tree. When a "parent" provider changes, all "children" that depend on it automatically update:

```
                    connectivityProvider
                    (Am I online or offline?)
                            │
          ┌─────────────────┼─────────────────────┐
          │                 │                       │
     authProvider    capabilitiesProvider    navigationProvider
     (Who am I?)    (What's available?)     (What's in the menu?)
          │                 │                       │
     sessionProvider   pageProvider(id)      routerProvider
     (My permissions)  (What's on this page?) (What routes exist?)
                            │
              ┌─────────────┼─────────────────┐
              │             │                  │
        schemaProvider  actionProvider   workflowProvider
        (Field defs)    (Button logic)   (Step tracking)
```

### Reading This Diagram

- **Arrows pointing down** mean "depends on"
- If `authProvider` changes (user logs out), then `sessionProvider` automatically updates (session cleared)
- If `navigationProvider` changes (BFF sends new menu), then `routerProvider` automatically updates (routes regenerated)
- If `connectivityProvider` changes (device goes offline), all providers that make network requests are notified

---

## Each Provider Explained

### connectivityProvider

**What it holds**: A boolean — `true` if the device has internet, `false` if not.

**How it works**: Uses the `connectivity_plus` library to listen for network changes. Emits a stream that other providers watch.

**Who uses it**: The cache coordinator (to decide whether to attempt network requests), the stale cache banner (to show offline warning).

**Lifecycle**: Always alive. Never disposed.

---

### authProvider

**What it holds**: The authentication state — logged out, logging in, logged in (with token), or error.

**How it works**:
1. On app start, reads token from `flutter_secure_storage`
2. If token exists and has not expired → state = logged in
3. If token expired → attempts refresh with the server
4. If refresh fails → state = logged out → redirect to login

**Who uses it**: Everything. If there is no auth, nothing else can work.

**Lifecycle**: Always alive. Never disposed.

**Key method**: `logout()` — clears the token, clears all caches, redirects to login.

---

### sessionProvider

**What it holds**: The current user's profile and permissions.

**Depends on**: `authProvider` (no auth = no session)

**How it works**: When auth state becomes "logged in," this provider fetches user info and permissions from the BFF. Permissions are cached locally with a 5-minute TTL.

**Who uses it**: `PermissionGate` widgets (to show/hide UI based on permissions), the sidebar (to filter menu items), the page renderer (to filter components and actions).

**Lifecycle**: Always alive while authenticated. Destroyed on logout.

---

### capabilitiesProvider

**What it holds**: Global capability flags — what features the BFF supports.

**How it works**: Fetches `GET /ui/capabilities` from the BFF. Caches the result. Contains things like:
- `globalVersion: 42` (used for cache invalidation)
- `features: { workflows: true, analytics: true, bulkActions: true }`
- `locales: ["en", "fr", "de"]`

**Who uses it**: The UI engine (to enable/disable entire feature areas), the cache coordinator (to detect global version changes).

**Lifecycle**: Always alive (`keepAlive: true`). Background-refreshed periodically.

---

### navigationProvider

**What it holds**: The sidebar menu tree — every menu item, its icon, path, children, and permission flags.

**How it works**:
1. Reads from navigation cache (Drift) immediately
2. If stale, fires background refresh to `GET /ui/navigation`
3. Exposes a reactive stream via Drift's `watch()`

**Who uses it**: The sidebar widget, the route builder (to generate routes), the breadcrumb bar.

**Lifecycle**: Always alive (`keepAlive: true`). This data must persist to enable instant sidebar rendering.

---

### pageProvider(pageId) — Family Provider

**What it holds**: The complete descriptor for a single page — its title, layout, components, actions, and permissions.

**How it works**:
1. `pageProvider('orders-list')` is activated when the user navigates to that page
2. Reads from page cache (Drift) → returns cached descriptor
3. If stale → background refresh to `GET /ui/pages/orders-list`
4. When the user navigates away, the provider is auto-disposed (memory freed)

**Who uses it**: `PageRenderer` (the widget that builds the page from the descriptor).

**Lifecycle**: Auto-disposed when the user leaves the page. Re-created if they return.

**Why `family`?**: There could be 5 pages or 500. We cannot pre-create providers for all of them. `family` creates them on demand.

---

### schemaProvider(schemaId) — Family Provider

**What it holds**: A single schema definition — field names, types, validation rules, visibility conditions.

**How it works**: Same cache-first pattern as `pageProvider`.

**Who uses it**: The form engine (to build form fields), the table engine (to format columns), the schema resolver (to follow `$ref` pointers).

**Lifecycle**: Always alive (`keepAlive: true`). Schemas are shared across pages, so disposing them would cause unnecessary re-fetches. Reference counting in the cache layer handles cleanup.

---

### actionProvider(actionId) — Family Provider

**What it holds**: The execution state of a single action — idle, executing, success, or error.

**How it works**:
1. User clicks an action button
2. Provider moves to "executing" state → UI shows loading indicator on button
3. Provider sends `POST /ui/actions/{actionId}` with the action's input data
4. On success → state = success → UI shows confirmation
5. On error → state = error → UI shows error message

**Who uses it**: Action buttons, bulk action toolbar.

**Lifecycle**: Auto-disposed after the action completes. Short-lived.

---

### workflowProvider(workflowId) — Family Provider

**What it holds**: The full state of an active workflow — current step, accumulated data from all completed steps, transition history.

**How it works**:
1. User starts a workflow → provider fetches `GET /ui/workflows/{workflowId}`
2. BFF responds with workflow definition (steps, transitions, conditions)
3. Provider renders the current step (a form, a review screen, etc.)
4. User completes a step → provider sends `POST /ui/workflows/{workflowId}/step`
5. BFF responds with the next step → provider advances

**Who uses it**: `WorkflowRenderer` (the widget that shows workflow steps).

**Lifecycle**: Always alive (`keepAlive: true`). Workflow progress is persisted to the Drift database so it survives app restarts. The provider remains alive until the workflow completes or the user explicitly abandons it.

---

### connectivityProvider

**What it holds**: A stream of connectivity states — online, offline, or limited.

**How it works**: Uses `connectivity_plus` to listen for system-level network changes. When the device's connection status changes, this provider emits a new value.

**Who uses it**: Cache coordinator (to skip network calls when offline), stale cache banner (to show offline indicator), background refresh coordinator (to pause/resume refreshes).

**Lifecycle**: Always alive.

---

## The Cache-First Provider Pattern

Every data-fetching provider in Thesa UI follows the same pattern. Understanding this pattern once means you understand all of them:

### Step-by-Step

```
Provider.build() is called (this runs when the provider is first accessed):

  STEP 1: Read from the local database
  ─────────────────────────────────────
  Query Drift for cached data matching the provider's key.
  This is fast — SQLite reads take <10 milliseconds.

  STEP 2: Check freshness
  ─────────────────────────────────────
  If cached data exists and its TTL has not expired:
    → Return the cached data immediately.
    → Done. No network request.
    → The widget renders instantly.

  STEP 3: Handle stale or empty cache
  ─────────────────────────────────────
  If cached data exists but is stale (TTL expired):
    → Set the provider's state to AsyncData(cachedData)
    → This means: "I have data. It might be old, but show it."
    → The widget renders immediately with cached data.

  If cached data does not exist:
    → Set the provider's state to AsyncLoading()
    → This means: "I have no data yet."
    → The widget shows a loading skeleton.

  STEP 4: Fetch fresh data in the background
  ─────────────────────────────────────
  Fire an async network request to the BFF.
  This does NOT block the widget from rendering.

  STEP 5: Process the response
  ─────────────────────────────────────
  When the network response arrives:
    → Write the fresh data to the Drift database.
    → Drift's watch() stream automatically emits the new data.
    → The Riverpod provider automatically updates its state.
    → The widget automatically rebuilds with the fresh data.

  If the network request fails (offline, server error):
    → Keep the stale cached data.
    → Show a "data may be outdated" banner.
    → Retry later.
```

### The Key Insight

The magic connection is between Drift's `watch()` and Riverpod:

1. A Riverpod provider subscribes to a Drift `watch()` stream
2. Drift `watch()` emits a new value whenever the underlying table row changes
3. When the background refresh writes new data to Drift, the stream fires
4. Riverpod receives the stream event and updates the provider
5. All widgets watching the provider automatically rebuild

This means: **writing to the database is the only way to update the UI**. There is no manual "setState" or "emit" call. The data flow is:

```
Network → Database → Stream → Provider → Widget
```

This single, unidirectional flow makes the system predictable and easy to debug. If the UI is showing wrong data, you check the database. If the database has wrong data, you check the network response.

---

## Permission State — How It Flows

Permissions deserve special attention because they are a security concern. Here is exactly how permissions work:

### The Rule

**The UI never decides permissions.** The BFF decides. The UI only reads the BFF's decisions.

### How It Works

1. **The BFF embeds permission flags in every response**:
   - Navigation items have `"allowed": true/false`
   - Page components have `"allowed": true/false`
   - Actions have `"allowed": true/false`
   - Workflow steps have `"allowed": true/false`

2. **The sessionProvider holds the user's permission context** (fetched from BFF)

3. **The `PermissionGate` widget** wraps any UI element that should be permission-controlled:
   ```
   PermissionGate(
     allowed: component.allowed,
     child: ActionButton(...)
   )
   ```
   - If `allowed == true` → render the child
   - If `allowed == false` → render nothing (the button does not exist)

4. **Key distinction**: The button is not "greyed out" or "disabled." It is **absent from the widget tree entirely**. This means:
   - A user cannot inspect the page and find hidden buttons
   - There is no risk of accidentally enabling a disabled button
   - The UI is cleaner — no confusing greyed-out options

### Why Not Evaluate Permissions Locally?

Some apps download a list of permissions and then check them locally:
```
// DON'T DO THIS:
if (user.permissions.contains('orders.delete')) {
    showDeleteButton();
}
```

This is dangerous because:
- The local permission check might be wrong (stale data, logic bug)
- The backend might have additional context the frontend lacks
- It duplicates security logic in two places (backend + frontend)
- If the frontend check disagrees with the backend, the user sees an error

In Thesa UI, the BFF has already evaluated permissions. The frontend just reads the boolean answer.

---

## Putting It All Together — A Real Example

Let us trace what happens when user "Alice" opens the app and navigates to the Orders page:

### Phase 1: App Start

```
1. main() runs → Drift database initialized, ProviderScope created
2. authProvider checks secure storage → finds token → state = authenticated
3. sessionProvider activates → reads permissions from cache → Alice has 'orders.view', 'orders.create'
4. capabilitiesProvider activates → reads capabilities from cache → global version = 42
5. navigationProvider activates → reads nav tree from cache → sidebar renders instantly
6. routerProvider activates → generates routes from cached nav
```

### Phase 2: Alice Clicks "Orders" in the Sidebar

```
7. Go_router navigates to /orders/list
8. pageProvider('orders-list') activates for the first time
9. Cache coordinator checks: page is in cache, fetched 8 minutes ago, TTL is 10 minutes → FRESH
10. PageRenderer receives the cached page descriptor
11. PageRenderer asks ComponentRegistry to build: search bar, filter panel, data table
12. Each component renders → Alice sees the full orders page
```

### Phase 3: Background Refresh (Invisible to Alice)

```
13. Background refresh coordinator triggers navigation refresh
14. ETagInterceptor sends If-None-Match: "nav-etag-old"
15. BFF responds 304 Not Modified → no update needed
16. Background refresh triggers capabilities refresh
17. BFF responds 200 OK with globalVersion: 43 (changed!)
18. Cache coordinator detects version change → marks all caches as stale
19. Page provider for 'orders-list' detects stale data → fetches fresh descriptor
20. BFF responds with updated descriptor (a new column was added to the table)
21. Fresh data written to Drift → stream fires → provider updates → table gains new column
22. Alice sees the new column appear smoothly — no page reload, no spinner
```
