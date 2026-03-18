/// Application error types and error handling.
///
/// Defines a hierarchy of errors for consistent error handling across the app.
/// All errors extend AppError for unified error reporting and telemetry.
library;

import 'package:freezed_annotation/freezed_annotation.dart';

part 'app_error.freezed.dart';

/// Base error type for all application errors
@Freezed(toJson: false, fromJson: false)
class AppError with _$AppError {
  /// Network error (HTTP, timeout, no connection)
  const factory AppError.network({
    required String message,
    int? statusCode,
    String? endpoint,
    Object? originalError,
    StackTrace? stackTrace,
  }) = NetworkError;

  /// Permission/authorization error
  const factory AppError.permission({
    required String message,
    String? resource,
    String? action,
    Object? originalError,
    StackTrace? stackTrace,
  }) = PermissionError;

  /// Data parsing error (invalid JSON, missing fields)
  const factory AppError.parse({
    required String message,
    String? field,
    Object? originalError,
    StackTrace? stackTrace,
  }) = ParseError;

  /// Cache/database error
  const factory AppError.cache({
    required String message,
    String? operation,
    Object? originalError,
    StackTrace? stackTrace,
  }) = CacheError;

  /// Validation error (form validation, business rule violation)
  const factory AppError.validation({
    required String message,
    Map<String, List<String>>? fieldErrors,
    Object? originalError,
    StackTrace? stackTrace,
  }) = ValidationError;

  /// Not found error (page, resource, component)
  const factory AppError.notFound({
    required String message,
    String? resourceType,
    String? resourceId,
    Object? originalError,
    StackTrace? stackTrace,
  }) = NotFoundError;

  /// Configuration error (missing config, invalid setup)
  const factory AppError.config({
    required String message,
    String? configKey,
    Object? originalError,
    StackTrace? stackTrace,
  }) = ConfigError;

  /// Component rendering error (UI engine error)
  const factory AppError.rendering({
    required String message,
    String? componentType,
    String? componentId,
    Object? originalError,
    StackTrace? stackTrace,
  }) = RenderingError;

  /// Workflow error (step transition, validation)
  const factory AppError.workflow({
    required String message,
    String? workflowId,
    String? stepId,
    Object? originalError,
    StackTrace? stackTrace,
  }) = WorkflowError;

  /// Unknown/unexpected error
  const factory AppError.unknown({
    required String message,
    Object? originalError,
    StackTrace? stackTrace,
  }) = UnknownError;
}

/// Error severity levels
enum ErrorSeverity {
  /// Low severity - informational
  info,

  /// Medium severity - warning
  warning,

  /// High severity - error
  error,

  /// Critical severity - system failure
  critical,
}

/// User-friendly error messages
extension AppErrorMessages on AppError {
  /// Get user-friendly error message
  String get userMessage => when(
        network: (msg, code, endpoint, _, __) {
          if (code == 401) return 'Your session has expired. Please log in again.';
          if (code == 403) return 'You don\'t have permission to perform this action.';
          if (code == 404) return 'The requested resource was not found.';
          if (code == 500) return 'Server error. Please try again later.';
          if (msg.toLowerCase().contains('timeout')) {
            return 'Request timed out. Please check your connection and try again.';
          }
          if (msg.toLowerCase().contains('connection')) {
            return 'No internet connection. Please check your network.';
          }
          return 'Network error: $msg';
        },
        permission: (msg, resource, action, _, __) =>
            'You don\'t have permission to $action ${resource ?? "this resource"}.',
        parse: (msg, field, _, __) => 'Invalid data received from server.',
        cache: (msg, operation, _, __) =>
            'Local storage error. Please try again.',
        validation: (msg, fieldErrors, _, __) => msg,
        notFound: (msg, resourceType, resourceId, _, __) =>
            '${resourceType ?? "Resource"} not found.',
        config: (msg, configKey, _, __) =>
            'Configuration error. Please contact support.',
        rendering: (msg, componentType, componentId, _, __) =>
            'Error displaying this component.',
        workflow: (msg, workflowId, stepId, _, __) => 'Workflow error: $msg',
        unknown: (msg, _, __) => 'An unexpected error occurred: $msg',
      );

  /// Get severity level for this error
  ErrorSeverity get severity => when(
        network: (_, code, __, ___, ____) =>
            code == 500 ? ErrorSeverity.critical : ErrorSeverity.error,
        permission: (_, __, ___, ____, _____) => ErrorSeverity.warning,
        parse: (_, __, ___, ____) => ErrorSeverity.error,
        cache: (_, __, ___, ____) => ErrorSeverity.warning,
        validation: (_, __, ___, ____) => ErrorSeverity.info,
        notFound: (_, __, ___, ____, _____) => ErrorSeverity.warning,
        config: (_, __, ___, ____) => ErrorSeverity.critical,
        rendering: (_, __, ___, ____, _____) => ErrorSeverity.error,
        workflow: (_, __, ___, ____, _____) => ErrorSeverity.warning,
        unknown: (_, __, ___) => ErrorSeverity.error,
      );

  /// Whether this error should be retried
  bool get isRetryable => when(
        network: (_, code, __, ___, ____) => code != 400 && code != 404,
        permission: (_, __, ___, ____, _____) => false,
        parse: (_, __, ___, ____) => false,
        cache: (_, __, ___, ____) => true,
        validation: (_, __, ___, ____) => false,
        notFound: (_, __, ___, ____, _____) => false,
        config: (_, __, ___, ____) => false,
        rendering: (_, __, ___, ____, _____) => false,
        workflow: (_, __, ___, ____, _____) => false,
        unknown: (_, __, ___) => false,
      );
}
