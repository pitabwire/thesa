# 2. Technology Choices and Justifications

## Overview

This section explains every major library and tool used in Thesa UI, why it was chosen over alternatives, and what role it plays. If you are new to Flutter development, this will help you understand the "why" behind the tech stack.

---

## 2.1 Flutter (The Framework)

### What Is Flutter?

Flutter is Google's open-source framework for building applications that run on multiple platforms (web, iOS, Android, Windows, macOS, Linux) from a single codebase. It uses the Dart programming language.

### Why Flutter?

- **Single codebase**: Write once, deploy to web, desktop, tablet, and phone
- **Rich widget library**: Comes with hundreds of pre-built UI components
- **High performance**: Compiles to native code (not a web wrapper)
- **Strong typing**: Dart catches many errors at compile time, before the app runs
- **Enterprise adoption**: Used by Google, BMW, Alibaba, and many large companies

### What Are the Alternatives?

| Alternative | Why Not |
|---|---|
| React Native | Web support is weaker; desktop support is community-driven, not official |
| Kotlin Multiplatform | UI layer is still per-platform; no single widget system |
| Native per platform | 4-5 separate codebases to maintain — impractical for this project |

---

## 2.2 Riverpod 3.0 (State Management)

### What Is State Management?

"State" is any data your app needs to remember while running: the current user, which page they are on, the contents of a form, the list of items in a table, whether a sidebar is open or closed.

"State management" is the system that stores this data, shares it between parts of the app, and ensures the screen updates when the data changes.

### What Is Riverpod?

Riverpod is a state management library for Flutter. It provides "providers" — named containers that hold pieces of state. Any widget in the app can read from a provider, and when the provider's data changes, the widget automatically redraws itself.

### Why Riverpod Over Bloc?

Bloc is the other major state management option in Flutter. Here is why Riverpod was chosen:

#### Reason 1: Dynamic Pages Are Effortless

Thesa UI has an **unknown number of pages** — the BFF could define 5 pages or 500. With Riverpod, we write one provider definition and use "family" parameters to create unlimited instances:

```
pageProvider('orders-list')    → creates a provider for the orders page
pageProvider('dashboard')      → creates a different provider for the dashboard
pageProvider('settings')       → creates yet another for settings
```

With Bloc, we would need to manually create, register, and dispose a separate Bloc instance for each page. For a dynamic app, this is a significant amount of boilerplate code.

#### Reason 2: Built-In Cache Control

Riverpod has built-in concepts for cache management:

- **`keepAlive`**: Tells a provider "don't throw away this data, even if nobody is looking at it right now." Perfect for navigation and schema caches that should persist.
- **`invalidateSelf`**: Tells a provider "your data is outdated, please refresh." Used when the BFF sends new data.
- **Auto-dispose**: When a user navigates away from a page, Riverpod automatically cleans up that page's provider. No memory leaks.

Bloc has none of these built-in. You would need to build them yourself.

#### Reason 3: Code Generation

With the `@riverpod` annotation, Riverpod generates boilerplate code automatically. You write the business logic; Riverpod generates the wiring.

#### Reason 4: Dependency Injection

Providers can depend on other providers. Riverpod tracks these dependencies and ensures everything updates in the correct order. For example:

```
sessionProvider depends on authProvider
navigationProvider depends on sessionProvider
routerProvider depends on navigationProvider
```

If the auth token changes, the entire chain updates automatically.

### Key Riverpod Concepts for This Project

| Concept | What It Does | Used For |
|---|---|---|
| `Provider` | Holds a value that does not change on its own | Configuration, constants |
| `FutureProvider` | Holds a value that comes from an async operation (like a network call) | One-time data fetches |
| `AsyncNotifier` | Holds async state that can be modified by methods | Pages, forms, workflows |
| `StreamProvider` | Holds a value from a continuous data stream | Drift database watching |
| `family` | Creates a separate instance of a provider for each unique parameter | Per-page, per-schema providers |
| `keepAlive` | Prevents auto-disposal | Shared caches |
| `invalidateSelf` | Forces a provider to refetch its data | Background refresh |

---

## 2.3 Drift (Local Database)

### What Is Drift?

Drift is a library that lets Flutter apps use SQLite — a small, fast, file-based database — with type-safe Dart code. Instead of writing raw SQL strings, you write Dart code that gets compiled into SQL.

### Why a Database? Why Not Just Memory?

Thesa UI is "offline-first," which means it must remember things even when:

- The app is closed and reopened
- The device loses internet connectivity
- The server is temporarily unavailable

In-memory state disappears when the app closes. A database persists data to disk.

### Why Drift Over Hive or Isar?

| Feature | Drift (SQLite) | Hive (Key-Value) | Isar (NoSQL) |
|---|---|---|---|
| **Relational queries** | Full SQL joins, subqueries | No | Limited |
| **Reactive streams** | `watch()` on any query | `watch()` on boxes | `watch()` on queries |
| **Schema migrations** | Built-in step-by-step system | Manual | Built-in |
| **Complex invalidation** | "Delete all pages referencing schema X" — trivial SQL | Must iterate all entries | Possible but verbose |
| **Cross-platform** | Web (sql.js), desktop, mobile | Yes | No web support |
| **Type safety** | Generated DAOs with compile-time checks | Weak | Moderate |
| **Maturity** | Very mature, backed by SQLite (decades old) | Mature | Newer, less proven |

The decisive factor: Thesa UI's cache has **relational data**. A page references schemas. Schemas are shared across pages. Navigation items have parent-child relationships. Actions reference workflows. These relationships are naturally expressed in SQL joins, which Drift makes trivial. In a key-value store like Hive, you would need to manually manage these relationships.

### Key Drift Concepts for This Project

| Concept | What It Does |
|---|---|
| **Table** | Defines a database table with typed columns |
| **DAO** (Data Access Object) | Groups related database operations together |
| **`watch()`** | Returns a Stream that emits new data whenever the underlying table changes |
| **Migration** | Code that transforms the database schema when the app updates |
| **Companion** | A helper object used to insert or update rows |

### How Drift Connects to Riverpod

This is a critical architecture pattern:

1. A Riverpod provider calls `driftDatabase.navigationDao.watchNavigation()`
2. Drift returns a `Stream<NavigationTree>`
3. Riverpod wraps this stream with `StreamProvider`
4. Any widget watching this provider rebuilds automatically when the database changes
5. When a background network fetch writes new data to the database, the widget rebuilds — no manual notification needed

This creates a clean reactive pipeline: **Network → Database → Stream → Provider → Widget**.

---

## 2.4 go_router (Navigation)

### What Is go_router?

go_router is Flutter's recommended routing library. "Routing" means deciding which screen to show based on the current URL or navigation action.

### Why go_router?

- **URL-based**: Each screen has a URL path (like `/orders/list`), which is essential for web support and deep linking
- **Nested navigation**: Supports a sidebar layout where the sidebar stays visible while the content area changes
- **StatefulShellRoute**: Preserves the state of each section — if you are on page 3 of the orders table, switch to settings, then switch back, you are still on page 3
- **Programmatic route generation**: Routes can be created from data (BFF navigation descriptors) rather than hardcoded

### Key go_router Concepts for This Project

| Concept | What It Does |
|---|---|
| `GoRouter` | The central router that maps URLs to screens |
| `GoRoute` | A single route definition (path → widget) |
| `ShellRoute` | A route that wraps child routes in a persistent layout (like a sidebar) |
| `StatefulShellRoute` | Like ShellRoute, but preserves state of each child's navigator independently |
| `redirect` | A function that can intercept navigation (used for auth guards) |

### Dynamic Route Generation

In a normal app, routes are hardcoded in the source code. In Thesa UI, routes are generated at runtime from BFF data:

1. BFF sends a navigation tree (a list of menu items with paths)
2. Thesa UI converts each menu item into a `GoRoute`
3. `GoRouter` is configured with these generated routes
4. When the BFF changes the navigation tree, routes are regenerated

---

## 2.5 dio + retrofit (Networking)

### What Is dio?

dio is an HTTP client for Dart — it makes network requests (GET, POST, PUT, DELETE) to servers. It is like `fetch` in JavaScript or `requests` in Python.

### Why dio Over http?

The built-in `http` package is simple but lacks:

- **Interceptors**: Middleware that runs before/after every request (for auth tokens, retries, caching headers)
- **Request cancellation**: Ability to cancel in-flight requests when the user navigates away
- **Timeout configuration**: Fine-grained timeout settings per request
- **File upload/download**: Built-in progress tracking

### What Is retrofit?

retrofit generates dio request code from annotated interface definitions. Instead of manually constructing URLs and parsing responses, you define:

```
GET /ui/pages/{pageId} → returns PageDescriptor
```

And retrofit generates the networking code automatically.

### Interceptor Chain

dio uses an "interceptor chain" — a series of middleware that processes every request and response:

```
Request flows through:

  1. AuthInterceptor     → Adds "Authorization: Bearer <token>" header
  2. ETagInterceptor     → Adds "If-None-Match: <etag>" header for cache validation
  3. DeduplicationInterceptor → Prevents duplicate in-flight requests
  4. TelemetryInterceptor → Records timing for performance monitoring
  5. RetryInterceptor    → Retries failed requests with exponential backoff

Response flows back through the chain in reverse.
```

---

## 2.6 freezed + json_serializable (Data Models)

### What Are These?

- **freezed**: Generates immutable Dart classes with equality, copying, and pattern matching
- **json_serializable**: Generates JSON parsing code (fromJson/toJson)

### Why?

The BFF sends JSON responses. We need to convert this JSON into typed Dart objects. Writing this conversion code by hand is tedious and error-prone. These libraries generate it automatically.

"Immutable" means once created, an object cannot be changed. Instead, you create a new copy with the changes you want. This prevents a class of bugs where one part of the app unexpectedly modifies data that another part is using.

---

## 2.7 Other Libraries

| Library | What It Does | Why It Is Needed |
|---|---|---|
| `flutter_secure_storage` | Stores sensitive data (auth tokens) in the OS keychain/keystore | Tokens must survive app restarts but must be encrypted |
| `connectivity_plus` | Detects whether the device has internet connectivity | The app must behave differently online vs. offline |
| `flutter_adaptive_scaffold` | Provides Material 3 adaptive layout components | Simplifies responsive sidebar/content layouts |
| `data_table_2` | A data table widget with sorting, fixed columns, and virtualization | The built-in DataTable does not scale to large datasets |
| `reactive_forms` | Form management with validation, field grouping, and reactivity | The dynamic form engine needs a robust form controller |
| `logging` | Structured logging | Telemetry and debugging |

---

## 2.8 Library Dependency Map

This shows how the libraries relate to each other:

```
┌──────────────────────────────────────────────────┐
│                Flutter Framework                  │
├──────────────────────────────────────────────────┤
│                                                    │
│  ┌──────────────┐    ┌──────────┐    ┌─────────┐ │
│  │  Riverpod    │    │ go_router│    │ freezed  │ │
│  │  (state)     │    │ (routes) │    │ (models) │ │
│  └──────┬───────┘    └────┬─────┘    └────┬─────┘ │
│         │                 │               │        │
│         │    ┌────────────┘               │        │
│         │    │                            │        │
│  ┌──────▼────▼───┐              ┌────────▼──────┐ │
│  │    Drift      │              │ json_serial.  │ │
│  │  (database)   │              │ (JSON parse)  │ │
│  └──────┬────────┘              └───────────────┘ │
│         │                                          │
│  ┌──────▼────────┐    ┌─────────────────────────┐ │
│  │   SQLite      │    │   dio + retrofit        │ │
│  │  (storage)    │    │   (networking)           │ │
│  └───────────────┘    └─────────────────────────┘ │
│                                                    │
│  ┌──────────────────┐  ┌────────────────────────┐ │
│  │ secure_storage   │  │  connectivity_plus     │ │
│  │ (tokens)         │  │  (network detection)   │ │
│  └──────────────────┘  └────────────────────────┘ │
└──────────────────────────────────────────────────┘
```
