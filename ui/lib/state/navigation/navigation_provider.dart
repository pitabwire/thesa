/// Navigation provider for sidebar menu tree.
///
/// Cache-first with 15-minute TTL.
/// Always alive - never disposed.
library;

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

    final cacheCoordinator = await ref.read(cacheCoordinatorProvider.future);
    final bffClient = ref.read(bffClientProvider);

    try {
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
