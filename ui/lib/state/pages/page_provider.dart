/// Page provider (family) for page descriptors.
///
/// Cache-first with 10-minute TTL.
/// Auto-dispose when page is no longer displayed.
library;

import 'dart:async';

import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../core/core.dart';
import '../core/dependencies_provider.dart';

part 'page_provider.g.dart';

final _logger = Logger('PageProvider');

/// Page provider (family - one instance per page ID)
@riverpod
class Page extends _$Page {
  @override
  Future<PageDescriptor> build(String pageId) async {
    _logger.info('Loading page: $pageId');

    final bffClient = ref.read(bffClientProvider);

    // Try cache-first, but fall back to direct BFF call if the cache
    // layer is unavailable (e.g. Drift WASM not loaded on web).
    try {
      final cacheCoordinator = await ref
          .read(cacheCoordinatorProvider.future)
          .timeout(const Duration(seconds: 3));

      final result = await cacheCoordinator.getPage(
        pageId,
        fetchFromNetwork: () => bffClient.getPage(pageId),
      );

      final data = result.data;
      if (data == null) {
        throw StateError('Page data was null for: $pageId');
      }

      _logger.info(
        'Page loaded: ${result.state.name} '
        '(${data.components.length} components)',
      );

      return data;
    } on TimeoutException {
      _logger.warning('Cache coordinator timed out for $pageId, fetching directly');
    } catch (e, stack) {
      _logger.warning('Cache-first page load failed for $pageId', e, stack);
    }

    // Direct BFF fallback — no cache involved.
    try {
      final data = await bffClient.getPage(pageId);
      _logger.info('Page loaded from BFF: $pageId');
      return data;
    } catch (e, stack) {
      _logger.severe('Failed to load page: $pageId', e, stack);
      rethrow;
    }
  }

  /// Refresh page from server
  Future<void> refresh() async {
    _logger.info('Refreshing page: $pageId');
    ref.invalidateSelf();
  }

  /// Get visible components (filtered by permissions)
  List<ComponentDescriptor> get visibleComponents {
    return state.value?.components
            .where((component) => component.permission.allowed)
            .toList() ??
        [];
  }

  /// Get page actions (filtered by permissions)
  List<ActionDescriptor> get visibleActions {
    return state.value?.actions
            .where((action) => action.permission.allowed)
            .toList() ??
        [];
  }
}
