/// Authentication interceptor for adding Bearer tokens and handling 401 refresh.
///
/// On request:
/// - Reads access token from secure storage and adds Authorization header
///
/// On 401 response:
/// - Attempts OIDC token refresh via the auth provider
/// - Retries the original request with the new token
/// - Forces logout if refresh fails
library;

import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:logging/logging.dart';

import '../../state/auth/oidc_service.dart';
import '../../telemetry/models/telemetry_event.dart';
import '../../telemetry/telemetry_service.dart';

/// Authentication interceptor
class AuthInterceptor extends Interceptor {
  AuthInterceptor({
    required this.secureStorage,
    required this.dio,
    this.telemetryService,
    this.onRefreshToken,
  });

  final FlutterSecureStorage secureStorage;
  final Dio dio;
  final TelemetryService? telemetryService;

  /// Callback to refresh tokens via the auth provider.
  /// Returns the new access token, or null if refresh failed.
  final Future<String?> Function()? onRefreshToken;

  final _logger = Logger('AuthInterceptor');

  // Mutex for token refresh to prevent concurrent refreshes
  Completer<String?>? _refreshCompleter;

  @override
  Future<void> onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    // Read access token from secure storage
    final accessToken = await secureStorage.read(
      key: AuthStorageKeys.accessToken,
    );

    if (accessToken != null) {
      options.headers['Authorization'] = 'Bearer $accessToken';
    }

    handler.next(options);
  }

  @override
  Future<void> onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) async {
    // Only handle 401 Unauthorized errors
    if (err.response?.statusCode != 401) {
      return handler.next(err);
    }

    _logger.info('Received 401, attempting OIDC token refresh');

    try {
      final newAccessToken = await _refreshToken();

      if (newAccessToken == null) {
        _logger.warning('Token refresh failed, session expired');
        return handler.reject(err);
      }

      // Retry the original request with the new token using the same
      // configured Dio instance (with base URL, timeouts, etc.)
      _logger.info('Token refreshed, retrying request');
      final options = err.requestOptions;
      options.headers['Authorization'] = 'Bearer $newAccessToken';

      final response = await dio.fetch<dynamic>(options);
      return handler.resolve(response);
    } catch (e, stack) {
      _logger.severe('Error during token refresh', e, stack);
      return handler.reject(err);
    }
  }

  Future<String?> _refreshToken() async {
    // If a refresh is already in progress, wait for it
    if (_refreshCompleter != null) {
      _logger.fine('Token refresh already in progress, waiting...');
      return _refreshCompleter!.future;
    }

    _refreshCompleter = Completer<String?>();
    final startTime = DateTime.now();

    try {
      if (onRefreshToken == null) {
        _logger.warning('No token refresh callback configured');
        _refreshCompleter!.complete(null);
        return null;
      }

      final newToken = await onRefreshToken!();

      _recordRefreshTelemetry(
        startTime: startTime,
        success: newToken != null,
        errorMessage: newToken == null ? 'Refresh returned null' : null,
      );

      _refreshCompleter!.complete(newToken);
      return newToken;
    } catch (e, stack) {
      _logger.severe('Token refresh request failed', e, stack);
      _recordRefreshTelemetry(
        startTime: startTime,
        success: false,
        errorMessage: e.toString(),
      );
      _refreshCompleter!.complete(null);
      return null;
    } finally {
      _refreshCompleter = null;
    }
  }

  void _recordRefreshTelemetry({
    required DateTime startTime,
    required bool success,
    String? errorMessage,
  }) {
    if (telemetryService == null) {
      return;
    }

    final durationMs = DateTime.now().difference(startTime).inMilliseconds;
    telemetryService!.record(
      TelemetryEvent.authRefresh(
        success: success,
        durationMs: durationMs,
        triggeredBy: '401_response',
        errorMessage: errorMessage,
        timestamp: DateTime.now(),
      ),
    );
  }
}
