/// Dio interceptor for API request telemetry.
library;

import 'package:dio/dio.dart';
import 'package:logger/logger.dart';

import '../../telemetry/models/telemetry_event.dart';
import '../../telemetry/telemetry_service.dart';

/// Dio interceptor that records telemetry events for API requests
///
/// Tracks request timing, status codes, cache hits, and retry counts.
class TelemetryInterceptor extends Interceptor {
  TelemetryInterceptor({
    required this.telemetryService,
  });

  final TelemetryService telemetryService;
  final Logger _logger = Logger();

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    // Store request start time
    options.extra['telemetry_start_time'] = DateTime.now();
    options.extra['telemetry_retry_count'] = options.extra['retry_count'] ?? 0;
    super.onRequest(options, handler);
  }

  @override
  void onResponse(Response response, ResponseInterceptorHandler handler) {
    _recordApiEvent(
      endpoint: response.requestOptions.uri.path,
      method: response.requestOptions.method,
      statusCode: response.statusCode ?? 0,
      startTime: response.requestOptions.extra['telemetry_start_time'],
      retryCount: response.requestOptions.extra['telemetry_retry_count'] ?? 0,
      etagHit: response.statusCode == 304,
      cached: response.statusCode == 304,
    );
    super.onResponse(response, handler);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    _recordApiEvent(
      endpoint: err.requestOptions.uri.path,
      method: err.requestOptions.method,
      statusCode: err.response?.statusCode ?? 0,
      startTime: err.requestOptions.extra['telemetry_start_time'],
      retryCount: err.requestOptions.extra['telemetry_retry_count'] ?? 0,
      etagHit: false,
      cached: false,
    );
    super.onError(err, handler);
  }

  void _recordApiEvent({
    required String endpoint,
    required String method,
    required int statusCode,
    required dynamic startTime,
    required int retryCount,
    required bool etagHit,
    required bool cached,
  }) {
    if (startTime is! DateTime) {
      _logger.warning('Missing telemetry start time for $method $endpoint');
      return;
    }

    final durationMs = DateTime.now().difference(startTime).inMilliseconds;

    telemetryService.record(
      TelemetryEvent.apiRequest(
        endpoint: endpoint,
        method: method,
        durationMs: durationMs,
        statusCode: statusCode,
        cached: cached,
        etagHit: etagHit,
        retryCount: retryCount,
        timestamp: DateTime.now(),
      ),
    );
  }
}
