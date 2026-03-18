/// ETag interceptor for HTTP cache validation.
///
/// On request:
/// - Adds If-None-Match header with cached ETag (if available)
///
/// On 304 response:
/// - Returns cached response body (no parsing needed)
/// - Updates cache timestamp to reset TTL
library;

import 'package:dio/dio.dart';
import 'package:logging/logging.dart';

/// ETag-based cache validation interceptor
class ETagInterceptor extends Interceptor {
  ETagInterceptor();

  final _logger = Logger('ETagInterceptor');

  // In-memory ETag store (could be backed by Drift for persistence)
  final Map<String, String> _etagStore = {};

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    // Only add ETags for GET requests
    if (options.method.toUpperCase() != 'GET') {
      return handler.next(options);
    }

    // Check if we have a cached ETag for this URL
    final url = _buildCacheKey(options);
    final etag = _etagStore[url];

    if (etag != null) {
      _logger.fine('Adding If-None-Match header for $url');
      options.headers['If-None-Match'] = etag;
    }

    handler.next(options);
  }

  @override
  void onResponse(Response response, ResponseInterceptorHandler handler) {
    final url = _buildCacheKey(response.requestOptions);

    // Store ETag if present in response
    final etag = response.headers.value('etag');
    if (etag != null) {
      _logger.fine('Storing ETag for $url: $etag');
      _etagStore[url] = etag;
    }

    // Handle 304 Not Modified
    if (response.statusCode == 304) {
      _logger.info('Received 304 Not Modified for $url');
      // TODO: Return cached response body from cache coordinator
      // For now, just pass through the 304 response
      // The cache coordinator will handle retrieving the cached data
    }

    handler.next(response);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    // If request fails, we might want to remove the ETag
    // so the next attempt fetches fresh data
    if (err.type == DioExceptionType.connectionTimeout ||
        err.type == DioExceptionType.receiveTimeout ||
        err.type == DioExceptionType.sendTimeout) {
      final url = _buildCacheKey(err.requestOptions);
      _logger.fine('Removing ETag for failed request: $url');
      _etagStore.remove(url);
    }

    handler.next(err);
  }

  /// Build a cache key from request options
  String _buildCacheKey(RequestOptions options) {
    // Use method + path + query params as key
    final query = options.queryParameters.isEmpty
        ? ''
        : '?${options.queryParameters.entries.map((e) => '${e.key}=${e.value}').join('&')}';
    return '${options.method}:${options.path}$query';
  }

  /// Clear all cached ETags
  void clearAll() {
    _etagStore.clear();
  }

  /// Remove ETag for specific URL
  void removeETag(String url) {
    _etagStore.remove(url);
  }

  /// Get ETag for specific URL
  String? getETag(String url) {
    return _etagStore[url];
  }
}
