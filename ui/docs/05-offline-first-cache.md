# 5. Offline-First Cache Architecture

## What Does "Offline-First" Mean?

Most apps work like this:
1. User opens the app
2. App shows a loading spinner
3. App fetches data from the server
4. App displays the data
5. If the server is unreachable → error screen

An "offline-first" app works differently:
1. User opens the app
2. App immediately shows data from local storage (no spinner)
3. App quietly checks the server for updates in the background
4. If newer data exists → smoothly update the display
5. If the server is unreachable → everything still works, just with slightly older data

The user experience difference is dramatic. Instead of staring at spinners, users see their content instantly.

---

## Why Is This Critical for Thesa UI?

Thesa UI is an enterprise admin tool. Enterprise users expect:

- **Instant response**: "I clicked Orders — show me the orders screen NOW, not in 3 seconds"
- **Reliability**: "The VPN dropped for 30 seconds; I should still be able to work"
- **Consistency**: "The page should look the same every time I open it, not flash and rearrange"

Without caching, every page navigation would:
1. Show a blank screen or skeleton
2. Make 2-5 API calls (navigation, page descriptor, schemas, permissions)
3. Wait for all responses (often 500ms-2s depending on network)
4. Render the page

With caching, navigation feels instant — the page appears within 100 milliseconds.

---

## What Gets Cached?

Everything the BFF tells the UI gets stored locally:

| Cache Type | What It Stores | Example |
|---|---|---|
| **Navigation Cache** | The sidebar menu tree | "Orders, Products, Settings, ..." |
| **Page Cache** | Full page layouts with all components | "The orders page has a search bar, a filter panel, and a data table" |
| **Schema Cache** | Data structure definitions | "An Order has fields: id, customer, total, status, date" |
| **Permission Cache** | What the user is allowed to see and do | "User can view orders but cannot delete them" |
| **Workflow State** | Progress in multi-step workflows | "User is on step 3 of 5 in the refund process" |
| **UI Decision Cache** | Server decisions about what to show | "For this user role, show the analytics dashboard with these 4 widgets" |

---

## The Database Schema

The cache uses Drift (SQLite). Each cache type is a database table. Here is what each table looks like:

### Navigation Cache Table

| Column | Type | Purpose |
|---|---|---|
| `id` | Text (primary key) | Unique identifier for the nav tree |
| `payload` | Text (JSON) | The actual navigation tree data, stored as JSON |
| `etag` | Text | A fingerprint from the server — used to check if data has changed |
| `version` | Integer | A version number that increments when the nav structure changes |
| `fetched_at` | Integer (timestamp) | When this data was last downloaded from the server |
| `expires_at` | Integer (timestamp) | When this data is considered "stale" and should be refreshed |
| `stale` | Boolean | Whether this data is known to be outdated |

All other cache tables follow the same pattern. This consistency makes the cache easy to understand and maintain.

### Special Fields Explained

**`payload`** — This is the actual data from the BFF, stored as a JSON string. When reading from cache, the app parses this JSON back into Dart objects. Storing JSON means we can cache any BFF response without changing the table structure.

**`etag`** — Short for "Entity Tag." It is a value the server assigns to each response. Think of it as a version fingerprint. When refreshing, the app sends the stored ETag to the server with the message: "I already have version X. Has anything changed?" If the server responds "304 Not Modified," no data transfer is needed — saving bandwidth and time.

**`fetched_at`** / **`expires_at`** — Together, these implement TTL (Time To Live). The difference between them is the TTL duration. For example, if navigation has a 15-minute TTL:
- `fetched_at`: 10:00:00 AM
- `expires_at`: 10:15:00 AM
- At 10:14:00 → data is "fresh" → use it directly
- At 10:16:00 → data is "stale" → use it but also refresh in background

**`stale`** — A flag that is set to `true` when the app detects that cached data is outdated. A stale cache entry is still usable — it is just accompanied by a visual indicator ("Data may be outdated. Last updated 15 minutes ago.").

### Schema Cache — Special: Reference Counting

The schema cache has an extra column:

| Column | Type | Purpose |
|---|---|---|
| `ref_count` | Integer | How many pages currently reference this schema |

Why? Schemas are shared. The "Order" schema might be used by:
- The orders list page (for table columns)
- The order detail page (for the edit form)
- The order creation page (for the new order form)

When a user navigates away from the orders list, we might consider evicting that page's data. But we must NOT evict the Order schema if other pages still need it. The `ref_count` tracks this:
- User visits orders list → Order schema ref_count goes from 0 to 1
- User visits order detail → ref_count goes from 1 to 2
- User leaves orders list → ref_count goes from 2 to 1
- User leaves order detail → ref_count goes from 1 to 0
- ref_count is 0 → schema can be evicted (but typically is just refreshed instead)

---

## Cache Lifecycle: Stale-While-Revalidate

This is the core caching strategy. The name means: "Use stale data while revalidating (refreshing) in the background."

### The Full Flow — Step by Step

```
Step 1: Something triggers a data need
        (User navigates to a page, or a provider is first accessed)
             │
             ▼
Step 2: The State Provider asks the Cache Coordinator for data
             │
             ▼
Step 3: Cache Coordinator checks the local database
             │
             ├── CASE A: Cache has fresh data (within TTL)
             │     → Return it immediately
             │     → Done. No network request needed.
             │
             ├── CASE B: Cache has stale data (past TTL, but data exists)
             │     → Return the stale data immediately (user sees content instantly)
             │     → Mark data as "stale" in the UI (subtle banner)
             │     → Fire a background network request to get fresh data
             │     → When fresh data arrives:
             │         → Write it to the database
             │         → Database stream auto-notifies the provider
             │         → Provider auto-notifies the widget
             │         → Widget rebuilds with fresh data (seamless update)
             │
             └── CASE C: Cache is empty (first visit ever, or data was evicted)
                   → Show a loading skeleton (this is the only time the user sees a spinner)
                   → Fire a network request
                   → When data arrives:
                       → Write it to the database
                       → Stream notifies provider → widget renders
```

### Why Not Just Always Fetch from the Server?

Because:
1. **Speed**: Cache reads take <10ms. Network requests take 200ms-2000ms.
2. **Reliability**: The network can fail. The cache is always there.
3. **Bandwidth**: ETags mean most refreshes transfer zero bytes (304 responses).
4. **User experience**: No spinners, no content jumps, no empty screens.

---

## The Cache Coordinator

The Cache Coordinator is the "decision-maker" that implements the stale-while-revalidate strategy. Every data provider talks to the coordinator instead of directly accessing the database or network.

### Decision Matrix

This table shows exactly what the coordinator does for every combination of cache state and network state:

| Cache State | Network State | What Happens |
|---|---|---|
| **Empty** | **Online** | Show loading skeleton → Fetch from server → Cache → Render |
| **Empty** | **Offline** | Show empty state with "No data available, connect to network" + retry button |
| **Fresh** (within TTL) | **Online** | Return cached data. No network request. |
| **Fresh** (within TTL) | **Offline** | Return cached data. No network request. |
| **Stale** (past TTL) | **Online** | Return cached data immediately. Fire background refresh. Update when done. |
| **Stale** (past TTL) | **Offline** | Return cached data. Show "Data may be outdated" banner. |
| **Any** | **Session expired** (401) | Clear permission cache. Redirect user to login screen. |

### How the Coordinator Detects Freshness

```
isFresh(cacheEntry):
    return DateTime.now() < cacheEntry.expiresAt

isStale(cacheEntry):
    return DateTime.now() >= cacheEntry.expiresAt

isEmpty(cacheKey):
    return database.find(cacheKey) == null
```

---

## TTL (Time To Live) Configuration

Each type of cached data has a different TTL, chosen based on how often the data changes and how sensitive it is:

| Cache Type | Default TTL | Why This Duration |
|---|---|---|
| **Navigation** | 15 minutes | Menu items change rarely (new pages are added maybe weekly). 15 minutes is a good balance between freshness and avoiding unnecessary refreshes. |
| **Page descriptors** | 10 minutes | Page layouts might change when developers deploy updates. 10 minutes means users see new layouts within minutes. |
| **Schemas** | 30 minutes | Schemas (field definitions) are very stable — they change only when data models are modified, which is rare. 30 minutes minimizes unnecessary schema fetches. |
| **Permissions** | 5 minutes | Security-sensitive. If an admin revokes a user's access, the user should lose that access within 5 minutes at most. |
| **UI decisions** | 10 minutes | Matches page descriptors since they are closely related. |
| **Workflow state** | No expiry | Workflow progress is user-driven. A user might start a workflow, close the app, and come back a day later. The state must persist until the workflow is completed or explicitly abandoned. |

These defaults are configurable. An enterprise deployment might tighten permission TTL to 2 minutes or relax schema TTL to 60 minutes.

---

## ETag / Version-Based Invalidation

### What Is an ETag?

An ETag (Entity Tag) is a string that the server assigns to a specific version of a response. Think of it like a hash or fingerprint:

- Server sends: `ETag: "abc123"` with the navigation data
- Client stores: navigation data + etag "abc123"
- Next refresh: Client sends `If-None-Match: "abc123"`
- Server checks: "Has my navigation data changed since abc123?"
  - If NO → Server sends `304 Not Modified` (very small response, no body)
  - If YES → Server sends `200 OK` with new data and a new ETag

### Why ETags Matter

Without ETags, every refresh downloads the full response body even if nothing changed. For a complex page descriptor that might be 50KB of JSON, this wastes bandwidth and processing time.

With ETags:
- 304 responses are tiny (~100 bytes)
- No JSON parsing needed for 304s
- Server can respond faster (it only checks the version, not serializing the full response)

### How It Works in the Interceptor Chain

```
Request goes out:
  1. ETagInterceptor checks: "Do I have a cached ETag for this URL?"
  2. If yes → adds header: If-None-Match: "abc123"
  3. Request hits the server

Response comes back:
  A) 304 Not Modified:
     → ETagInterceptor says "data hasn't changed"
     → Cache coordinator touches fetched_at (resets TTL)
     → Clears the stale flag
     → No payload parsing, no state update
     → Very fast

  B) 200 OK with new ETag:
     → New data is parsed
     → Cache coordinator writes new data + new ETag to database
     → Database stream notifies provider → widget rebuilds
```

### Global Version Invalidation

The capabilities endpoint (`GET /ui/capabilities`) returns a global version number. If this number changes, it means the BFF has been updated in a way that might affect all cached data. When this is detected:

1. A single database transaction marks ALL cache entries as stale
2. Background refreshes are triggered for navigation, permissions, and the current page
3. The user's current view continues working from stale cache
4. Updated data flows in smoothly as refreshes complete

---

## Schema Reuse and Reference Counting

### The Problem

A large enterprise app might have 200 pages, but only 50 unique schemas. Many pages reuse the same schemas:

- "Customer" schema: used by customer list, customer detail, order detail (customer reference), invoice detail (billing customer)
- "Money" schema: used by orders, invoices, payments, refunds, reports

If we evicted a schema every time a page using it was navigated away from, we would constantly re-fetch the same schemas.

### The Solution: Reference Counting

Every time a page is loaded and it references a schema:
```
schemaCache.incrementRefCount('customer-schema')
```

Every time a page is unloaded:
```
schemaCache.decrementRefCount('customer-schema')
```

Schemas with `ref_count > 0` are NEVER evicted. They may be refreshed (if stale), but the old data stays available during the refresh.

Schemas with `ref_count == 0` can be evicted during cache cleanup, but in practice they are usually kept anyway (they are small and useful).

---

## Partial Invalidation

Sometimes, only part of the cache needs to be refreshed. Examples:

| Trigger | What Gets Invalidated |
|---|---|
| User logs in with a different account | All permission caches, all UI decisions |
| BFF global version changes | All navigation, all pages, all schemas |
| A specific page is modified on the server | Only that page's cache entry |
| A schema is updated | That schema + all pages that reference it |
| User starts a workflow | Only the workflow state entry |

Partial invalidation is efficient — it avoids throwing away data that is still valid. The cache coordinator handles this by accepting invalidation rules:

```
invalidate(cacheType: 'page', key: 'orders-list')  → only one entry
invalidate(cacheType: 'page')                       → all page entries
invalidateAll()                                     → everything (nuclear option, used on logout)
```

---

## Practical Example: What Happens When a User Opens the App

### First Time Ever (Cold Start)

```
1. App starts
2. Database is empty (first launch)
3. Auth check: no stored token → show login screen
4. User logs in → token stored in secure storage
5. capabilitiesProvider activates:
   - Cache: empty → fetch from network
   - Loading skeleton shown
   - BFF responds → data cached → render
6. navigationProvider activates:
   - Cache: empty → fetch from network
   - Sidebar shows skeleton
   - BFF responds → navigation cached → sidebar renders with real items
7. User sees the first page:
   - pageProvider('dashboard') activates
   - Cache: empty → fetch from network
   - Content area shows skeleton
   - BFF responds → page descriptor cached → schemas fetched → cached → page renders

Total time: ~2-3 seconds (dominated by network calls)
```

### Second Visit (Warm Start)

```
1. App starts
2. Database has data from last time
3. Auth check: stored token is valid
4. capabilitiesProvider activates:
   - Cache: has data from 3 hours ago (stale, TTL is 15 min)
   - Return cached capabilities immediately
   - Background: refresh from network
5. navigationProvider activates:
   - Cache: has data from 3 hours ago (stale)
   - Sidebar renders immediately from cache
   - Background: refresh from network
6. User sees the dashboard:
   - pageProvider('dashboard') activates
   - Cache: has page descriptor (stale)
   - Page renders immediately from cache
   - Background: refresh from network → seamless update

Total time: <100ms to first render (all from cache)
Background refreshes complete in 500ms-2s, updating any changed data seamlessly
```

### Offline Visit

```
1. App starts
2. No network connection detected
3. Auth check: stored token (cannot validate with server, but token exists → proceed)
4. All providers read from cache
5. Everything renders immediately
6. Stale cache banner appears: "Offline. Showing cached data."
7. When connection returns → background refreshes fire automatically
```
