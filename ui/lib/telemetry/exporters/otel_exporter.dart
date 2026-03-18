/// OpenTelemetry exporter for OTLP format.
library;

import 'dart:io';

import 'package:dio/dio.dart';
import 'package:logging/logging.dart';
import 'package:package_info_plus/package_info_plus.dart';

import '../models/telemetry_event.dart';
import 'telemetry_exporter.dart';

/// OpenTelemetry exporter that sends events in OTLP format
///
/// Converts telemetry events to OpenTelemetry Protocol (OTLP) format
/// and sends them to an OpenTelemetry collector endpoint.
class OtelExporter implements TelemetryExporter {
  OtelExporter({
    required this.endpoint,
    required this.serviceName,
    Dio? dio,
  }) : _dio = dio ?? Dio();

  /// OpenTelemetry collector endpoint (e.g., http://localhost:4318/v1/traces)
  final String endpoint;

  /// Service name for resource attributes
  final String serviceName;

  final Dio _dio;
  final Logger _logger = Logger('OtelExporter');

  /// Cached package info for resource attributes
  PackageInfo? _packageInfo;

  @override
  Future<void> export(List<TelemetryEvent> events) async {
    if (events.isEmpty) {
      return;
    }

    try {
      // Initialize package info if needed
      _packageInfo ??= await PackageInfo.fromPlatform();

      // Convert events to OTLP format
      final otlpPayload = await _convertToOtlp(events);

      // Send to collector
      await _dio.post(
        endpoint,
        data: otlpPayload,
        options: Options(
          headers: {
            'Content-Type': 'application/json',
          },
          // Don't retry on failure to avoid blocking
          extra: {'retry': false},
        ),
      );

      _logger.info('Exported ${events.length} events to OpenTelemetry');
    } on DioException catch (error) {
      // Log but don't throw - we don't want telemetry failures to break the app
      if (error.type == DioExceptionType.connectionError ||
          error.type == DioExceptionType.connectionTimeout) {
        _logger.fine('OpenTelemetry collector unreachable - buffering events');
      } else {
        _logger.warning(
          'Failed to export telemetry: ${error.message}',
        );
      }
    } catch (error, stack) {
      _logger.severe(
        'Unexpected error exporting telemetry',
        error,
        stack,
      );
    }
  }

  /// Convert events to OpenTelemetry OTLP format
  Future<Map<String, dynamic>> _convertToOtlp(
    List<TelemetryEvent> events,
  ) async {
    return {
      'resourceSpans': [
        {
          'resource': {
            'attributes': await _buildResourceAttributes(),
          },
          'scopeSpans': [
            {
              'scope': {
                'name': serviceName,
                'version': _packageInfo?.version ?? '1.0.0',
              },
              'spans': events.map(_eventToSpan).toList(),
            },
          ],
        },
      ],
    };
  }

  /// Build resource attributes for OTLP
  Future<List<Map<String, dynamic>>> _buildResourceAttributes() async {
    final attributes = <Map<String, dynamic>>[
      _stringAttribute('service.name', serviceName),
      _stringAttribute('service.version', _packageInfo?.version ?? '1.0.0'),
      _stringAttribute(
        'deployment.environment',
        const String.fromEnvironment('ENV', defaultValue: 'development'),
      ),
    ];

    // Add platform-specific attributes
    if (Platform.isAndroid) {
      attributes.add(_stringAttribute('device.type', 'android'));
      attributes.add(_stringAttribute('os.type', 'android'));
    } else if (Platform.isIOS) {
      attributes.add(_stringAttribute('device.type', 'ios'));
      attributes.add(_stringAttribute('os.type', 'ios'));
    } else if (Platform.isLinux) {
      attributes.add(_stringAttribute('device.type', 'desktop'));
      attributes.add(_stringAttribute('os.type', 'linux'));
    } else if (Platform.isMacOS) {
      attributes.add(_stringAttribute('device.type', 'desktop'));
      attributes.add(_stringAttribute('os.type', 'macos'));
    } else if (Platform.isWindows) {
      attributes.add(_stringAttribute('device.type', 'desktop'));
      attributes.add(_stringAttribute('os.type', 'windows'));
    } else {
      attributes.add(_stringAttribute('device.type', 'web'));
      attributes.add(_stringAttribute('os.type', 'web'));
    }

    return attributes;
  }

  /// Convert a telemetry event to an OpenTelemetry span
  Map<String, dynamic> _eventToSpan(TelemetryEvent event) {
    final timestamp = switch (event) {
      PageRenderEvent(:final timestamp) => timestamp,
      ApiRequestEvent(:final timestamp) => timestamp,
      WorkflowTransitionEvent(:final timestamp) => timestamp,
      UiErrorEvent(:final timestamp) => timestamp,
      RenderFailureEvent(:final timestamp) => timestamp,
      CacheHitEvent(:final timestamp) => timestamp,
      CacheMissEvent(:final timestamp) => timestamp,
      AuthRefreshEvent(:final timestamp) => timestamp,
      FrameTimingEvent(:final timestamp) => timestamp,
      ActionExecutionEvent(:final timestamp) => timestamp,
      FormSubmissionEvent(:final timestamp) => timestamp,
      TableInteractionEvent(:final timestamp) => timestamp,
    };

    final startTimeNano = timestamp.microsecondsSinceEpoch * 1000;

    return {
      'name': event.eventName,
      'kind': 'SPAN_KIND_INTERNAL',
      'startTimeUnixNano': startTimeNano.toString(),
      'endTimeUnixNano': startTimeNano.toString(), // Point in time event
      'attributes': _buildSpanAttributes(event),
      if (_shouldIncludeStatus(event)) 'status': _buildSpanStatus(event),
    };
  }

  /// Build span attributes from event
  List<Map<String, dynamic>> _buildSpanAttributes(TelemetryEvent event) {
    return switch (event) {
      PageRenderEvent e => [
          _stringAttribute('page.id', e.pageId),
          _intAttribute('page.render_time_ms', e.renderTimeMs),
          _intAttribute('page.component_count', e.componentCount),
          _boolAttribute('cache.from_cache', e.fromCache),
          if (e.cacheAgeMs != null)
            _intAttribute('cache.age_ms', e.cacheAgeMs!),
          _boolAttribute('cache.stale', e.stale),
        ],
      ApiRequestEvent e => [
          _stringAttribute('http.endpoint', e.endpoint),
          _stringAttribute('http.method', e.method),
          _intAttribute('http.duration_ms', e.durationMs),
          _intAttribute('http.status_code', e.statusCode),
          _boolAttribute('http.cached', e.cached),
          _boolAttribute('http.etag_hit', e.etagHit),
          _intAttribute('http.retry_count', e.retryCount),
        ],
      WorkflowTransitionEvent e => [
          _stringAttribute('workflow.id', e.workflowId),
          _stringAttribute('workflow.from_step', e.fromStep),
          _stringAttribute('workflow.to_step', e.toStep),
          _intAttribute('workflow.duration_ms', e.durationMs),
        ],
      UiErrorEvent e => [
          _stringAttribute('error.type', e.errorType),
          _stringAttribute('component.type', e.componentType),
          _stringAttribute('component.id', e.componentId),
          _stringAttribute('page.id', e.pageId),
          _stringAttribute('error.message', e.errorMessage),
          if (e.stackTrace != null)
            _stringAttribute('error.stack_trace', e.stackTrace!),
        ],
      RenderFailureEvent e => [
          _stringAttribute('component.id', e.componentId),
          _stringAttribute('descriptor.type', e.descriptorType),
          _stringAttribute('error.message', e.errorMessage),
          _stringAttribute('page.id', e.pageId),
          if (e.stackTrace != null)
            _stringAttribute('error.stack_trace', e.stackTrace!),
        ],
      CacheHitEvent e => [
          _stringAttribute('cache.type', e.cacheType),
          _stringAttribute('cache.key', e.key),
          _intAttribute('cache.age_ms', e.ageMs),
          _boolAttribute('cache.stale', e.stale),
        ],
      CacheMissEvent e => [
          _stringAttribute('cache.type', e.cacheType),
          _stringAttribute('cache.key', e.key),
        ],
      AuthRefreshEvent e => [
          _boolAttribute('auth.success', e.success),
          _intAttribute('auth.duration_ms', e.durationMs),
          _stringAttribute('auth.triggered_by', e.triggeredBy),
          if (e.errorMessage != null)
            _stringAttribute('error.message', e.errorMessage!),
        ],
      FrameTimingEvent e => [
          _stringAttribute('page.id', e.pageId),
          _doubleAttribute('frame.time_ms', e.frameTimeMs),
          _boolAttribute('frame.is_jank', e.isJank),
          _intAttribute('frame.widget_build_count', e.widgetBuildCount),
        ],
      ActionExecutionEvent e => [
          _stringAttribute('action.id', e.actionId),
          _stringAttribute('action.type', e.actionType),
          _stringAttribute('page.id', e.pageId),
          _boolAttribute('action.success', e.success),
          _intAttribute('action.duration_ms', e.durationMs),
          if (e.errorMessage != null)
            _stringAttribute('error.message', e.errorMessage!),
        ],
      FormSubmissionEvent e => [
          _stringAttribute('form.schema_id', e.schemaId),
          _stringAttribute('page.id', e.pageId),
          _boolAttribute('form.success', e.success),
          _intAttribute('form.duration_ms', e.durationMs),
          _intAttribute('form.field_count', e.fieldCount),
          if (e.errorMessage != null)
            _stringAttribute('error.message', e.errorMessage!),
        ],
      TableInteractionEvent e => [
          _stringAttribute('table.id', e.tableId),
          _stringAttribute('table.interaction_type', e.interactionType),
          _stringAttribute('page.id', e.pageId),
          if (e.rowCount != null) _intAttribute('table.row_count', e.rowCount!),
        ],
    };
  }

  /// Check if event should include status
  bool _shouldIncludeStatus(TelemetryEvent event) {
    return switch (event) {
      UiErrorEvent() => true,
      RenderFailureEvent() => true,
      AuthRefreshEvent(:final success) => !success,
      ActionExecutionEvent(:final success) => !success,
      FormSubmissionEvent(:final success) => !success,
      _ => false,
    };
  }

  /// Build span status (for errors)
  Map<String, dynamic> _buildSpanStatus(TelemetryEvent event) {
    return {
      'code': 'STATUS_CODE_ERROR',
      'message': switch (event) {
        UiErrorEvent(:final errorMessage) => errorMessage,
        RenderFailureEvent(:final errorMessage) => errorMessage,
        AuthRefreshEvent(:final errorMessage) =>
          errorMessage ?? 'Auth refresh failed',
        ActionExecutionEvent(:final errorMessage) =>
          errorMessage ?? 'Action execution failed',
        FormSubmissionEvent(:final errorMessage) =>
          errorMessage ?? 'Form submission failed',
        _ => '',
      },
    };
  }

  /// Helper to create string attribute
  Map<String, dynamic> _stringAttribute(String key, String value) {
    return {
      'key': key,
      'value': {'stringValue': value},
    };
  }

  /// Helper to create int attribute
  Map<String, dynamic> _intAttribute(String key, int value) {
    return {
      'key': key,
      'value': {'intValue': value.toString()},
    };
  }

  /// Helper to create double attribute
  Map<String, dynamic> _doubleAttribute(String key, double value) {
    return {
      'key': key,
      'value': {'doubleValue': value},
    };
  }

  /// Helper to create bool attribute
  Map<String, dynamic> _boolAttribute(String key, bool value) {
    return {
      'key': key,
      'value': {'boolValue': value},
    };
  }
}
