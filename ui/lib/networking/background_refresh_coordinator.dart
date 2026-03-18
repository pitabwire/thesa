/// Background refresh coordinator for keeping cached data fresh.
library;

import 'dart:async';

import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:logger/logger.dart';
import 'package:riverpod/riverpod.dart';

/// Background refresh coordinator
///
/// Schedules periodic refreshes of cached data to keep it fresh without blocking the UI.
/// Refreshes at different intervals based on data criticality:
/// - Permissions: Every 5 minutes (security-critical)
/// - Current page: Every 10 minutes
/// - Capabilities/navigation: Every 15 minutes
///
/// On connectivity change (offline → online), refreshes all stale caches immediately.
class BackgroundRefreshCoordinator {
  BackgroundRefreshCoordinator({
    required this.ref,
    this.permissionsRefreshInterval = const Duration(minutes: 5),
    this.pageRefreshInterval = const Duration(minutes: 10),
    this.navigationRefreshInterval = const Duration(minutes: 15),
  }) {
    _startPeriodicRefresh();
    _listenToConnectivityChanges();
  }

  final Ref ref;

  /// Refresh interval for permissions (security-critical, short TTL)
  final Duration permissionsRefreshInterval;

  /// Refresh interval for current page and schemas
  final Duration pageRefreshInterval;

  /// Refresh interval for capabilities and navigation tree
  final Duration navigationRefreshInterval;

  final Logger _logger = Logger();

  Timer? _permissionsTimer;
  Timer? _pageTimer;
  Timer? _navigationTimer;

  StreamSubscription<List<ConnectivityResult>>? _connectivitySubscription;
  bool _wasOffline = false;

  /// Start periodic refresh timers
  void _startPeriodicRefresh() {
    // Permissions refresh (every 5 minutes)
    _permissionsTimer = Timer.periodic(
      permissionsRefreshInterval,
      (_) => _refreshPermissions(),
    );

    // Page refresh (every 10 minutes)
    _pageTimer = Timer.periodic(
      pageRefreshInterval,
      (_) => _refreshCurrentPage(),
    );

    // Navigation refresh (every 15 minutes)
    _navigationTimer = Timer.periodic(
      navigationRefreshInterval,
      (_) => _refreshNavigation(),
    );

    _logger.info('Background refresh coordinator started');
  }

  /// Listen to connectivity changes and trigger refresh on recovery
  void _listenToConnectivityChanges() {
    _connectivitySubscription = Connectivity()
        .onConnectivityChanged
        .listen((List<ConnectivityResult> results) {
      final isOffline = results.every(
        (result) => result == ConnectivityResult.none,
      );

      // If we were offline and now we're online, refresh all stale caches
      if (_wasOffline && !isOffline) {
        _logger.info('Connectivity restored, refreshing all stale caches');
        _refreshAllOnConnectivityRecovery();
      }

      _wasOffline = isOffline;
    });
  }

  /// Refresh permissions in background
  Future<void> _refreshPermissions() async {
    try {
      _logger.fine('Background refresh: permissions');
      // TODO: Invalidate permissions provider to trigger refresh
      // ref.invalidate(permissionsProvider);
    } catch (error, stack) {
      _logger.warning(
        'Failed to refresh permissions in background',
        error,
        stack,
      );
    }
  }

  /// Refresh current page and schemas in background
  Future<void> _refreshCurrentPage() async {
    try {
      _logger.fine('Background refresh: current page');
      // TODO: Invalidate current page provider to trigger refresh
      // Note: This should use ETag to avoid unnecessary data transfer
      // ref.invalidate(currentPageProvider);
    } catch (error, stack) {
      _logger.warning(
        'Failed to refresh current page in background',
        error,
        stack,
      );
    }
  }

  /// Refresh capabilities and navigation tree in background
  Future<void> _refreshNavigation() async {
    try {
      _logger.fine('Background refresh: navigation');
      // TODO: Invalidate navigation and capabilities providers
      // ref.invalidate(navigationProvider);
      // ref.invalidate(capabilitiesProvider);
    } catch (error, stack) {
      _logger.warning(
        'Failed to refresh navigation in background',
        error,
        stack,
      );
    }
  }

  /// Refresh all stale caches when connectivity is restored
  Future<void> _refreshAllOnConnectivityRecovery() async {
    try {
      // Refresh all providers concurrently
      await Future.wait([
        _refreshPermissions(),
        _refreshCurrentPage(),
        _refreshNavigation(),
      ]);

      _logger.info('All stale caches refreshed after connectivity recovery');
    } catch (error, stack) {
      _logger.severe(
        'Failed to refresh caches after connectivity recovery',
        error,
        stack,
      );
    }
  }

  /// Pause background refresh (e.g., when app is backgrounded)
  void pause() {
    _permissionsTimer?.cancel();
    _pageTimer?.cancel();
    _navigationTimer?.cancel();
    _logger.info('Background refresh paused');
  }

  /// Resume background refresh
  void resume() {
    _startPeriodicRefresh();
    _logger.info('Background refresh resumed');
  }

  /// Dispose resources
  void dispose() {
    _permissionsTimer?.cancel();
    _pageTimer?.cancel();
    _navigationTimer?.cancel();
    _connectivitySubscription?.cancel();

    _permissionsTimer = null;
    _pageTimer = null;
    _navigationTimer = null;
    _connectivitySubscription = null;

    _logger.info('Background refresh coordinator disposed');
  }
}
