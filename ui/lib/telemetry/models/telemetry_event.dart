/// Telemetry event models.
library;

import 'package:freezed_annotation/freezed_annotation.dart';

part 'telemetry_event.freezed.dart';
part 'telemetry_event.g.dart';

/// Base telemetry event interface
///
/// All telemetry events must implement this sealed class
@freezed
sealed class TelemetryEvent with _$TelemetryEvent {
  /// Page render timing event
  const factory TelemetryEvent.pageRender({
    required String pageId,
    required int renderTimeMs,
    required int componentCount,
    required bool fromCache,
    int? cacheAgeMs,
    required bool stale,
    required DateTime timestamp,
  }) = PageRenderEvent;

  /// API request timing event
  const factory TelemetryEvent.apiRequest({
    required String endpoint,
    required String method,
    required int durationMs,
    required int statusCode,
    required bool cached,
    required bool etagHit,
    required int retryCount,
    required DateTime timestamp,
  }) = ApiRequestEvent;

  /// Workflow transition event
  const factory TelemetryEvent.workflowTransition({
    required String workflowId,
    required String fromStep,
    required String toStep,
    required int durationMs,
    required DateTime timestamp,
  }) = WorkflowTransitionEvent;

  /// UI error event
  const factory TelemetryEvent.uiError({
    required String errorType,
    required String componentType,
    required String componentId,
    required String pageId,
    required String errorMessage,
    String? stackTrace,
    required DateTime timestamp,
  }) = UiErrorEvent;

  /// Render failure event
  const factory TelemetryEvent.renderFailure({
    required String componentId,
    required String descriptorType,
    required String errorMessage,
    required String pageId,
    String? stackTrace,
    required DateTime timestamp,
  }) = RenderFailureEvent;

  /// Cache hit event
  const factory TelemetryEvent.cacheHit({
    required String cacheType,
    required String key,
    required int ageMs,
    required bool stale,
    required DateTime timestamp,
  }) = CacheHitEvent;

  /// Cache miss event
  const factory TelemetryEvent.cacheMiss({
    required String cacheType,
    required String key,
    required DateTime timestamp,
  }) = CacheMissEvent;

  /// Authentication refresh event
  const factory TelemetryEvent.authRefresh({
    required bool success,
    required int durationMs,
    required String triggeredBy,
    String? errorMessage,
    required DateTime timestamp,
  }) = AuthRefreshEvent;

  /// Frame timing event (performance)
  const factory TelemetryEvent.frameTiming({
    required String pageId,
    required double frameTimeMs,
    required bool isJank,
    required int widgetBuildCount,
    required DateTime timestamp,
  }) = FrameTimingEvent;

  /// Action execution event
  const factory TelemetryEvent.actionExecution({
    required String actionId,
    required String actionType,
    required String pageId,
    required bool success,
    required int durationMs,
    String? errorMessage,
    required DateTime timestamp,
  }) = ActionExecutionEvent;

  /// Form submission event
  const factory TelemetryEvent.formSubmission({
    required String schemaId,
    required String pageId,
    required bool success,
    required int durationMs,
    required int fieldCount,
    String? errorMessage,
    required DateTime timestamp,
  }) = FormSubmissionEvent;

  /// Table interaction event
  const factory TelemetryEvent.tableInteraction({
    required String tableId,
    required String interactionType, // sort, filter, paginate, bulk_action
    required String pageId,
    int? rowCount,
    required DateTime timestamp,
  }) = TableInteractionEvent;

  factory TelemetryEvent.fromJson(Map<String, dynamic> json) =>
      _$TelemetryEventFromJson(json);
}

/// Event type enum for categorization
enum EventType {
  pageRender,
  apiRequest,
  workflowTransition,
  uiError,
  renderFailure,
  cacheHit,
  cacheMiss,
  authRefresh,
  frameTiming,
  actionExecution,
  formSubmission,
  tableInteraction,
}

/// Extension to get event type from event
extension TelemetryEventType on TelemetryEvent {
  EventType get eventType => when(
        pageRender: (_) => EventType.pageRender,
        apiRequest: (_) => EventType.apiRequest,
        workflowTransition: (_) => EventType.workflowTransition,
        uiError: (_) => EventType.uiError,
        renderFailure: (_) => EventType.renderFailure,
        cacheHit: (_) => EventType.cacheHit,
        cacheMiss: (_) => EventType.cacheMiss,
        authRefresh: (_) => EventType.authRefresh,
        frameTiming: (_) => EventType.frameTiming,
        actionExecution: (_) => EventType.actionExecution,
        formSubmission: (_) => EventType.formSubmission,
        tableInteraction: (_) => EventType.tableInteraction,
      );

  String get eventName => when(
        pageRender: (_) => 'page.render',
        apiRequest: (_) => 'api.request',
        workflowTransition: (_) => 'workflow.transition',
        uiError: (_) => 'ui.error',
        renderFailure: (_) => 'render.failure',
        cacheHit: (_) => 'cache.hit',
        cacheMiss: (_) => 'cache.miss',
        authRefresh: (_) => 'auth.refresh',
        frameTiming: (_) => 'frame.timing',
        actionExecution: (_) => 'action.execution',
        formSubmission: (_) => 'form.submission',
        tableInteraction: (_) => 'table.interaction',
      );
}
