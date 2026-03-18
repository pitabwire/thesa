# 20. Data Flow, Startup Sequence, and Scalability

## Complete Data Flow

This section ties everything together by showing exactly how data moves through the system from the server to the user's screen and back.

### Read Flow: Server → Screen

This is what happens when the user needs to see data (opening a page, loading a table, viewing a dashboard):

```
                                          ┌──────────┐
                                          │   BFF    │
                                          │  Server  │
                                          └────┬─────┘
                                               │
                                    HTTP Response (JSON)
                                               │
                                               ▼
                                     ┌────────────────┐
                                     │  dio Client    │
                                     │  (interceptors)│
                                     └───────┬────────┘
                                             │
                                     Parsed Dart Objects
                                             │
                                             ▼
                                     ┌────────────────┐
                                     │  Cache Layer   │
                                     │  (Drift/SQLite)│ ←── writes data to local DB
                                     └───────┬────────┘
                                             │
                                    Drift watch() Stream
                                             │
                                             ▼
                                     ┌────────────────┐
                                     │  Riverpod      │
                                     │  Provider      │ ←── receives stream update
                                     └───────┬────────┘
                                             │
                                    Provider State Update
                                             │
                                             ▼
                                     ┌────────────────┐
                                     │  UI Engine     │
                                     │  (renderer)    │ ←── interprets descriptor
                                     └───────┬────────┘
                                             │
                                       Widget Tree
                                             │
                                             ▼
                                     ┌────────────────┐
                                     │  Screen        │
                                     │  (user sees    │
                                     │   the result)  │
                                     └────────────────┘
```

**Key insight**: Data always flows in ONE direction: Server → Network → Cache → Provider → Engine → Screen. There are no shortcuts. This unidirectional flow makes bugs easy to trace — if the screen shows wrong data, check each layer in sequence.

### Write Flow: Screen → Server

This is what happens when the user takes an action (submitting a form, clicking a button, advancing a workflow):

```
┌────────────────┐
│  User Action   │ ←── tap, type, select
│  (screen)      │
└───────┬────────┘
        │
  User Input Data
        │
        ▼
┌────────────────┐
│  UI Engine     │ ←── form validation (UX only, not security)
│  (form/action) │
└───────┬────────┘
        │
  Validated Data
        │
        ▼
┌────────────────┐
│  Riverpod      │ ←── actionProvider / workflowProvider
│  Provider      │     manages submission state
└───────┬────────┘
        │
  POST Request
        │
        ▼
┌────────────────┐
│  dio Client    │ ←── auth token added by interceptor
│  (interceptors)│
└───────┬────────┘
        │
  HTTP Request
        │
        ▼
┌────────────────┐
│  BFF Server    │ ←── validates, processes, returns result
└───────┬────────┘
        │
  Response (success or error)
        │
        ▼
┌────────────────┐
│  Provider      │ ←── updates state (success → refresh data, error → show error)
└───────┬────────┘
        │
        ├── Success → invalidate relevant data providers → triggers read flow → screen updates
        └── Error → error state → screen shows error message
```

### Cache-First Read Flow (The Common Path)

Most data reads do NOT hit the network. They are served from cache:

```
User navigates to /orders/list
        │
        ▼
pageProvider('orders-list') activated
        │
        ▼
Cache Coordinator: "Do I have this page cached?"
        │
        ├── YES, fresh → Return cached data → Screen renders in <100ms
        │                                      │
        │                                      └── Done. No network request.
        │
        ├── YES, stale → Return cached data → Screen renders in <100ms
        │                    │
        │                    └── ALSO: trigger background refresh
        │                              │
        │                              ▼
        │                         Network fetch → Update cache → Stream notifies → Screen updates
        │
        └── NO → Show loading skeleton → Network fetch → Write to cache → Screen renders
```

---

## Startup Sequence — Detailed

Here is the exact order of operations when the app starts, with timing estimates:

### Cold Start (First Launch Ever)

```
T+0ms     main() begins
            │
T+10ms    Initialize Drift database (create SQLite file, run migrations)
            │
T+20ms    Initialize flutter_secure_storage
            │
T+30ms    Create ProviderScope (Riverpod container)
            │
T+50ms    MaterialApp.router builds
            │
T+60ms    authProvider.build() runs
            ├── Checks secure storage → no token found
            ├── State = Unauthenticated
            │
T+70ms    Route guard: not authenticated → redirect to /login
            │
T+80ms    Login screen renders
            │
            ... user enters credentials ...
            │
T+user    POST /auth/login { email, password }
            │
T+user+300ms  BFF responds: { accessToken, refreshToken }
            │
T+user+310ms  Tokens stored in secure storage
            │
T+user+320ms  authProvider state = Authenticated
            │
T+user+330ms  (PARALLEL) Launch:
            │   ├── capabilitiesProvider → GET /ui/capabilities
            │   ├── navigationProvider → GET /ui/navigation
            │   └── sessionProvider → GET /ui/session
            │
T+user+350ms  Sidebar shows loading skeleton
            │
T+user+600ms  Navigation response arrives → cached → sidebar renders with real items
            │
T+user+650ms  Router generates routes from navigation → navigates to /dashboard
            │
T+user+660ms  pageProvider('main-dashboard') → GET /ui/pages/main-dashboard
            │
T+user+700ms  Capabilities and session responses arrive → cached
            │
T+user+900ms  Dashboard page response arrives → cached → page renders
            │
T+user+950ms  Schema requests for dashboard components (parallel)
            │
T+user+1200ms All schemas arrive → cached → full dashboard visible

TOTAL: ~1.2 seconds after login (dominated by network latency)
```

### Warm Start (Returning User)

```
T+0ms     main() begins
            │
T+10ms    Initialize Drift database (existing file, no migrations)
            │
T+15ms    Initialize flutter_secure_storage
            │
T+20ms    Create ProviderScope
            │
T+30ms    MaterialApp.router builds
            │
T+40ms    authProvider.build() runs
            ├── Checks secure storage → token found, not expired
            ├── State = Authenticated
            │
T+50ms    (PARALLEL, from cache — all instant):
            ├── capabilitiesProvider → reads from Drift → available in 5ms
            ├── navigationProvider → reads from Drift → available in 5ms
            ├── sessionProvider → reads from Drift → available in 5ms
            │
T+60ms    Router generates routes from cached navigation
            │
T+65ms    Sidebar renders from cached navigation data
            │
T+70ms    Navigates to /dashboard (or last-visited page)
            │
T+75ms    pageProvider('main-dashboard') → reads from Drift → available in 5ms
            │
T+80ms    PageRenderer builds dashboard from cached descriptor and schemas
            │
T+90ms    FULL DASHBOARD VISIBLE
            │
            ... (background, invisible to user) ...
            │
T+100ms   Background refresh: capabilities, navigation, session, current page (parallel)
            │
T+400ms   Responses arrive → caches updated → if anything changed, screen updates smoothly

TOTAL: ~90ms to full render (all from cache)
```

### Offline Start

```
T+0ms     main() begins
            │
T+10ms    Initialize Drift, secure storage
            │
T+30ms    authProvider → token found (cannot validate with server, but token exists)
            │
T+40ms    connectivityProvider → offline detected
            │
T+50ms    All providers read from cache (same as warm start)
            │
T+90ms    FULL UI VISIBLE from cache
            │
T+100ms   Stale cache banner: "Offline. Showing cached data."
            │
            ... user works with cached data ...
            │
T+???ms   Network returns → connectivityProvider emits online
            │
T+???+10ms  Background refreshes trigger automatically
            │
T+???+500ms  All caches updated → stale banner disappears
```

---

## Scalability

Scalability means the system's ability to handle growth. For Thesa UI, growth comes from several directions:

### Scaling Axis 1: More Backend Domains

**Scenario**: The company adds a new inventory management system. The BFF exposes new navigation items, pages, and schemas for inventory.

**Impact on frontend**: ZERO code changes.

```
Before:                          After:
Navigation:                      Navigation:
  Dashboard                        Dashboard
  Orders                           Orders
  Customers                        Customers
                                   Inventory    ← new, from BFF
                                     Products   ← new
                                     Warehouses ← new
```

The BFF adds inventory items to the navigation response. The frontend:
1. Background-refreshes navigation → new items cached
2. Sidebar rebuilds → inventory section appears
3. User clicks "Products" → `pageProvider('inventory-products')` activates
4. Page descriptor fetched and cached → table renders

No frontend developer was involved. No deployment was needed.

### Scaling Axis 2: More Pages Per Domain

**Scenario**: The orders domain grows from 3 pages to 30 pages (analytics, reports, configurations, etc.).

**Impact on frontend**: ZERO code changes. The `pageProvider(pageId)` family creates provider instances on demand. Whether there are 3 pages or 300, the same code handles them.

### Scaling Axis 3: More Complex Pages

**Scenario**: A dashboard page grows from 4 widgets to 40 widgets.

**Impact on frontend**: The UI Engine renders all 40 widgets. Performance strategies handle the load:
- Virtualization for scrollable areas
- RepaintBoundary for expensive widgets
- Lazy rendering for off-screen components

If 40 widgets still cause performance issues, the BFF can paginate them (e.g., load widgets in batches as the user scrolls).

### Scaling Axis 4: More Users

**Scenario**: User count grows from 100 to 10,000.

**Impact on frontend**: None. Each user runs their own instance of the app with their own local cache. The BFF handles the load of 10,000 concurrent users.

### Scaling Axis 5: More Data

**Scenario**: The orders table grows from 1,000 rows to 10 million rows.

**Impact on frontend**: None. Server-side pagination means the frontend always loads one page at a time (25-100 rows). The server handles the query optimization.

### Scaling Axis 6: New Component Types

**Scenario**: The company needs a Gantt chart component that the built-in ComponentRegistry does not support.

**Impact on frontend**: Register a new component plugin:

```
pluginRegistry.registerComponent(
  componentType: 'gantt_chart',
  builder: (descriptor, ref) => GanttChartWidget(descriptor: descriptor),
);
```

The core engine is unchanged. The BFF starts sending `"type": "gantt_chart"` in page descriptors, and the plugin renders them.

### Scaling Axis 7: New Workflow Types

**Scenario**: A new 15-step employee onboarding workflow is created.

**Impact on frontend**: ZERO code changes. The workflow engine handles any number of steps, any transition pattern, any rendering type. The BFF defines the workflow, and the engine executes it.

---

## Summary: What Grows vs. What Stays Fixed

| What Changes | Frontend Impact |
|---|---|
| New backend domain | None — BFF navigation update |
| New pages | None — BFF page descriptor |
| New form fields | None — BFF schema update |
| New table columns | None — BFF column descriptor |
| New actions/buttons | None — BFF action descriptor |
| New workflows | None — BFF workflow definition |
| New component type | Small — register a component plugin |
| Entirely new UX paradigm | Medium — develop a page plugin |
| Core rendering engine change | Large — modify ui_engine (rare) |

The first six rows — the most common changes in any enterprise — require **zero frontend code changes**. This is the core value proposition of the architecture.
