# 17. Performance Strategy

## Why Performance Matters

Enterprise users work with Thesa UI for hours every day. Even small delays compound:
- A 200ms delay on every page navigation means 2 minutes lost per 600 navigations
- A janky (stuttering) scroll makes large tables painful to use
- A 3-second initial load makes the app feel broken on every launch

The performance target: **every interaction should feel instant** (under 100ms for cached content, under 300ms for network-dependent content).

---

## Performance Challenges Unique to Thesa UI

Thesa UI faces challenges that a regular app does not:

1. **Unknown page complexity**: A BFF descriptor might contain 3 components or 300. The engine must handle both.
2. **Dynamic widget trees**: Widgets are constructed at runtime from descriptors, not statically compiled. This adds interpretation overhead.
3. **Large data sets**: Tables may contain tens of thousands of rows (server-paginated).
4. **Cross-platform rendering**: The same code runs on web (DOM-based), mobile (Skia/Impeller), and desktop (Impeller). Each has different performance characteristics.
5. **Cache synchronization**: Reading from Drift and reconciling with network data adds complexity.

---

## Strategy 1: Virtualized Lists

### The Problem

If a table has 100 visible rows, Flutter must build 100 row widgets. Each row might have 8 columns, each column a formatted widget. That is 800 widgets. If the user scrolls, all 100 rows are in memory even though only ~12 are visible on screen.

### The Solution

**Virtualization** means only building widgets for rows that are currently visible (plus a small buffer above and below for smooth scrolling):

```
Conceptual model:
  Total rows: 100
  Visible rows: 12
  Buffer rows: 3 above + 3 below
  Widgets in memory: 18

  As the user scrolls down:
    Row 1 scrolls out of view → its widget is destroyed
    Row 19 scrolls into view → its widget is created
    Total widgets in memory remains ~18
```

### Implementation

The `DynamicTable` uses `CustomScrollView` with `SliverList.builder`:

- `SliverList.builder` only calls the builder function for visible items
- Items that scroll out of view are garbage-collected
- This keeps memory usage constant regardless of total item count

### Impact

| Scenario | Without Virtualization | With Virtualization |
|---|---|---|
| 25 rows | 25 widgets (~fast) | 18 widgets (~fast) |
| 100 rows | 100 widgets (noticeable lag) | 18 widgets (fast) |
| 1,000 rows | 1,000 widgets (very slow) | 18 widgets (fast) |
| 10,000 rows | Impossible (freezes/crashes) | 18 widgets (fast) |

---

## Strategy 2: Debounced User Input

### The Problem

When a user types in a search box, each keystroke could trigger a network request:

```
User types: "A" → request 1
            "Al" → request 2
            "Ali" → request 3
            "Alic" → request 4
            "Alice" → request 5
```

Five requests, but only the last one matters. The first four are wasted and create unnecessary server load.

### The Solution

**Debouncing** delays the action until the user stops typing. A timer starts after each keystroke. If the user types again before the timer expires, the timer resets.

```
User types: "A" → start 300ms timer
            "l" → reset timer (200ms elapsed, not fired)
            "i" → reset timer
            "c" → reset timer
            "e" → reset timer
            ... 300ms passes with no typing ...
            → fire ONE request for "Alice"
```

### Where Debouncing Is Applied

| Input | Debounce Duration | Why |
|---|---|---|
| Search box | 300ms | Balance between responsiveness and request volume |
| Table filter controls | 300ms | Same as search |
| Form field validation | 150ms | Faster feedback, but still avoids validating on every keystroke |
| Window resize | 200ms | Avoid recalculating layout on every pixel of window resize |

---

## Strategy 3: Efficient Widget Rebuilds

### The Problem

In Flutter, when a provider's data changes, all widgets watching that provider rebuild. If a single provider holds the entire page state, changing one field rebuilds the entire page.

### The Solutions

**a) Granular providers**: Instead of one big `pageStateProvider`, use separate providers for separate concerns:
- `pageDescriptorProvider(pageId)` — the page layout (changes rarely)
- `tableDataProvider(tableId)` — the table rows (changes on page/sort/filter)
- `filterStateProvider(pageId)` — the active filters (changes on user input)
- `selectionProvider(tableId)` — selected rows (changes on checkbox clicks)

When the user toggles a checkbox, only the `selectionProvider` changes, and only the checkbox column rebuilds — not the entire table.

**b) `const` constructors**: Widgets that never change (static labels, icons, layout containers) are declared with `const` constructors. Flutter skips rebuilding `const` widgets entirely.

```
// This widget is NEVER rebuilt, even if its parent rebuilds:
const SizedBox(height: 16)

// This label is NEVER rebuilt:
const Text('Order #', style: TextStyle(fontWeight: FontWeight.bold))
```

**c) `select` on providers**: When a widget only needs part of a provider's data, it uses `select` to watch just that part:

```
// Instead of watching the entire page descriptor:
final descriptor = ref.watch(pageProvider(pageId));

// Watch only the title (rebuild only if title changes):
final title = ref.watch(pageProvider(pageId).select((d) => d.title));
```

If the page descriptor's components change but the title stays the same, the title widget does NOT rebuild.

**d) `RepaintBoundary`**: Heavy components (charts, complex tables) are wrapped in `RepaintBoundary`. This tells Flutter: "Even if the parent repaints, do not repaint this child unless its own state changed." This prevents a sidebar hover animation from causing a complex chart to repaint.

---

## Strategy 4: Schema Memoization

### The Problem

Schemas are used to build forms and tables. Resolving a schema (following `$ref` pointers, merging `allOf` definitions) is moderately expensive. If a schema is resolved every time a widget rebuilds, this adds up.

### The Solution

Resolved schemas are cached in the `schemaProvider`. Once a schema is resolved, the result is stored in memory (via Riverpod's `keepAlive`). Subsequent accesses return the cached result instantly.

```
First access to schema "order-summary":
  1. Read raw schema from Drift cache → 5ms
  2. Resolve $ref pointers → 10ms
  3. Merge allOf definitions → 5ms
  4. Total: 20ms
  5. Store resolved schema in provider state

Second access (same session):
  1. Provider already has resolved schema in memory → 0ms
```

---

## Strategy 5: Optimized Initial Load

### The Problem

When the app starts, it needs to load:
1. Auth state
2. Capabilities
3. Navigation
4. First page descriptor
5. Schemas for that page

If these happen sequentially, the total wait time is the sum of all requests.

### The Solution: Parallel Loading with Cache Priority

```
App Start
  │
  ├── (Parallel, from cache — all instant)
  │   ├── Read auth token from secure storage
  │   ├── Read capabilities from Drift
  │   ├── Read navigation from Drift
  │   └── Read last-visited page descriptor from Drift
  │
  ▼ All cached data available in <100ms
  │
  ├── Render UI immediately from cache
  │
  └── (Parallel, background — user does not wait)
      ├── Refresh capabilities from BFF
      ├── Refresh navigation from BFF
      └── Refresh current page from BFF
```

On a warm start (returning user), the UI is fully rendered within 100ms. Background refreshes happen silently.

On a cold start (first ever launch), the parallel network requests minimize wait time:

```
Sequential: auth (300ms) + capabilities (200ms) + navigation (250ms) + page (300ms) = 1,050ms
Parallel:   auth (300ms) then [capabilities + navigation + page in parallel] (300ms) = 600ms
```

Auth must come first (other requests need the token). Everything else runs in parallel.

---

## Strategy 6: Image and Asset Optimization

### Icon Loading

The BFF specifies Material icon names (e.g., `"icon": "shopping_cart"`). Material icons are bundled with Flutter — no network request needed.

If the BFF specifies custom icon URLs, they are:
1. Loaded lazily (only when the icon's widget is built)
2. Cached in memory after the first load
3. Show a placeholder during loading

### Chart Rendering

Charts are rendered using Flutter's Canvas API, not web-based chart libraries. This avoids the overhead of embedding web views.

For very complex charts on web, the rendering is deferred to the first idle frame (using `SchedulerBinding.instance.scheduleFrameCallback`) to avoid blocking the initial page paint.

---

## Strategy 7: Network Request Optimization

### Request Deduplication

If two widgets on the same page both need schema "order-summary", the deduplication interceptor ensures only ONE network request is made. Both widgets share the response.

### Request Cancellation

When the user navigates away from a page, all in-flight requests for that page are cancelled:
1. `pageProvider(pageId)` is auto-disposed
2. The dispose callback cancels the dio `CancelToken`
3. The server does not send a response → saves bandwidth
4. The cancelled request does not write stale data to cache

### Batching (Future Enhancement)

For pages with many schema references, the BFF could support batch requests:

```
POST /ui/schemas/batch
body: { ids: ["order-summary", "customer-profile", "address"] }
```

One request instead of three. The networking layer can batch requests that are initiated within the same frame.

---

## Performance Budgets

A performance budget is a target that, if exceeded, triggers an alert:

| Metric | Budget | Measured By |
|---|---|---|
| **First paint (cached)** | < 100ms | Time from app start to first meaningful UI |
| **First paint (cold)** | < 2,000ms | Time from app start to first meaningful UI (no cache) |
| **Page navigation (cached)** | < 100ms | Time from tap to new page visible |
| **Page navigation (network)** | < 500ms | Time from tap to new page visible (cache miss) |
| **Table scroll FPS** | > 55 FPS | Frames per second during continuous scroll |
| **Form field response** | < 50ms | Time from keystroke to character appearing on screen |
| **Action button response** | < 100ms | Time from tap to visible loading indicator |
| **Memory (per page)** | < 50 MB | Peak memory usage for a single page |
| **JS bundle size (web)** | < 2 MB compressed | Initial download size for web deployment |

These budgets are measured by the telemetry system and tracked over time. Regressions trigger alerts.

---

## Platform-Specific Considerations

| Platform | Performance Note |
|---|---|
| **Web** | Dart compiles to JavaScript. Use `--release` mode for production (tree-shaking, minification). Avoid heavy widget trees that stress the DOM. Use `CanvasKit` renderer for complex visuals, `HTML` renderer for text-heavy pages. |
| **Android** | Impeller rendering engine provides consistent frame rates. Profile on mid-range devices (not just flagships). |
| **iOS** | Impeller is the default. Metal GPU support ensures smooth rendering. |
| **Desktop** | Large screens mean more visible widgets. Ensure virtualization is active for all scrollable lists. |
