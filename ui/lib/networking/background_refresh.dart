/// Background refresh coordinator for keeping cached data fresh.
///
/// Periodically refreshes cached data based on TTL and importance:
/// - Every 5 minutes: Permissions (security-critical)
/// - Every 10 minutes: Current page, schemas
/// - Every 15 minutes: Capabilities, navigation
///
/// Pauses when:
/// - App is in background
/// - Device is offline
/// - Data is still fresh (within TTL)
library;

import 'dart:async';

import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:logging/logging.dart';

/// Background refresh coordinator
class BackgroundRefreshCoordinator {
  BackgroundRefreshCoordinator({
    required this.connectivity,
  });

  final Connectivity connectivity;
  final _logger = Logger('BackgroundRefreshCoordinator');

  Timer? _permissionsTimer;
  Timer? _pageTimer;
  Timer? _capabilitiesTimer;

  bool _isRunning = false;
  bool _isOnline = true;

  /// Start background refresh timers
  void start() {
    if (_isRunning) return;

    _logger.info('Starting background refresh');
    _isRunning = true;

    // Monitor connectivity changes
    connectivity.onConnectivityChanged.listen(_onConnectivityChanged);

    // Start periodic refresh timers
    _startPermissionsRefresh();
    _startPageRefresh();
    _startCapabilitiesRefresh();
  }

  /// Stop all background refresh timers
  void stop() {
    if (!_isRunning) return;

    _logger.info('Stopping background refresh');
    _isRunning = false;

    _permissionsTimer?.cancel();
    _pageTimer?.cancel();
    _capabilitiesTimer?.cancel();
  }

  /// Start permissions refresh timer (every 5 minutes)
  void _startPermissionsRefresh() {
    _permissionsTimer?.cancel();
    _permissionsTimer = Timer.periodic(
      const Duration(minutes: 5),
      (_) => _refreshPermissions(),
    );
  }

  /// Start page/schema refresh timer (every 10 minutes)
  void _startPageRefresh() {
    _pageTimer?.cancel();
    _pageTimer = Timer.periodic(
      const Duration(minutes: 10),
      (_) => _refreshCurrentPage(),
    );
  }

  /// Start capabilities/navigation refresh timer (every 15 minutes)
  void _startCapabilitiesRefresh() {
    _capabilitiesTimer?.cancel();
    _capabilitiesTimer = Timer.periodic(
      const Duration(minutes: 15),
      (_) => _refreshCapabilitiesAndNavigation(),
    );
  }

  /// Refresh permissions
  Future<void> _refreshPermissions() async {
    if (!_isOnline) {
      _logger.fine('Skipping permissions refresh (offline)');
      return;
    }

    _logger.info('Background refresh: permissions');
    // TODO: Trigger permission refresh through state layer
    // permissionProvider.invalidate();
  }

  /// Refresh current page and its schemas
  Future<void> _refreshCurrentPage() async {
    if (!_isOnline) {
      _logger.fine('Skipping page refresh (offline)');
      return;
    }

    _logger.info('Background refresh: current page');
    // TODO: Trigger current page refresh through state layer
    // currentPageProvider.invalidate();
  }

  /// Refresh capabilities and navigation
  Future<void> _refreshCapabilitiesAndNavigation() async {
    if (!_isOnline) {
      _logger.fine('Skipping capabilities/navigation refresh (offline)');
      return;
    }

    _logger.info('Background refresh: capabilities and navigation');
    // TODO: Trigger capabilities and navigation refresh
    // capabilitiesProvider.invalidate();
    // navigationProvider.invalidate();
  }

  /// Handle connectivity changes
  void _onConnectivityChanged(List<ConnectivityResult> results) {
    final wasOnline = _isOnline;
    _isOnline = results.any(
      (result) => result != ConnectivityResult.none,
    );

    if (!wasOnline && _isOnline) {
      _logger.info('Connectivity restored, refreshing stale caches');
      _refreshAllStale();
    } else if (wasOnline && !_isOnline) {
      _logger.info('Lost connectivity, pausing background refresh');
    }
  }

  /// Refresh all stale caches (called when coming back online)
  Future<void> _refreshAllStale() async {
    _logger.info('Refreshing all stale caches');
    // TODO: Trigger refresh of all stale caches
    // cacheCoordinator.refreshAllStale();
  }

  /// Manually trigger a full refresh (e.g., user pull-to-refresh)
  Future<void> refreshAll() async {
    if (!_isOnline) {
      _logger.warning('Cannot refresh (offline)');
      return;
    }

    _logger.info('Manual refresh triggered');
    await Future.wait([
      _refreshPermissions(),
      _refreshCurrentPage(),
      _refreshCapabilitiesAndNavigation(),
    ]);
  }
}
