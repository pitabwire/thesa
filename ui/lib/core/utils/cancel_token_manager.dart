/// Cancel token management for request cancellation.
library;

import 'package:dio/dio.dart';
import 'package:logger/logger.dart';

/// Manages cancel tokens for page and data requests
///
/// Allows cancellation of in-flight requests when the user navigates away
/// or when a new request supersedes a pending one.
class CancelTokenManager {
  CancelTokenManager();

  final Logger _logger = Logger();

  /// Active cancel tokens keyed by request identifier
  final Map<String, CancelToken> _tokens = {};

  /// Create a new cancel token for a request
  ///
  /// If a token already exists for this ID, the old request is cancelled
  /// and a new token is created.
  CancelToken create(String requestId) {
    // Cancel any existing request with this ID
    cancel(requestId);

    // Create new token
    final token = CancelToken();
    _tokens[requestId] = token;

    _logger.fine('Created cancel token for: $requestId');
    return token;
  }

  /// Get an existing cancel token
  CancelToken? get(String requestId) {
    return _tokens[requestId];
  }

  /// Cancel a specific request
  void cancel(String requestId, {String? reason}) {
    final token = _tokens[requestId];
    if (token != null && !token.isCancelled) {
      token.cancel(reason ?? 'Request cancelled');
      _logger.fine('Cancelled request: $requestId (reason: $reason)');
    }
    _tokens.remove(requestId);
  }

  /// Cancel all active requests
  void cancelAll({String? reason}) {
    final count = _tokens.length;
    for (final entry in _tokens.entries) {
      if (!entry.value.isCancelled) {
        entry.value.cancel(reason ?? 'All requests cancelled');
      }
    }
    _tokens.clear();
    _logger.info('Cancelled all requests (count: $count, reason: $reason)');
  }

  /// Remove a token after request completes
  void complete(String requestId) {
    _tokens.remove(requestId);
    _logger.fine('Request completed: $requestId');
  }

  /// Get count of active tokens
  int get activeCount => _tokens.length;

  /// Check if a request is active
  bool isActive(String requestId) {
    final token = _tokens[requestId];
    return token != null && !token.isCancelled;
  }
}
