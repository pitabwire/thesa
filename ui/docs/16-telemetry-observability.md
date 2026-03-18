# 16. Telemetry and Observability

## What Is Telemetry?

Telemetry is the process of collecting data about how the application is performing and how users are interacting with it. Think of it like the dashboard instruments in a car — speedometer, fuel gauge, engine temperature — but for software.

Observability is the ability to understand what is happening inside the app by looking at the data it produces. If something goes wrong, telemetry data helps you figure out what, where, and why.

---

## Why Telemetry Matters for Thesa UI

Thesa UI is a dynamic application — pages, forms, and components are all generated at runtime. This means:

- You cannot predict which pages users will visit (they are BFF-defined)
- You cannot predict which components will be on those pages
- You cannot predict which workflows users will follow
- You need to know if the dynamic rendering engine is working correctly
- You need to know if the cache is effective or if users are waiting on network requests

Without telemetry, you are flying blind. With it, you can:
- Detect slow pages before users complain
- Find components that fail frequently
- Measure cache effectiveness
- Track workflow completion rates
- Identify network bottlenecks

---

## What Gets Measured

### Page Render Timing

Every time a page is displayed, the telemetry system records:

```json
{
  "event": "page.render",
  "pageId": "orders-list",
  "renderTimeMs": 145,
  "componentCount": 4,
  "fromCache": true,
  "cacheAgeMs": 300000,
  "stale": false,
  "timestamp": "2026-02-06T10:15:30.123Z"
}
```

| Field | What It Means |
|---|---|
| `pageId` | Which page was rendered |
| `renderTimeMs` | How long it took from request to visible (in milliseconds) |
| `componentCount` | How many components were on the page |
| `fromCache` | Whether the page was rendered from cached data (true) or a fresh network fetch (false) |
| `cacheAgeMs` | If from cache, how old the cached data was (in milliseconds) |
| `stale` | Whether the cached data was marked as stale |

**Why this matters**: If `renderTimeMs` is consistently above 500ms, users experience noticeable lag. If `fromCache` is usually `false`, the cache strategy needs tuning.

### API Request Timing

Every network request is instrumented:

```json
{
  "event": "api.request",
  "endpoint": "/ui/pages/orders-list",
  "method": "GET",
  "durationMs": 230,
  "statusCode": 200,
  "cached": false,
  "etagHit": false,
  "retryCount": 0,
  "timestamp": "2026-02-06T10:15:29.893Z"
}
```

| Field | What It Means |
|---|---|
| `endpoint` | Which BFF endpoint was called |
| `method` | HTTP method (GET, POST, etc.) |
| `durationMs` | Round-trip time in milliseconds |
| `statusCode` | HTTP status code (200, 304, 401, 500, etc.) |
| `cached` | Whether the response came from the ETag cache (304) |
| `etagHit` | Whether the ETag matched (304 Not Modified) |
| `retryCount` | How many retries were needed (0 means first attempt succeeded) |

**Why this matters**: Slow endpoints can be identified and optimized. High retry counts indicate reliability issues. A low ETag hit rate means the server is sending full payloads unnecessarily.

### Workflow Transitions

```json
{
  "event": "workflow.transition",
  "workflowId": "refund-process",
  "fromStep": "enter-reason",
  "toStep": "manager-approval",
  "durationMs": 45000,
  "timestamp": "2026-02-06T10:16:14.567Z"
}
```

| Field | What It Means |
|---|---|
| `workflowId` | Which workflow type |
| `fromStep` | The step the user was on |
| `toStep` | The step the user moved to |
| `durationMs` | How long the user spent on the `fromStep` |

**Why this matters**: If users spend a long time on a particular step, the form might be confusing. If many users abandon a workflow at a specific step, there might be a UX problem.

### UI Errors

```json
{
  "event": "ui.error",
  "errorType": "component_render_failure",
  "componentType": "filter_panel",
  "componentId": "order-filters",
  "pageId": "orders-list",
  "errorMessage": "Schema 'filter-schema' not found",
  "stackTrace": "...",
  "timestamp": "2026-02-06T10:15:30.456Z"
}
```

**Why this matters**: Component render failures indicate mismatches between BFF descriptors and the frontend's capabilities. Tracking these helps identify BFF changes that break the UI.

### Rendering Failures

```json
{
  "event": "render.failure",
  "componentId": "revenue-chart",
  "descriptorType": "chart",
  "errorMessage": "Chart library failed to initialize: invalid dataset format",
  "pageId": "main-dashboard",
  "timestamp": "2026-02-06T10:15:31.789Z"
}
```

### Cache Events

```json
{
  "event": "cache.hit",
  "cacheType": "page",
  "key": "orders-list",
  "ageMs": 300000,
  "stale": false
}

{
  "event": "cache.miss",
  "cacheType": "schema",
  "key": "order-summary"
}
```

**Why this matters**: Cache hit rate is a key health metric. A high hit rate means the cache is working well. A low hit rate means users are frequently waiting for network requests.

### Authentication Events

```json
{
  "event": "auth.refresh",
  "success": true,
  "durationMs": 320,
  "triggeredBy": "401_response"
}
```

---

## How Telemetry Is Collected

### The Telemetry Service

The `TelemetryService` is a class that receives events from across the application and buffers them for export.

```
Flow:
  1. Components, providers, and interceptors call: telemetry.record(event)
  2. Events are added to an in-memory buffer (a simple list)
  3. Every 30 seconds (or when buffer reaches 100 events), the buffer is flushed
  4. Flushing sends the events to the export pipeline
```

### Where Events Come From

| Source | Events Produced |
|---|---|
| `PageRenderer` | `page.render` (after building the widget tree) |
| `TelemetryInterceptor` (dio) | `api.request` (for every network request) |
| `WorkflowStateMachine` | `workflow.transition` (on every step change) |
| `ErrorBoundary` | `ui.error`, `render.failure` (when a component fails) |
| `CacheCoordinator` | `cache.hit`, `cache.miss` (on every data request) |
| `AuthInterceptor` | `auth.refresh` (on every token refresh) |
| `PerformanceMonitor` | Frame timing, jank detection (when rendering takes too long) |

### Performance Monitor

The `PerformanceMonitor` tracks rendering performance:

- **Frame timing**: Records how long each frame takes to render. Frames should take less than 16ms (for 60fps). If frames consistently take longer, the UI stutters.
- **Jank detection**: A "jank" is a frame that takes more than 32ms (missing the 60fps target). The monitor counts janks per page.
- **Widget build count**: How many widgets were built during a page render. Excessive rebuilds indicate performance issues.

---

## OpenTelemetry Export

### What Is OpenTelemetry?

OpenTelemetry (OTel) is an industry standard for collecting and exporting telemetry data. It defines:
- **Traces**: The journey of a request through the system (e.g., page load → cache check → network fetch → render)
- **Metrics**: Numerical measurements over time (e.g., average page render time, cache hit rate)
- **Logs**: Structured log events (e.g., errors, warnings, info messages)

### How Thesa UI Exports

The `OtelExporter` formats buffered events into OpenTelemetry-compatible JSON:

```json
{
  "resourceSpans": [
    {
      "resource": {
        "attributes": {
          "service.name": "thesa-ui",
          "service.version": "1.0.0",
          "deployment.environment": "production",
          "device.type": "web",
          "os.type": "linux"
        }
      },
      "scopeSpans": [
        {
          "spans": [
            {
              "name": "page.render",
              "kind": "INTERNAL",
              "startTimeUnixNano": "1738836930123000000",
              "endTimeUnixNano": "1738836930268000000",
              "attributes": {
                "page.id": "orders-list",
                "page.component_count": 4,
                "cache.from_cache": true,
                "cache.stale": false
              }
            }
          ]
        }
      ]
    }
  ]
}
```

### Export Targets

The exporter sends data to an OpenTelemetry Collector, which can then forward it to:

| Backend | What It Provides |
|---|---|
| **Jaeger** | Request tracing visualization |
| **Prometheus + Grafana** | Metrics dashboards and alerting |
| **Elastic/Kibana** | Log search and analysis |
| **Datadog** | All-in-one monitoring |
| **Custom backend** | Any system that accepts OTLP format |

The Thesa UI frontend does not need to know which backend is used. It speaks the standard OTel protocol, and the collector routes data to the appropriate destination.

### Offline Behavior

When the device is offline:
1. Events continue to be buffered in memory
2. Export attempts fail silently (no error shown to the user)
3. When connectivity returns, the buffer is flushed
4. If the buffer exceeds a size limit (e.g., 10,000 events), the oldest events are dropped

---

## Key Metrics Dashboard

With the telemetry data, operations teams can build dashboards showing:

| Metric | Formula | Healthy Target |
|---|---|---|
| **Page render P95** | 95th percentile of `page.render.renderTimeMs` | < 300ms |
| **Cache hit rate** | `cache.hit` / (`cache.hit` + `cache.miss`) | > 85% |
| **API success rate** | Requests with status 2xx / total requests | > 99% |
| **API latency P95** | 95th percentile of `api.request.durationMs` | < 500ms |
| **Token refresh success rate** | Successful refreshes / total refreshes | > 99.5% |
| **Component error rate** | `render.failure` events / total component renders | < 0.1% |
| **Workflow completion rate** | Completed workflows / started workflows | Business-dependent |
| **Workflow step duration** | Average `workflow.transition.durationMs` per step | Business-dependent |
| **Jank rate** | Jank frames / total frames | < 1% |

---

## Privacy Considerations

### What Is NOT Collected

- User passwords or credentials
- Personal data entered into forms (field values)
- Full request/response bodies
- Screenshots or screen recordings
- Precise geographic location

### What IS Collected

- Anonymous usage patterns (which pages are visited, how long)
- Performance data (render times, API latencies)
- Error data (error messages, component types, page IDs)
- Device/platform metadata (OS type, screen size category — not exact resolution)

### User Identification

Events include a `userId` (opaque identifier) for correlating events within a session. This is necessary for debugging ("user X experienced an error on page Y") but does not include personal information (name, email).

### Data Retention

Telemetry data should follow the organization's data retention policy. Typical defaults:
- Performance metrics: 30 days
- Error logs: 90 days
- Traces: 7 days
