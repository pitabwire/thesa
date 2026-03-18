/// OpenTelemetry exporter for OTLP format.
library;

import 'dart:io';

import 'package:dio/dio.dart';
import 'package:logger/logger.dart';
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
  final Logger _logger = Logger();

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
        _logger.debug('OpenTelemetry collector unreachable - buffering events');
      } else {
        _logger.warning(
          'Failed to export telemetry: ${error.message}',
        );
      }
    } catch (error, stack) {
      _logger.error(
        'Unexpected error exporting telemetry',
        error: error,
        stackTrace: stack,
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
    final timestamp = event.when(
      pageRender: (event) => event.timestamp,
      apiRequest: (event) => event.timestamp,
      workflowTransition: (event) => event.timestamp,
      uiError: (event) => event.timestamp,
      renderFailure: (event) => event.timestamp,
      cacheHit: (event) => event.timestamp,
      cacheMiss: (event) => event.timestamp,
      authRefresh: (event) => event.timestamp,
      frameTiming: (event) => event.timestamp,
      actionExecution: (event) => event.timestamp,
      formSubmission: (event) => event.timestamp,
      tableInteraction: (event) => event.timestamp,
    );

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
    return event.when(
      pageRender: (event) => [
        _stringAttribute('page.id', event.pageId),
        _intAttribute('page.render_time_ms', event.renderTimeMs),
        _intAttribute('page.component_count', event.componentCount),
        _boolAttribute('cache.from_cache', event.fromCache),
        if (event.cacheAgeMs != null)
          _intAttribute('cache.age_ms', event.cacheAgeMs!),
        _boolAttribute('cache.stale', event.stale),
      ],
      apiRequest: (event) => [
        _stringAttribute('http.endpoint', event.endpoint),
        _stringAttribute('http.method', event.method),
        _intAttribute('http.duration_ms', event.durationMs),
        _intAttribute('http.status_code', event.statusCode),
        _boolAttribute('http.cached', event.cached),
        _boolAttribute('http.etag_hit', event.etagHit),
        _intAttribute('http.retry_count', event.retryCount),
      ],
      workflowTransition: (event) => [
        _stringAttribute('workflow.id', event.workflowId),
        _stringAttribute('workflow.from_step', event.fromStep),
        _stringAttribute('workflow.to_step', event.toStep),
        _intAttribute('workflow.duration_ms', event.durationMs),
      ],
      uiError: (event) => [
        _stringAttribute('error.type', event.errorType),
        _stringAttribute('component.type', event.componentType),
        _stringAttribute('component.id', event.componentId),
        _stringAttribute('page.id', event.pageId),
        _stringAttribute('error.message', event.errorMessage),
        if (event.stackTrace != null)
          _stringAttribute('error.stack_trace', event.stackTrace!),
      ],
      renderFailure: (event) => [
        _stringAttribute('component.id', event.componentId),
        _stringAttribute('descriptor.type', event.descriptorType),
        _stringAttribute('error.message', event.errorMessage),
        _stringAttribute('page.id', event.pageId),
        if (event.stackTrace != null)
          _stringAttribute('error.stack_trace', event.stackTrace!),
      ],
      cacheHit: (event) => [
        _stringAttribute('cache.type', event.cacheType),
        _stringAttribute('cache.key', event.key),
        _intAttribute('cache.age_ms', event.ageMs),
        _boolAttribute('cache.stale', event.stale),
      ],
      cacheMiss: (event) => [
        _stringAttribute('cache.type', event.cacheType),
        _stringAttribute('cache.key', event.key),
      ],
      authRefresh: (event) => [
        _boolAttribute('auth.success', event.success),
        _intAttribute('auth.duration_ms', event.durationMs),
        _stringAttribute('auth.triggered_by', event.triggeredBy),
        if (event.errorMessage != null)
          _stringAttribute('error.message', event.errorMessage!),
      ],
      frameTiming: (event) => [
        _stringAttribute('page.id', event.pageId),
        _doubleAttribute('frame.time_ms', event.frameTimeMs),
        _boolAttribute('frame.is_jank', event.isJank),
        _intAttribute('frame.widget_build_count', event.widgetBuildCount),
      ],
      actionExecution: (event) => [
        _stringAttribute('action.id', event.actionId),
        _stringAttribute('action.type', event.actionType),
        _stringAttribute('page.id', event.pageId),
        _boolAttribute('action.success', event.success),
        _intAttribute('action.duration_ms', event.durationMs),
        if (event.errorMessage != null)
          _stringAttribute('error.message', event.errorMessage!),
      ],
      formSubmission: (event) => [
        _stringAttribute('form.schema_id', event.schemaId),
        _stringAttribute('page.id', event.pageId),
        _boolAttribute('form.success', event.success),
        _intAttribute('form.duration_ms', event.durationMs),
        _intAttribute('form.field_count', event.fieldCount),
        if (event.errorMessage != null)
          _stringAttribute('error.message', event.errorMessage!),
      ],
      tableInteraction: (event) => [
        _stringAttribute('table.id', event.tableId),
        _stringAttribute('table.interaction_type', event.interactionType),
        _stringAttribute('page.id', event.pageId),
        if (event.rowCount != null)
          _intAttribute('table.row_count', event.rowCount!),
      ],
    );
  }

  /// Check if event should include status
  bool _shouldIncludeStatus(TelemetryEvent event) {
    return event.maybeWhen(
      uiError: (_) => true,
      renderFailure: (_) => true,
      authRefresh: (event) => !event.success,
      actionExecution: (event) => !event.success,
      formSubmission: (event) => !event.success,
      orElse: () => false,
    );
  }

  /// Build span status (for errors)
  Map<String, dynamic> _buildSpanStatus(TelemetryEvent event) {
    return {
      'code': 'STATUS_CODE_ERROR',
      'message': event.when(
        uiError: (event) => event.errorMessage,
        renderFailure: (event) => event.errorMessage,
        authRefresh: (event) => event.errorMessage ?? 'Auth refresh failed',
        actionExecution: (event) =>
            event.errorMessage ?? 'Action execution failed',
        formSubmission: (event) =>
            event.errorMessage ?? 'Form submission failed',
        pageRender: (_) => '',
        apiRequest: (_) => '',
        workflowTransition: (_) => '',
        cacheHit: (_) => '',
        cacheMiss: (_) => '',
        frameTiming: (_) => '',
        tableInteraction: (_) => '',
      ),
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
