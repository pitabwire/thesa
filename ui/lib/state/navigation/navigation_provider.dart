/// Navigation provider for sidebar menu tree.
///
/// Cache-first with 15-minute TTL.
/// Always alive - never disposed.
library;

import 'dart:async';

import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../core/core.dart';
import '../core/dependencies_provider.dart';

part 'navigation_provider.g.dart';

final _logger = Logger('NavigationProvider');

/// Navigation provider
@Riverpod(keepAlive: true)
class Navigation extends _$Navigation {
  @override
  Future<NavigationTree> build() async {
    _logger.info('Loading navigation tree');

    final bffClient = ref.read(bffClientProvider);

    // Try cache-first, but fall back to direct BFF call if the cache
    // layer is unavailable (e.g. Drift WASM not loaded on web).
    try {
      final cacheCoordinator = await ref
          .read(cacheCoordinatorProvider.future)
          .timeout(const Duration(seconds: 3));

      final result = await cacheCoordinator.getNavigation(
        'main',
        fetchFromNetwork: bffClient.getNavigation,
      );

      final data = result.data;
      if (data == null) {
        throw StateError('Navigation data was null');
      }

      _logger.info(
        'Navigation loaded: ${result.state.name} '
        '(${data.items.length} items)',
      );

      return data;
    } on TimeoutException {
      _logger.warning('Cache coordinator timed out, fetching directly from BFF');
    } catch (e, stack) {
      _logger.warning('Cache-first navigation failed, trying direct BFF', e, stack);
    }

    // Direct BFF fallback — no cache involved.
    try {
      final data = await bffClient.getNavigation();
      _logger.info('Navigation loaded from BFF (${data.items.length} items)');
      return data;
    } catch (e, stack) {
      _logger.severe('Failed to load navigation', e, stack);
      rethrow;
    }
  }

  /// Refresh navigation from server
  Future<void> refresh() async {
    _logger.info('Refreshing navigation');
    state = const AsyncValue.loading();
    state = await AsyncValue.guard(() async {
      final bffClient = ref.read(bffClientProvider);
      return bffClient.getNavigation();
    });
  }

  /// Get visible navigation items (filtered by permissions)
  List<NavigationItem> get visibleItems {
    return state.value?.items
            .where((item) => item.permission.allowed)
            .toList() ??
        [];
  }
}
