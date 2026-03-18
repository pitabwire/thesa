/// Deduplication interceptor to prevent duplicate simultaneous requests.
///
/// If the same URL is requested multiple times before the first request
/// completes, all callers share the same response.
library;

import 'dart:async';

import 'package:dio/dio.dart';
import 'package:logging/logging.dart';

/// Request deduplication interceptor
class DeduplicationInterceptor extends Interceptor {
  DeduplicationInterceptor();

  final _logger = Logger('DeduplicationInterceptor');

  // Map of in-flight requests
  final Map<String, Completer<Response>> _inFlightRequests = {};

  @override
  Future<void> onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    // Only deduplicate GET requests
    if (options.method.toUpperCase() != 'GET') {
      return handler.next(options);
    }

    final key = _buildRequestKey(options);

    // Check if this request is already in flight
    if (_inFlightRequests.containsKey(key)) {
      _logger.info('Deduplicating request: $key');

      try {
        // Wait for the in-flight request to complete
        final response = await _inFlightRequests[key]!.future;

        // Return the shared response
        return handler.resolve(response);
      } catch (e) {
        // If the in-flight request failed, let this one proceed
        _logger.warning('In-flight request failed, proceeding with new request');
        return handler.next(options);
      }
    }

    // This is a new request - add it to the in-flight map
    _inFlightRequests[key] = Completer<Response>();

    handler.next(options);
  }

  @override
  void onResponse(Response response, ResponseInterceptorHandler handler) {
    final key = _buildRequestKey(response.requestOptions);

    // Complete the completer for this request
    if (_inFlightRequests.containsKey(key)) {
      _inFlightRequests[key]!.complete(response);
      _inFlightRequests.remove(key);
    }

    handler.next(response);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    final key = _buildRequestKey(err.requestOptions);

    // Complete the completer with an error
    if (_inFlightRequests.containsKey(key)) {
      _inFlightRequests[key]!.completeError(err);
      _inFlightRequests.remove(key);
    }

    handler.next(err);
  }

  /// Build a unique key for a request
  String _buildRequestKey(RequestOptions options) {
    // Include method, path, and query parameters
    final query = options.queryParameters.isEmpty
        ? ''
        : '?${options.queryParameters.entries.map((e) => '${e.key}=${e.value}').join('&')}';

    // For POST/PUT/PATCH, also include body to differentiate
    final bodyHash = options.data != null ? options.data.hashCode : 0;

    return '${options.method}:${options.path}$query:$bodyHash';
  }

  /// Clear all in-flight request tracking
  void clearAll() {
    // Complete all pending requests with an error
    for (final completer in _inFlightRequests.values) {
      if (!completer.isCompleted) {
        completer.completeError(
          DioException(
            requestOptions: RequestOptions(),
            message: 'Request deduplication cleared',
          ),
        );
      }
    }
    _inFlightRequests.clear();
  }
}
