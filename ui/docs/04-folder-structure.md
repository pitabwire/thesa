# 4. Folder Structure

## Why Folder Structure Matters

A well-organized folder structure is like a well-organized filing cabinet. When a new developer joins the team, they should be able to find any file within seconds just by reading folder names. When a bug is reported in "the table on the orders page," the developer should know exactly where to look.

A bad folder structure leads to:
- Files dumped in random locations
- Developers creating duplicate files because they cannot find existing ones
- Long onboarding times for new team members
- Confusion about where new code should go

---

## Top-Level Structure

```
lib/
├── main.dart              # The starting point of the entire application
├── app/                   # App-level concerns: routing, shell layout, responsive
├── core/                  # Shared data models, error types, constants
├── networking/            # Everything related to server communication
├── cache/                 # Everything related to local data storage
├── state/                 # Riverpod providers (the app's brain)
├── ui_engine/             # The dynamic rendering engine
├── plugins/               # Custom domain-specific extensions
├── design_system/         # Colors, fonts, spacing, reusable styled components
├── telemetry/             # Performance monitoring and logging
└── shared_widgets/        # Small reusable widgets used across the whole app
```

### Rule of Thumb

If you are wondering "where does this file go?", ask yourself:

| Question | Folder |
|---|---|
| Does it define how data looks when it comes from the server? | `core/models/` |
| Does it make a network request? | `networking/` |
| Does it read/write to the local database? | `cache/` |
| Does it hold data in memory and coordinate cache + network? | `state/` |
| Does it convert BFF descriptions into widgets? | `ui_engine/` |
| Does it define the sidebar, top bar, or shell layout? | `app/shell/` |
| Does it define a route or navigation guard? | `app/router/` |
| Is it a small, reusable UI piece like a loading spinner? | `shared_widgets/` |
| Is it a color, font, or spacing constant? | `design_system/` |
| Is it a custom page for a specific business domain? | `plugins/` |

---

## Detailed Breakdown

### `main.dart`

This is the entry point. When the app starts, Dart runs `main()` in this file. It does three things:

1. Initializes the local database (Drift)
2. Initializes secure storage (for auth tokens)
3. Creates the `ProviderScope` (Riverpod's container) and launches the app

Think of it as turning the ignition key in a car.

---

### `app/` — Application Shell and Navigation

This folder contains everything about the app's "frame" — the outer structure that stays visible as the user navigates between pages.

```
app/
├── app.dart                        # The MaterialApp.router widget (the top-level app widget)
├── router/
│   ├── app_router.dart             # Creates the GoRouter with all routes
│   ├── dynamic_route_builder.dart  # Converts BFF navigation data into GoRoute objects
│   └── route_guards.dart           # Checks like "is the user logged in?" before allowing navigation
└── shell/
    ├── app_shell.dart              # The StatefulShellRoute: sidebar on the left, content on the right
    ├── sidebar/
    │   ├── sidebar.dart            # The sidebar widget itself
    │   ├── sidebar_item.dart       # A single item in the sidebar (icon + label)
    │   └── sidebar_state.dart      # Tracks whether the sidebar is expanded or collapsed
    ├── breadcrumb/
    │   └── breadcrumb_bar.dart     # Shows the path: "Home > Orders > Order #1234"
    └── responsive/
        └── layout_breakpoints.dart # Defines screen size categories (phone, tablet, desktop)
```

**`app.dart`**: The root widget. Wraps the entire app in a `MaterialApp.router` which connects to go_router. Applies the theme. This file is small — it delegates everything to other modules.

**`router/app_router.dart`**: Creates the `GoRouter` instance. The key thing: routes are NOT hardcoded here. Instead, it reads from `navigationProvider` and calls `DynamicRouteBuilder` to generate routes from BFF data.

**`router/dynamic_route_builder.dart`**: This is the "translator" that reads BFF navigation descriptors (JSON data describing menu items and their paths) and creates `GoRoute` objects that Flutter's router can understand.

**`router/route_guards.dart`**: Before allowing navigation to a page, these guards check:
- Is the user authenticated? If not → redirect to login
- Does the user have permission for this page? If not → redirect to an error page

**`shell/app_shell.dart`**: Uses `StatefulShellRoute` to create the layout where the sidebar is persistent and the content area changes. The sidebar does not rebuild when you switch pages — only the content area does.

**`shell/sidebar/`**: The sidebar is built dynamically from BFF navigation data. It supports:
- Expandable/collapsible groups
- Nested menu items
- Icons from the BFF
- Bottom-positioned items (like Settings, Logout)
- Collapsing to icons-only on smaller screens

**`shell/breadcrumb/breadcrumb_bar.dart`**: Shows the user where they are in the navigation hierarchy. Built automatically from the current route path.

**`shell/responsive/layout_breakpoints.dart`**: Defines the screen size categories:
- Phone: < 600px wide
- Tablet: 600-960px
- Laptop: 960-1200px
- Desktop: 1200-1600px
- Wide: > 1600px

Other parts of the app reference these breakpoints to decide layout.

---

### `core/` — Shared Foundations

Contains data structures and definitions used across every other folder.

```
core/
├── models/                          # Data classes for everything the BFF sends
│   ├── capability.dart              # "What features are available"
│   ├── navigation.dart              # "What menu items exist"
│   ├── page_descriptor.dart         # "What does this page look like"
│   ├── component_descriptor.dart    # "What does this single UI element look like"
│   ├── schema.dart                  # "What fields does this data type have"
│   ├── action_descriptor.dart       # "What actions can the user take"
│   ├── workflow_descriptor.dart     # "What steps does this workflow have"
│   ├── permission.dart              # "What is the user allowed to do"
│   └── ui_metadata.dart             # "Extra display hints (colors, icons, etc.)"
├── errors/
│   ├── app_error.dart               # Defines all error types the app can encounter
│   └── error_handler.dart           # Central error processing and reporting
└── constants/
    └── bff_endpoints.dart           # URL paths for all BFF endpoints
```

**`models/`**: Every JSON response from the BFF is represented by a Dart class in this folder. These classes are generated using `freezed` and `json_serializable`, so they are immutable and have automatic JSON parsing. They are **pure data** — no logic, no side effects.

**`errors/`**: Defines a hierarchy of error types (network error, permission error, parse error, etc.) so that every part of the app handles errors consistently.

**`constants/bff_endpoints.dart`**: A single file listing every BFF URL path. If the BFF changes a URL, you update it in one place.

---

### `networking/` — Server Communication

```
networking/
├── bff_client.dart                  # The interface defining all BFF API calls
├── dio_factory.dart                 # Creates and configures the dio HTTP client
├── interceptors/
│   ├── auth_interceptor.dart        # Adds auth tokens to every request
│   ├── etag_interceptor.dart        # Adds caching headers (If-None-Match)
│   ├── retry_interceptor.dart       # Retries failed requests with backoff
│   ├── dedup_interceptor.dart       # Prevents duplicate simultaneous requests
│   └── telemetry_interceptor.dart   # Records request timing and errors
└── background_refresh.dart          # Coordinates periodic data refreshes
```

**`bff_client.dart`**: Annotated with retrofit. Defines every endpoint:
- `GET /ui/capabilities` → returns `Capabilities`
- `GET /ui/navigation` → returns `NavigationTree`
- `GET /ui/pages/{pageId}` → returns `PageDescriptor`
- etc.

You never write HTTP code manually. retrofit generates it from these annotations.

**`dio_factory.dart`**: Creates the `Dio` instance and attaches all interceptors in the correct order. Think of this as assembling a pipeline — every request flows through all interceptors sequentially.

**`interceptors/`**: Each interceptor is a small, focused piece of middleware:

- **auth_interceptor.dart**: Before every request, adds the user's auth token. If a response comes back as 401 (unauthorized), it attempts to refresh the token and retry.
- **etag_interceptor.dart**: Before a request, checks if there is a cached ETag for this endpoint. If so, adds `If-None-Match` header. If the server responds 304 (not modified), the interceptor short-circuits — no need to parse the response body.
- **retry_interceptor.dart**: If a request fails due to network issues, waits and retries. Uses "exponential backoff" — wait 1 second, then 2, then 4, then give up. Prevents hammering a struggling server.
- **dedup_interceptor.dart**: If two parts of the app request the same URL at the same time, only one actual HTTP request is made. Both callers share the result.
- **telemetry_interceptor.dart**: Records how long each request takes, whether it succeeded, and the response status code. This data feeds into the telemetry system.

**`background_refresh.dart`**: A coordinator that periodically refreshes cached data. For example, every 15 minutes, it re-fetches the navigation tree. Every 5 minutes, it re-fetches permissions. This runs silently in the background.

---

### `cache/` — Local Data Storage

```
cache/
├── database/
│   ├── app_database.dart            # The Drift database definition (all tables)
│   ├── app_database.g.dart          # Generated code (do not edit)
│   └── tables/
│       ├── navigation_cache.dart    # Table for cached navigation data
│       ├── page_cache.dart          # Table for cached page descriptors
│       ├── schema_cache.dart        # Table for cached schemas
│       ├── permission_cache.dart    # Table for cached permissions
│       ├── workflow_state.dart      # Table for in-progress workflow data
│       └── ui_decision_cache.dart   # Table for cached UI decisions
├── daos/
│   ├── navigation_dao.dart          # Database operations for navigation
│   ├── page_dao.dart                # Database operations for pages
│   ├── schema_dao.dart              # Database operations for schemas
│   └── workflow_dao.dart            # Database operations for workflows
├── cache_coordinator.dart           # Decision-maker: use cache, network, or both?
└── cache_policy.dart                # TTL rules and staleness detection
```

**`database/tables/`**: Each file defines a Drift table. A table is like a spreadsheet:
- `navigation_cache` has columns: id, payload (the JSON), etag, version, fetched_at, expires_at, stale
- `page_cache` has similar columns for page descriptors
- etc.

**`daos/`**: DAO stands for "Data Access Object." Each DAO groups related database operations:
- `navigation_dao.dart` has methods like `watchNavigation()`, `saveNavigation()`, `clearNavigation()`
- `page_dao.dart` has methods like `watchPage(pageId)`, `savePage(pageId, data)`, `invalidateAllPages()`

**`cache_coordinator.dart`**: The "brain" of caching. When a provider asks for data, the coordinator decides:
1. Is the data in the cache? If not → must fetch from network
2. Is the cached data fresh (within TTL)? If yes → return it, skip network
3. Is the cached data stale but we are offline? → return it, show stale warning
4. Is the cached data stale and we are online? → return it immediately, then refresh in the background

**`cache_policy.dart`**: Defines how long each type of data stays "fresh":
- Navigation: 15 minutes
- Schemas: 30 minutes
- Permissions: 5 minutes (because security changes should propagate quickly)

---

### `state/` — Riverpod Providers

```
state/
├── auth/
│   ├── auth_provider.dart           # Login state, token, logout
│   └── session_provider.dart        # Current user info, permissions
├── capabilities/
│   └── capabilities_provider.dart   # What features are enabled
├── navigation/
│   └── navigation_provider.dart     # The menu tree
├── pages/
│   └── page_provider.dart           # Page descriptors (one per page, using family)
├── schemas/
│   └── schema_provider.dart         # Schema definitions (one per schema, using family)
├── actions/
│   └── action_provider.dart         # Action execution state
├── workflows/
│   └── workflow_provider.dart       # Workflow progress tracking
└── connectivity/
    └── connectivity_provider.dart   # Online/offline detection
```

Each provider follows the same pattern:
1. On first access: read from cache (instant)
2. If cache is empty or stale: trigger network fetch
3. When network response arrives: write to cache
4. Cache write triggers Drift stream → provider updates → widgets rebuild

**`page_provider.dart`** uses Riverpod's `family` feature. Instead of one provider per page, there is one definition that creates instances on demand:
- `pageProvider('orders-list')` → one instance
- `pageProvider('dashboard')` → another instance
- `pageProvider('settings')` → yet another

Each instance manages its own cache lifecycle independently.

---

### `ui_engine/` — The Dynamic Rendering Engine

```
ui_engine/
├── page_renderer.dart               # Entry point: takes a page descriptor, outputs widgets
├── component_registry.dart          # Maps component type strings to widget builders
├── components/
│   ├── data_table/                  # Dynamic table widgets
│   ├── forms/                       # Dynamic form widgets
│   ├── cards/                       # Card display widgets
│   ├── metrics/                     # Number/KPI display widgets
│   ├── charts/                      # Basic chart widgets
│   ├── search/                      # Search bar widget
│   ├── status/                      # Status badge widget
│   ├── actions/                     # Action button widgets
│   └── layout/                      # Layout containers (sections, grids, tabs)
├── workflows/
│   ├── workflow_renderer.dart       # Renders the current workflow step
│   ├── workflow_stepper.dart        # Step indicator (step 1 → 2 → 3)
│   └── workflow_state_machine.dart  # Manages workflow transitions
└── schemas/
    └── schema_resolver.dart         # Resolves $ref pointers in schemas
```

This is the heart of the application. The `page_renderer.dart` is the main entry point: it receives a `PageDescriptor` and walks through its components, asking the `ComponentRegistry` "what widget should I use for this component type?" For each component, the registry returns a builder function that produces the appropriate Flutter widget.

---

### `plugins/` — Custom Extensions

```
plugins/
├── plugin_registry.dart             # Central registry where plugins register
├── plugin_interface.dart            # Contracts that plugins must implement
└── examples/
    └── custom_order_page.dart       # Example: custom UI for order details
```

This folder is intentionally small. Most of the time, plugins are developed by other teams and live in their own packages. They register themselves at app startup by calling methods on the `PluginRegistry`.

---

### `design_system/` — Visual Design Tokens

```
design_system/
├── tokens/
│   ├── colors.dart                  # Color palette (primary, secondary, status colors)
│   ├── typography.dart              # Font sizes, weights, line heights
│   └── spacing.dart                 # Margin and padding scale (4px, 8px, 16px, 24px, ...)
├── theme/
│   ├── app_theme.dart               # Builds Flutter ThemeData from tokens
│   ├── dark_theme.dart              # Dark mode variant
│   └── theme_extensions.dart        # Custom ThemeExtension for app-specific styles
└── components/
    ├── app_button.dart              # Styled button
    ├── app_card.dart                # Styled card
    ├── app_chip.dart                # Styled chip/tag
    └── app_dialog.dart              # Styled dialog
```

**Design tokens** are named values for visual properties. Instead of writing `Color(0xFF1E88E5)` everywhere, you write `AppColors.primary`. If the brand color changes, you update one file.

---

### `telemetry/` — Monitoring and Logging

```
telemetry/
├── telemetry_service.dart           # Records structured events
├── performance_monitor.dart         # Measures page render times
└── otel_exporter.dart               # Formats events for OpenTelemetry
```

---

### `shared_widgets/` — Reusable UI Pieces

```
shared_widgets/
├── loading_overlay.dart             # A semi-transparent overlay with a spinner
├── error_boundary.dart              # Catches errors in child widgets, shows error card
├── permission_gate.dart             # Hides children if user lacks permission
├── stale_cache_banner.dart          # "Data may be outdated" warning banner
└── empty_state.dart                 # "No items found" placeholder
```

These are small, general-purpose widgets used by multiple features. They do NOT contain business logic.
