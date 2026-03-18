/// Retry interceptor for handling transient failures with exponential backoff.
///
/// Retries on:
/// - Network errors (timeout, connection failure)
/// - HTTP 500, 502, 503 (server errors)
/// - HTTP 429 (rate limiting)
///
/// Does NOT retry:
/// - HTTP 400, 401, 403, 404, 422 (client errors)
library;

import 'dart:io';

import 'package:dio/dio.dart';
import 'package:logging/logging.dart';

/// Retry interceptor with exponential backoff
class RetryInterceptor extends Interceptor {
  RetryInterceptor({
    required this.dio,
    this.maxRetries = 3,
  });

  final Dio dio;
  final int maxRetries;
  final _logger = Logger('RetryInterceptor');

  @override
  Future<void> onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) async {
    // Check if this error is retryable
    if (!_isRetryable(err)) {
      return handler.next(err);
    }

    // Check if we've exceeded max retries
    final retryCount = err.requestOptions.extra['retry_count'] as int? ?? 0;
    if (retryCount >= maxRetries) {
      _logger.warning(
        'Max retries ($maxRetries) exceeded for ${err.requestOptions.path}',
      );
      return handler.next(err);
    }

    // Calculate backoff delay (exponential: 1s, 2s, 4s)
    final delay = Duration(seconds: 1 << retryCount);

    _logger.info(
      'Retrying request (attempt ${retryCount + 1}/$maxRetries) '
      'after ${delay.inSeconds}s: ${err.requestOptions.path}',
    );

    // Wait for backoff period
    await Future<void>.delayed(delay);

    // Increment retry count
    err.requestOptions.extra['retry_count'] = retryCount + 1;

    try {
      // Retry the request
      final response = await dio.fetch(err.requestOptions);
      return handler.resolve(response);
    } on DioException catch (e) {
      // If retry fails, pass the new error to the next handler
      return handler.next(e);
    }
  }

  /// Check if an error should be retried
  bool _isRetryable(DioException err) {
    // Network errors are retryable
    if (err.type == DioExceptionType.connectionTimeout ||
        err.type == DioExceptionType.sendTimeout ||
        err.type == DioExceptionType.receiveTimeout ||
        err.type == DioExceptionType.connectionError) {
      return true;
    }

    // Socket exceptions are retryable
    if (err.error is SocketException) {
      return true;
    }

    // Check HTTP status code
    final statusCode = err.response?.statusCode;
    if (statusCode == null) return false;

    // Retry on server errors
    if (statusCode >= 500 && statusCode < 600) {
      return true;
    }

    // Retry on rate limiting
    if (statusCode == 429) {
      return true;
    }

    // Don't retry client errors
    return false;
  }
}
