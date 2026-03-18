/// Core dependency providers for shared services.
///
/// Provides BFF client, cache coordinator, and database instances.
library;

import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../cache/cache_coordinator.dart';
import '../../cache/database/app_database.dart';
import '../../core/config/app_config.dart';
import '../../networking/bff_client.dart';
import '../../networking/dio_factory.dart';
import '../../telemetry/telemetry_provider.dart';
import '../auth/auth_provider.dart';
import '../auth/oidc_service.dart';

part 'dependencies_provider.g.dart';

/// Secure storage provider (singleton)
@Riverpod(keepAlive: true)
FlutterSecureStorage secureStorage(Ref ref) {
  return const FlutterSecureStorage();
}

/// Dio instance provider (singleton)
@Riverpod(keepAlive: true)
Dio dio(Ref ref) {
  final secureStorage = ref.watch(secureStorageProvider);
  final telemetryService = ref.watch(telemetryServiceProvider);
  final oidc = ref.watch(oidcServiceProvider);

  return DioFactory.create(
    baseUrl: AppConfig.bffBaseUrl,
    secureStorage: secureStorage,
    telemetryService: telemetryService,
    onRefreshToken: () async {
      final result = await oidc.refreshTokenWithResult();
      if (result.result == TokenRefreshResult.success) {
        return result.token?.accessToken;
      }
      return null;
    },
  );
}

/// BFF client provider (singleton)
@Riverpod(keepAlive: true)
BffClient bffClient(Ref ref) {
  final dio = ref.watch(dioProvider);
  return BffClient(dio);
}

/// App database provider (singleton)
@Riverpod(keepAlive: true)
Future<AppDatabase> database(Ref ref) async {
  final db = createDatabase();
  ref.onDispose(db.close);
  return db;
}

/// Cache coordinator provider (singleton)
@Riverpod(keepAlive: true)
Future<CacheCoordinator> cacheCoordinator(Ref ref) async {
  final database = await ref.watch(databaseProvider.future);
  return CacheCoordinator(database);
}
