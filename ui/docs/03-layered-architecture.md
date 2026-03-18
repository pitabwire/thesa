# 3. Layered Architecture

## What Is a Layered Architecture?

A layered architecture is a way of organizing code so that each "layer" has a specific job and only talks to the layers directly above or below it. Think of it like a building:

- The **top floor** (Presentation) is what the user sees and interacts with
- The **middle floors** (UI Engine, State, Cache) process and manage data
- The **ground floor** (Networking) communicates with the outside world
- The **basement** (Plugins) provides special extensions

Each floor only knows about the floors immediately next to it. The top floor does not reach down to the ground floor directly — it goes through the middle floors.

## Why Layers?

Without layers, any part of the code can call any other part. This creates a tangled mess (often called "spaghetti code") where changing one thing breaks ten other things. Layers create clean boundaries:

- You can replace the database without changing the UI
- You can change the UI framework without touching the networking code
- You can test each layer independently
- New developers can understand one layer at a time

---

## The Six Layers

```
┌─────────────────────────────────────────────────────────┐
│  LAYER 1: PRESENTATION                                   │
│  "What the user sees"                                    │
├─────────────────────────────────────────────────────────┤
│  LAYER 2: UI ENGINE                                      │
│  "How BFF descriptions become widgets"                   │
├─────────────────────────────────────────────────────────┤
│  LAYER 3: STATE & CAPABILITY                             │
│  "What data the app is currently holding"                │
├─────────────────────────────────────────────────────────┤
│  LAYER 4: CACHE                                          │
│  "What data is saved on the device"                      │
├─────────────────────────────────────────────────────────┤
│  LAYER 5: NETWORKING                                     │
│  "How the app talks to the server"                       │
├─────────────────────────────────────────────────────────┤
│  LAYER 6: PLUGINS                                        │
│  "Custom extensions for specific domains"                │
└─────────────────────────────────────────────────────────┘
```

---

### Layer 1: Presentation Layer

**Job**: Render what the user sees and handle their interactions (taps, scrolls, typing).

**Contains**:
- The app shell (sidebar, top bar, breadcrumbs)
- Responsive layout logic (how things rearrange on different screen sizes)
- Page containers that host dynamic content
- Theme and design system application

**Does NOT contain**:
- Business logic of any kind
- Direct network calls
- Direct database access
- Knowledge of specific BFF endpoints

**Analogy**: The presentation layer is like the picture frame. It provides the structure and border, but the picture inside comes from elsewhere (the UI Engine).

**Talks to**: UI Engine Layer, State Layer

---

### Layer 2: UI Engine Layer

**Job**: Read descriptions from the BFF (via the State layer) and convert them into Flutter widgets.

**Contains**:
- The **Page Renderer**: Takes a page description and builds a widget tree
- The **Component Registry**: A lookup table that maps component type names (like "data_table" or "form") to widget builders
- The **Form Engine**: Builds dynamic forms from schema descriptions
- The **Table Engine**: Builds dynamic data tables from column descriptions
- The **Workflow Renderer**: Renders workflow steps as appropriate UIs
- The **Schema Resolver**: Follows references in schemas to assemble complete field definitions

**Does NOT contain**:
- Permanent data storage
- Network calls
- Business rules
- Hardcoded page layouts

**Analogy**: The UI Engine is like a 3D printer. You give it a blueprint (the BFF descriptor), and it produces a physical object (Flutter widgets). It does not design the blueprint — it just faithfully executes it.

**Talks to**: State Layer, Plugin Layer (for overrides)

---

### Layer 3: State & Capability Layer

**Job**: Hold the app's current data in memory and coordinate between the cache (local storage) and the network (server).

**Contains**:
- **Riverpod providers** for every piece of data the app needs:
  - Authentication state (who is logged in, their token)
  - Navigation tree (what menu items exist)
  - Page descriptors (what a specific page looks like)
  - Schemas (data structure definitions)
  - Permissions (what the user is allowed to do)
  - Workflow states (what step of a workflow the user is on)
  - Connectivity state (is the device online or offline?)
- **Cache-first orchestration logic**: "Read from cache first, then refresh from server"

**Does NOT contain**:
- Widget code
- SQL queries
- HTTP request construction
- Business domain logic

**Analogy**: The state layer is like a librarian. When someone asks for a book (data), the librarian first checks the local shelves (cache). If the book is there and not too old, the librarian hands it over immediately. Meanwhile, the librarian orders a fresh copy from the publisher (server) to replace the shelf copy later.

**Talks to**: Cache Layer (to read/write stored data), Networking Layer (to fetch fresh data)

---

### Layer 4: Cache Layer

**Job**: Persistently store data on the device so the app can work offline and start instantly.

**Contains**:
- **Drift database** with tables for navigation, pages, schemas, permissions, workflows, and UI decisions
- **DAOs** (Data Access Objects) — organized groups of database operations
- **Cache Coordinator** — decides whether to read from cache, fetch from network, or both
- **Cache Policy** — rules about how long data stays fresh (TTL = Time To Live)

**Does NOT contain**:
- Network calls
- Widget code
- State management logic
- Business logic

**Analogy**: The cache is like a refrigerator. Food (data) from the grocery store (server) is stored here for quick access. Each item has an expiration date (TTL). The refrigerator does not cook food — it just stores and retrieves it.

**Talks to**: The raw SQLite database on the device

---

### Layer 5: Networking Layer

**Job**: Send HTTP requests to the BFF server and receive responses.

**Contains**:
- **BFF Client**: Defines all BFF endpoints (e.g., `GET /ui/pages/{pageId}`)
- **dio instance**: The HTTP client that executes requests
- **Interceptors**: Middleware chain that adds auth tokens, handles caching headers, retries failures, and records performance metrics
- **Background refresh coordinator**: Schedules periodic data refreshes

**Does NOT contain**:
- Data storage
- Widget code
- State management
- Request business logic

**Analogy**: The networking layer is like a postal service. It knows how to send letters (requests) and receive packages (responses). It does not read the letters or decide what to do with the packages — it just delivers them.

**Talks to**: The BFF server over the internet

---

### Layer 6: Plugin / Extension Layer

**Job**: Allow developers to register custom UI components for specific domains without modifying the core platform.

**Contains**:
- **Plugin Registry**: A central directory where plugins register themselves
- **Plugin Interfaces**: Contracts that plugins must follow
- **Example plugins**: Demonstrations of how to build custom pages

**Does NOT contain**:
- Core platform logic
- Base component implementations
- Cache or networking code

**Analogy**: The plugin layer is like a set of custom picture frames. The platform provides standard frames for every picture, but if you want a gold frame for the Mona Lisa, you register a custom frame for that specific picture. All other pictures still use the standard frame.

**Talks to**: UI Engine Layer (the engine asks the plugin registry "is there a custom renderer for this?" before using the default one)

---

## Layer Communication Rules

These rules are **strict boundaries** that must never be violated:

| Rule | Meaning |
|---|---|
| Presentation → UI Engine ✅ | Presentation can ask the UI Engine to render things |
| Presentation → State ✅ | Presentation can read and watch state providers |
| Presentation → Cache ❌ | Presentation must NEVER directly access the database |
| Presentation → Networking ❌ | Presentation must NEVER make network calls directly |
| UI Engine → State ✅ | UI Engine reads state to get page descriptors and schemas |
| UI Engine → Cache ❌ | UI Engine must NEVER directly access the database |
| State → Cache ✅ | State reads from and writes to the cache |
| State → Networking ✅ | State fetches fresh data from the network |
| Cache → Networking ❌ | Cache does NOT make network calls. It is purely storage. |
| Networking → Cache ❌ | Networking does NOT write to the cache. It returns data to the State layer, which writes it. |

### Why This Strictness?

If the Presentation layer could call the network directly, you would have:
- Network calls scattered throughout widget code
- No centralized error handling
- No caching — every screen load hits the network
- Difficult testing — you would need to mock network calls in widget tests

By forcing all data through the State layer, we get:
- One place to add caching logic
- One place to handle errors
- Easy testing — mock the state provider, not the network
- Clear data flow that any developer can follow

---

## Visual Data Flow Through Layers

Here is how data flows when a user opens a page:

```
User taps "Orders" in sidebar
        │
        ▼
PRESENTATION: Navigates to /orders/list
        │
        ▼
STATE: pageProvider('orders-list') is activated
        │
        ▼
STATE: Asks Cache Coordinator for page descriptor
        │
        ├── CACHE: "I have it cached from 5 minutes ago" ──▶ Return cached data
        │                                                         │
        │                                                         ▼
        │                                              STATE: Provide to UI Engine
        │                                                         │
        │                                                         ▼
        │                                              UI ENGINE: Build widgets
        │                                                         │
        │                                                         ▼
        │                                              PRESENTATION: Render on screen
        │
        └── STATE: Also triggers background refresh
                │
                ▼
            NETWORKING: GET /ui/pages/orders-list
                │
                ▼
            STATE: Receives fresh data from BFF
                │
                ▼
            CACHE: Writes fresh data to database
                │
                ▼
            STATE: Drift stream emits update → Provider updates
                │
                ▼
            UI ENGINE: Rebuilds if data changed
                │
                ▼
            PRESENTATION: Screen updates seamlessly
```

The user sees the page instantly from cache. If the server has newer data, the screen updates smoothly in the background. The user never sees a loading spinner for cached pages.
