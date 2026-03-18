/// Dio factory for creating configured HTTP clients.
///
/// Sets up the complete interceptor chain and configures timeouts,
/// base URL, and other HTTP client settings.
library;

import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../telemetry/telemetry_service.dart';
import 'interceptors/auth_interceptor.dart';
import 'interceptors/dedup_interceptor.dart';
import 'interceptors/etag_interceptor.dart';
import 'interceptors/retry_interceptor.dart';
import 'interceptors/telemetry_interceptor.dart';

/// Creates and configures a Dio instance with the full interceptor chain
class DioFactory {
  DioFactory._();

  /// Create a configured Dio instance
  static Dio create({
    required String baseUrl,
    required FlutterSecureStorage secureStorage,
    required TelemetryService telemetryService,
    Future<String?> Function()? onRefreshToken,
    Duration connectTimeout = const Duration(seconds: 15),
    Duration receiveTimeout = const Duration(seconds: 30),
    Duration sendTimeout = const Duration(seconds: 30),
  }) {
    final dio = Dio(
      BaseOptions(
        baseUrl: baseUrl,
        connectTimeout: connectTimeout,
        receiveTimeout: receiveTimeout,
        sendTimeout: sendTimeout,
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
        },
        validateStatus: (status) {
          // Don't throw on any status code - let interceptors handle it
          return status != null && status < 500;
        },
      ),
    );

    // Add interceptors in order
    // The order matters - they execute in order for requests,
    // and in reverse order for responses

    // 1. Auth interceptor (adds token, handles 401 refresh via OIDC)
    dio.interceptors.add(
      AuthInterceptor(
        secureStorage: secureStorage,
        dio: dio,
        telemetryService: telemetryService,
        onRefreshToken: onRefreshToken,
      ),
    );

    // 2. ETag interceptor (cache validation)
    dio.interceptors.add(ETagInterceptor());

    // 3. Deduplication interceptor (prevent duplicate requests)
    dio.interceptors.add(DeduplicationInterceptor());

    // 4. Telemetry interceptor (performance tracking)
    dio.interceptors.add(
      TelemetryInterceptor(telemetryService: telemetryService),
    );

    // 5. Retry interceptor (exponential backoff)
    dio.interceptors.add(RetryInterceptor(dio: dio));

    // Add logging in debug mode
    if (const bool.fromEnvironment('dart.vm.product') == false) {
      dio.interceptors.add(
        LogInterceptor(
          requestBody: true,
          responseBody: true,
          requestHeader: false,
          responseHeader: false,
        ),
      );
    }

    return dio;
  }
}
