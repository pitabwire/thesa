# 21. Networking Layer

## What Is the Networking Layer?

The networking layer is the part of the application that communicates with the BFF server over the internet. Every piece of data Thesa UI displays — navigation, pages, schemas, permissions — comes through this layer. Every action the user takes — submitting forms, triggering actions, advancing workflows — goes out through this layer.

Think of it as the postal service for the app: it sends letters (requests) and receives packages (responses), following a strict set of rules about how to handle them.

---

## The Single Server Rule

Thesa UI talks to **exactly one server**: the BFF. No exceptions.

```
CORRECT:
  Thesa UI ←→ BFF Server ←→ Domain Service A
                           ←→ Domain Service B
                           ←→ Domain Service C

INCORRECT:
  Thesa UI ←→ BFF Server
  Thesa UI ←→ Domain Service A   ← NEVER
  Thesa UI ←→ Domain Service B   ← NEVER
```

The BFF aggregates all backend services. The frontend does not need to know how many backend services exist or where they are located.

---

## BFF Endpoints

The BFF exposes a specific set of endpoints that the networking layer consumes:

| Method | Endpoint | Purpose | Response |
|---|---|---|---|
| `GET` | `/ui/capabilities` | What features are available globally | Capabilities object with version, feature flags |
| `GET` | `/ui/navigation` | What menu items exist and their structure | Navigation tree with items, paths, permissions |
| `GET` | `/ui/pages/{pageId}` | Full layout descriptor for a page | Page descriptor with components, layout, actions |
| `GET` | `/ui/resources/{resource}` | Data for a specific resource (table rows) | Paginated data with items, pagination metadata |
| `GET` | `/ui/schemas/{schemaId}` | Data structure definition | Schema with fields, types, validation rules |
| `POST` | `/ui/actions/{actionId}` | Execute a user-triggered action | Action result or error |
| `GET` | `/ui/workflows/{workflowId}` | Workflow definition and current state | Workflow with steps, transitions, current state |
| `POST` | `/ui/workflows/{workflowId}/step` | Submit a workflow step | Next step or completion |
| `POST` | `/auth/login` | Authenticate | Tokens |
| `POST` | `/auth/refresh` | Refresh access token | New tokens |
| `POST` | `/auth/logout` | End session | Confirmation |

The `BffClient` interface defines all of these endpoints with their parameter types and response types. The `retrofit` library generates the actual HTTP code.

---

## The Interceptor Chain

Every request and response passes through a chain of interceptors — middleware functions that add behavior to the networking process. Interceptors execute in order for requests and in reverse order for responses.

### Request Flow Through Interceptors

```
Your code calls: bffClient.getPage('orders-list')
        │
        ▼
┌─────────────────────────────────────────────────┐
│  1. AuthInterceptor (request)                    │
│     Adds: Authorization: Bearer <access_token>   │
└─────────────────────────┬───────────────────────┘
                          │
┌─────────────────────────▼───────────────────────┐
│  2. ETagInterceptor (request)                    │
│     Checks: Do I have a cached ETag for this URL?│
│     If yes, adds: If-None-Match: "etag-value"    │
└─────────────────────────┬───────────────────────┘
                          │
┌─────────────────────────▼───────────────────────┐
│  3. DeduplicationInterceptor (request)           │
│     Checks: Is there already an identical        │
│     request in flight?                           │
│     If yes: Wait for existing request's response │
│     If no: Allow this request to proceed         │
└─────────────────────────┬───────────────────────┘
                          │
┌─────────────────────────▼───────────────────────┐
│  4. TelemetryInterceptor (request)               │
│     Records: Start timestamp for this request    │
└─────────────────────────┬───────────────────────┘
                          │
┌─────────────────────────▼───────────────────────┐
│  5. RetryInterceptor (request)                   │
│     Notes: This request can be retried if it     │
│     fails with a transient error                 │
└─────────────────────────┬───────────────────────┘
                          │
                          ▼
                    HTTP Request sent to BFF
                          │
                          ▼
                    HTTP Response received
                          │
┌─────────────────────────▼───────────────────────┐
│  5. RetryInterceptor (response)                  │
│     If 500/502/503 or network error:             │
│       Wait (exponential backoff) → retry request │
│     If max retries reached: pass error through   │
│     If success: pass response through            │
└─────────────────────────┬───────────────────────┘
                          │
┌─────────────────────────▼───────────────────────┐
│  4. TelemetryInterceptor (response)              │
│     Records: End timestamp, duration, status code│
│     Emits: api.request telemetry event           │
└─────────────────────────┬───────────────────────┘
                          │
┌─────────────────────────▼───────────────────────┐
│  3. DeduplicationInterceptor (response)          │
│     If other requests were waiting for this one: │
│     Deliver the response to all waiters          │
└─────────────────────────┬───────────────────────┘
                          │
┌─────────────────────────▼───────────────────────┐
│  2. ETagInterceptor (response)                   │
│     If 304 Not Modified:                         │
│       Use cached response body (no parsing)      │
│       Update cache timestamp                     │
│     If 200 OK:                                   │
│       Store new ETag for this URL                │
│       Pass new response body through             │
└─────────────────────────┬───────────────────────┘
                          │
┌─────────────────────────▼───────────────────────┐
│  1. AuthInterceptor (response)                   │
│     If 401 Unauthorized:                         │
│       Attempt token refresh                      │
│       If refresh succeeds: retry original request│
│       If refresh fails: clear session → login    │
│     Otherwise: pass response through             │
└─────────────────────────┬───────────────────────┘
                          │
                          ▼
                Your code receives the response
```

---

## Each Interceptor Explained

### AuthInterceptor

**Purpose**: Ensure every request is authenticated and handle session expiry.

**On request**:
1. Read access token from secure storage
2. Add header: `Authorization: Bearer <token>`
3. If no token exists: skip (request will fail with 401, handled on response)

**On error (401)**:
1. Acquire a mutex lock (prevent concurrent refreshes)
2. Call `POST /auth/refresh` with the stored refresh token
3. If refresh succeeds:
   - Store new access + refresh tokens
   - Retry the original request with the new access token
   - Release the lock
4. If refresh fails:
   - Clear all tokens
   - Clear session and permission caches
   - Navigate to login screen
   - Release the lock

**Concurrent request handling**: If requests A, B, and C all get 401 at the same time:
- Request A acquires the lock and starts refreshing
- Requests B and C find the lock held → they wait
- Request A's refresh completes → new token stored → lock released
- Requests B and C are retried with the new token

### ETagInterceptor

**Purpose**: Avoid downloading unchanged data.

**On request**:
1. Look up the URL in a local ETag store (in-memory map, backed by Drift)
2. If an ETag exists: add header `If-None-Match: <etag>`

**On response**:
1. If response has `ETag` header: store it for this URL
2. If status is `304 Not Modified`:
   - Do NOT parse the response body (it is empty)
   - Construct a "response" from the cached data
   - Update the cache's `fetched_at` timestamp (resets TTL)
   - Return the cached data as if it were a fresh response

**Impact**: For data that has not changed (which is the common case during background refreshes), the server sends a tiny 304 response instead of the full payload. This saves bandwidth and processing time.

### DeduplicationInterceptor

**Purpose**: Prevent the same URL from being fetched multiple times simultaneously.

**How it works**:
1. Maintain a map of in-flight requests: `Map<String, Future<Response>>`
2. When a request starts:
   - Generate a key from the method + URL + parameters
   - Check if this key already exists in the map
   - If YES: return the existing Future (share the result)
   - If NO: add the request to the map, send it, remove from map when complete

**Example**: A page loads and its descriptor references schemas A and B. The search bar also references schema A. Without deduplication, schema A is fetched twice. With deduplication, it is fetched once, and both consumers share the response.

### TelemetryInterceptor

**Purpose**: Record performance metrics for every network request.

**On request**: Record start timestamp.

**On response**: Calculate duration. Emit a structured telemetry event:
```json
{
  "event": "api.request",
  "endpoint": "/ui/pages/orders-list",
  "method": "GET",
  "durationMs": 230,
  "statusCode": 200,
  "retryCount": 0
}
```

### RetryInterceptor

**Purpose**: Automatically retry transient failures.

**Retry conditions**:
- Network error (SocketException, TimeoutException): RETRY
- HTTP 500 (Internal Server Error): RETRY
- HTTP 502 (Bad Gateway): RETRY
- HTTP 503 (Service Unavailable): RETRY
- HTTP 429 (Too Many Requests): RETRY after `Retry-After` header delay
- HTTP 400, 401, 403, 404, 422: DO NOT RETRY (these are not transient)

**Backoff schedule**:
```
Attempt 1: Immediate
Attempt 2: Wait 1 second
Attempt 3: Wait 2 seconds
Attempt 4: Wait 4 seconds
(Give up after 4 attempts)
```

**Maximum retry count**: 3 retries (4 total attempts). Configurable per endpoint if needed.

---

## Request Cancellation

When the user navigates away from a page, any in-flight requests for that page should be cancelled. Uncancelled requests waste bandwidth and may write stale data to the cache.

**How it works**:

1. Each `pageProvider` creates a `CancelToken` when it starts fetching
2. The `CancelToken` is passed to the dio request
3. When the provider is disposed (user navigated away), the `CancelToken` is cancelled
4. dio aborts the request — no response is processed
5. The BFF may still process the request, but the client ignores the response

**Impact**: On fast page switches, the user does not see data from the previous page flickering in.

---

## Background Refresh

The `BackgroundRefreshCoordinator` periodically refreshes cached data to keep it fresh:

### How It Works

```
App starts → coordinator begins schedule:

  Every 5 minutes:
    - Refresh permissions (security-critical, short TTL)

  Every 10 minutes:
    - Refresh current page descriptor
    - Refresh schemas used by current page

  Every 15 minutes:
    - Refresh capabilities
    - Refresh navigation tree

  On connectivity change (offline → online):
    - Refresh ALL stale caches immediately
```

### Rules

1. Background refreshes use ETags to minimize bandwidth
2. If the device is offline, no refreshes are attempted
3. If the app is in the background (minimized), refresh frequency is reduced
4. Refreshes are skipped if the data is still fresh (within TTL)
5. Each refresh is a normal request through the full interceptor chain (auth, telemetry, etc.)

---

## Error Response Handling

The networking layer translates HTTP error responses into typed Dart errors:

| HTTP Status | Dart Error Type | Handling |
|---|---|---|
| 200-299 | Success | Parse body, return data |
| 304 | Cache hit | Use cached body (ETag interceptor) |
| 400 | `BadRequestError` | Show generic error |
| 401 | `UnauthorizedError` | Attempt refresh (AuthInterceptor) |
| 403 | `ForbiddenError` | Show "Access Denied" |
| 404 | `NotFoundError` | Show "Not Found" |
| 422 | `ValidationError` | Parse field errors, show on form |
| 429 | `RateLimitError` | Wait and retry (RetryInterceptor) |
| 500+ | `ServerError` | Show "Server Error" + retry button |
| Network error | `NetworkError` | Show "Cannot connect" + retry |
| Timeout | `TimeoutError` | Show "Request timed out" + retry |

These typed errors are caught by providers and translated into user-facing messages (see Section 15 for error UX details).

---

## Timeout Configuration

| Request Type | Timeout | Rationale |
|---|---|---|
| Page/schema/navigation fetches | 15 seconds | These are read operations; 15s allows for slow networks |
| Action submissions | 30 seconds | Actions may trigger server-side processing |
| Workflow step submissions | 30 seconds | Same as actions |
| Auth login | 15 seconds | Login should not take long |
| Auth refresh | 10 seconds | Quick operation |
| File uploads | 120 seconds | Large files take time |
| Background refreshes | 10 seconds | If it takes longer, it is not worth blocking the schedule |
