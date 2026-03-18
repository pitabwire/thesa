/// Capabilities provider for global feature flags and version.
///
/// Cache-first with 15-minute TTL.
/// Always alive - never disposed.
library;

import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../cache/cache_policy.dart';
import '../../core/core.dart';
import '../core/dependencies_provider.dart';

part 'capabilities_provider.g.dart';

final _logger = Logger('CapabilitiesProvider');

/// Capabilities provider
@Riverpod(keepAlive: true)
class CapabilitiesNotifier extends _$CapabilitiesNotifier {
  @override
  Future<Capabilities> build() async {
    _logger.info('Loading capabilities');

    // Note: Capabilities don't use cache coordinator since they have no specific ID
    // We implement a simple cache-first pattern inline
    final bffClient = ref.read(bffClientProvider);

    try {
      final capabilities = await bffClient.getCapabilities();
      _logger.info('Capabilities loaded: version ${capabilities.version}');
      return capabilities;
    } catch (e, stack) {
      _logger.severe('Failed to load capabilities', e, stack);
      rethrow;
    }
  }

  /// Refresh capabilities from server
  Future<void> refresh() async {
    _logger.info('Refreshing capabilities');
    state = const AsyncValue.loading();
    state = await AsyncValue.guard(() async {
      final bffClient = ref.read(bffClientProvider);
      return await bffClient.getCapabilities();
    });
  }

  /// Check if a feature is enabled
  bool isFeatureEnabled(String feature) {
    return state.valueOrNull?.capabilities[feature]?.enabled ?? false;
  }

  /// Get global version for cache invalidation
  String? get globalVersion => state.valueOrNull?.version;
}
