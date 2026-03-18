/// Central error handling and reporting.
///
/// Provides unified error processing for:
/// - User-facing error messages
/// - Telemetry reporting
/// - Error recovery strategies
library;

import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:logging/logging.dart';

import 'app_error.dart';

/// Central error handler
class ErrorHandler {
  ErrorHandler._();

  static final _logger = Logger('ErrorHandler');

  /// Handle an error and return user-facing message
  static String handleError(Object error, [StackTrace? stackTrace]) {
    // If it's already an AppError, use its message
    if (error is AppError) {
      _logError(error, stackTrace);
      return error.userMessage;
    }

    // Convert other errors to AppError
    final appError = _convertToAppError(error, stackTrace);
    _logError(appError, stackTrace);
    return appError.userMessage;
  }

  /// Convert arbitrary error to AppError
  static AppError _convertToAppError(Object error, [StackTrace? stackTrace]) {
    final errorString = error.toString().toLowerCase();

    // Network errors
    if (errorString.contains('socket') ||
        errorString.contains('connection') ||
        errorString.contains('timeout') ||
        errorString.contains('network')) {
      return AppError.network(
        message: error.toString(),
        originalError: error,
        stackTrace: stackTrace,
      );
    }

    // Parse errors
    if (errorString.contains('json') ||
        errorString.contains('parse') ||
        errorString.contains('format')) {
      return AppError.parse(
        message: error.toString(),
        originalError: error,
        stackTrace: stackTrace,
      );
    }

    // Unknown error
    return AppError.unknown(
      message: error.toString(),
      originalError: error,
      stackTrace: stackTrace,
    );
  }

  /// Log error with appropriate level
  static void _logError(AppError error, [StackTrace? stackTrace]) {
    final severity = error.severity;
    final message = 'AppError [${severity.name}]: ${error.userMessage}';

    switch (severity) {
      case ErrorSeverity.info:
        _logger.info(message, error, stackTrace);
      case ErrorSeverity.warning:
        _logger.warning(message, error, stackTrace);
      case ErrorSeverity.error:
        _logger.severe(message, error, stackTrace);
      case ErrorSeverity.critical:
        _logger.shout(message, error, stackTrace);
    }

    // TODO: Send to telemetry service in production
    if (!kDebugMode && severity == ErrorSeverity.critical) {
      // Report critical errors to monitoring system
    }
  }

  /// Handle async errors in a zone
  static Future<T> handleAsync<T>(
    Future<T> Function() operation, {
    String? operationName,
  }) async {
    try {
      return await operation();
    } catch (error, stackTrace) {
      _logger.severe(
        'Async error in ${operationName ?? "operation"}',
        error,
        stackTrace,
      );
      rethrow;
    }
  }

  /// Run operation with error boundary
  static Future<T?> runSafe<T>(
    Future<T> Function() operation, {
    String? operationName,
    T? fallbackValue,
  }) async {
    try {
      return await operation();
    } catch (error, stackTrace) {
      handleError(error, stackTrace);
      return fallbackValue;
    }
  }

  /// Create an error boundary for synchronous operations
  static T? runSafeSync<T>(
    T Function() operation, {
    String? operationName,
    T? fallbackValue,
  }) {
    try {
      return operation();
    } catch (error, stackTrace) {
      handleError(error, stackTrace);
      return fallbackValue;
    }
  }
}
