# 15. Error Handling and User Experience

## Why Error Handling Matters

In enterprise applications, errors are inevitable:
- Networks drop
- Servers go down
- Users lose permissions
- Data becomes stale
- Components have bugs

How the app handles errors determines whether users trust it. A good error handling strategy means:
- Users understand what went wrong
- Users know what to do about it
- The rest of the app keeps working even when one part fails
- The app recovers automatically when possible

---

## Error Categories

Thesa UI categorizes all errors into distinct types. Each type has a specific UX treatment:

### 1. Network Errors

**What causes them**: Wi-Fi drops, server unreachable, DNS failure, VPN disconnect, request timeout.

**UX treatment**:

```
┌────────────────────────────────────────────────────────┐
│ ⚠ Unable to connect to the server                      │
│                                                         │
│ Showing cached data from 15 minutes ago.               │
│                                                         │
│ [Retry Now]                                             │
└────────────────────────────────────────────────────────┘
```

**Behavior**:
- If cached data exists → show cached data with a stale warning banner
- If no cached data → show this error with a retry button
- Automatic retry after a delay (exponential backoff: 1s, 2s, 4s, 8s)
- When connection returns → automatically refresh (no user action needed)

### 2. Authentication Errors (401)

**What causes them**: Token expired, token revoked, session ended by admin, password changed.

**UX treatment**:
- Silent token refresh (user does not see anything if refresh succeeds)
- If refresh fails: redirect to login with message

```
┌────────────────────────────────────────────────────────┐
│ Your session has expired. Please sign in again.         │
│                                                         │
│ [Sign In]                                               │
└────────────────────────────────────────────────────────┘
```

**Behavior**:
- Never show a "401 error" to users — translate it to human language
- Preserve the URL the user was on so they return to it after login

### 3. Permission Errors (403)

**What causes them**: User tries to access a page or perform an action they are not allowed to. This should be rare because the UI already hides disallowed elements, but it can happen if:
- Permissions changed after the page loaded
- A stale cache showed an action that is no longer allowed
- A deep link points to a restricted page

**UX treatment**:

```
┌────────────────────────────────────────────────────────┐
│ 🔒 Access Denied                                       │
│                                                         │
│ You do not have permission to view this page.           │
│ If you believe this is an error, contact your           │
│ administrator.                                          │
│                                                         │
│ [Go to Dashboard]                                       │
└────────────────────────────────────────────────────────┘
```

**Behavior**:
- Full-page error for page-level 403
- Inline error for action-level 403: "You do not have permission to perform this action."
- Trigger a permission refresh to update the local cache

### 4. Not Found Errors (404)

**What causes them**: A page or resource was deleted, a stale link, a typo in a URL.

**UX treatment**:

```
┌────────────────────────────────────────────────────────┐
│ 📭 Page Not Found                                      │
│                                                         │
│ The page you're looking for doesn't exist or has        │
│ been removed.                                           │
│                                                         │
│ [Go to Dashboard]  [Go Back]                            │
└────────────────────────────────────────────────────────┘
```

### 5. Validation Errors (422)

**What causes them**: Form data that does not pass server-side validation.

**UX treatment**: Inline field errors (see Form Engine, Section 8):

```
Email *
┌──────────────────────────────────────┐
│ alice@                               │
└──────────────────────────────────────┘
⚠ Please enter a valid email address

Quantity *
┌──────────────────────────────────────┐
│ 0                                    │
└──────────────────────────────────────┘
⚠ Quantity must be at least 1
```

**Behavior**:
- Scroll the first errored field into view
- Focus the first errored field
- Show all field errors simultaneously (not just the first one)
- If the error is not field-specific, show it as a banner above the form

### 6. Server Errors (500)

**What causes them**: Backend bug, database failure, third-party service down.

**UX treatment**:

```
┌────────────────────────────────────────────────────────┐
│ ⚠ Something went wrong                                 │
│                                                         │
│ The server encountered an unexpected error.             │
│ This has been reported automatically.                   │
│                                                         │
│ [Retry]  [Report Issue]                                 │
└────────────────────────────────────────────────────────┘
```

**Behavior**:
- Log the error details to the telemetry system (see Section 16)
- Show a user-friendly message (never show stack traces or technical details to users)
- Offer a retry button
- If the error is on a specific component (not the whole page), only that component shows the error

### 7. Stale Cache Warnings

**What causes them**: The user is seeing cached data that is past its TTL and the background refresh has not completed yet (or failed).

**UX treatment**:

```
┌────────────────────────────────────────────────────────┐
│ ℹ Showing cached data · Last updated 15 min ago  [↻]  │
└────────────────────────────────────────────────────────┘
```

A subtle, non-blocking banner at the top of the page. It does NOT prevent the user from working. Clicking the refresh icon forces an immediate refresh.

**Behavior**:
- Appears when data is stale AND a background refresh is pending or failed
- Disappears automatically when fresh data arrives
- Uses a distinct (info-level) color, not an error color — stale data is not an error

### 8. Component Rendering Errors

**What causes them**: A bug in a component, a malformed BFF descriptor, a missing schema, an unsupported field type.

**UX treatment**: Only the failing component shows an error. The rest of the page renders normally:

```
┌─────────────────────────────────────────────────────────┐
│ Orders                                    [+ New Order]  │
├─────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────┐ │
│ │ ⚠ Failed to render "order-filters"                  │ │
│ │ Error: Component "advanced_filter" is not supported │ │
│ │ [Retry] [Hide]                                      │ │
│ └─────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────┤
│ (The rest of the page — table, search bar — works fine) │
│ ☐ │ ORD-1234 │ Alice Smith │ $142.50 │ ● Pending │     │
│ ☐ │ ORD-1235 │ Bob Jones   │ $89.00  │ ● Shipped │     │
└─────────────────────────────────────────────────────────┘
```

**Behavior**:
- ErrorBoundary widget catches the error
- Renders an error card in place of the failing component
- Logs the error to the telemetry system
- The rest of the page is unaffected

This is the single most important error handling feature. Without it, one bad component descriptor from the BFF would crash the entire page.

---

## The ErrorBoundary Widget

### What It Is

An `ErrorBoundary` is a wrapper widget that catches errors thrown by its child widget during rendering. It is analogous to a `try/catch` block, but for the widget tree.

### How It Works

```
ErrorBoundary(
  child: DynamicFilterPanel(descriptor: filterDescriptor),
  fallback: (error, stackTrace, retry) => ErrorCard(
    message: "Failed to render filters",
    error: error,
    onRetry: retry,
  ),
)
```

1. Thesa UI wraps every dynamically rendered component in an `ErrorBoundary`
2. If the child throws during build → `ErrorBoundary` catches it and renders the fallback
3. The fallback is an `ErrorCard` with the error message and a retry button
4. Clicking retry rebuilds the child (which triggers the provider to refetch)

### Why Every Component Is Wrapped

In a dynamic UI, you cannot predict which components will fail. A BFF update might introduce a new component type that the client does not support. A schema might reference a non-existent sub-schema. A chart widget might receive data in an unexpected format.

By wrapping every component, these failures are isolated. The principle is: **no single component should be able to bring down the page**.

---

## Retry Strategy

### Automatic Retries (Network Layer)

The retry interceptor in dio automatically retries failed requests:

```
Attempt 1: Request fails (network error)
  Wait 1 second
Attempt 2: Request fails (network error)
  Wait 2 seconds
Attempt 3: Request fails (network error)
  Wait 4 seconds
Attempt 4: Give up. Return error to the caller.
```

This is called "exponential backoff" — each retry waits twice as long as the previous one. This prevents overwhelming a struggling server with rapid retries.

**Which requests are retried**:
- Network errors (timeout, connection reset, DNS failure) → always retry
- 500 / 502 / 503 responses → retry (server might recover)
- 400 / 401 / 403 / 404 / 422 → do NOT retry (these are not transient errors)

### Manual Retries (UI Level)

Error cards and error pages show a "Retry" button. Clicking it:
1. Invalidates the relevant Riverpod provider (`ref.invalidate(pageProvider(pageId))`)
2. The provider refetches from the cache coordinator
3. The cache coordinator attempts a new network request
4. If successful → the widget rebuilds with fresh data
5. If still failing → the error is shown again

---

## Error Logging and Reporting

### What Gets Logged

Every error is logged to the telemetry system (see Section 16) with structured data:

```json
{
  "event": "ui.error",
  "timestamp": "2026-02-06T10:15:30Z",
  "error_type": "component_render_failure",
  "component_type": "filter_panel",
  "component_id": "order-filters",
  "page_id": "orders-list",
  "error_message": "Schema 'filter-schema' not found",
  "stack_trace": "...",
  "user_id": "user-abc123",
  "session_id": "session-xyz"
}
```

### What the User Sees

Users **never** see raw error messages, stack traces, or technical jargon. All errors are translated to human-friendly messages:

| Technical Error | User Message |
|---|---|
| `SocketException: Connection refused` | "Unable to connect to the server" |
| `FormatException: Invalid JSON` | "Received invalid data from the server" |
| `RangeError: index out of bounds` | "Something went wrong displaying this component" |
| `HTTP 500 Internal Server Error` | "The server encountered an unexpected error" |
| `HTTP 429 Too Many Requests` | "Too many requests. Please wait a moment and try again" |

---

## Graceful Degradation

### The Principle

When something fails, the app should degrade gracefully — showing as much as possible rather than nothing at all.

### Examples

| Failure | Graceful Degradation |
|---|---|
| Navigation fetch fails | Show cached navigation (sidebar still works) |
| One component on a page fails | Show error placeholder for that component; render all others |
| Chart library fails to load | Show the data in a simple table instead |
| Permission fetch fails | Use cached permissions; show stale warning |
| Schema not found for a form | Show raw field names instead of labels |
| Background refresh fails | Keep showing cached data; retry later |
| Image fails to load | Show a placeholder icon |
| Entire page fetch fails | Show cached page if available; show error page if not |

### Partial Rendering

This is a key concept. A page with 5 components where 1 fails should still render the other 4:

```
┌─────────────────────────────────────────────────────────┐
│ Dashboard                                                │
├─────────────────────────────────────────────────────────┤
│ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐    │
│ │ Orders   │ │ Revenue  │ │ ⚠ Error  │ │ Returns  │    │
│ │  1,247   │ │ $89,432  │ │ [Retry]  │ │    23    │    │
│ └──────────┘ └──────────┘ └──────────┘ └──────────┘    │  ← 3 out of 4 metrics render fine
├─────────────────────────────────────────────────────────┤
│ Revenue Trend                                            │
│ [chart renders normally]                                 │  ← chart is unaffected
├─────────────────────────────────────────────────────────┤
│ Recent Activity                                          │
│ [table renders normally]                                 │  ← table is unaffected
└─────────────────────────────────────────────────────────┘
```

The user can still work with the dashboard. The one failing metric widget does not prevent them from seeing revenue trends or recent activity.
