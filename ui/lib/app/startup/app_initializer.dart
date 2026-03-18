/// App initialization and startup optimization.
library;

import 'package:logger/logger.dart';
import 'package:riverpod/riverpod.dart';

/// App initializer for optimized startup sequence
///
/// Handles parallel initialization of core services:
/// - Capabilities
/// - Navigation
/// - Session/authentication
///
/// On warm start: renders from cache first (~90ms), then background refresh
/// On cold start: shows loading skeletons while initial data loads
class AppInitializer {
  AppInitializer({
    required this.ref,
  });

  final Ref ref;
  final Logger _logger = Logger();

  /// Initialize app with parallel loading
  ///
  /// Returns true if initialization succeeded, false otherwise.
  Future<bool> initialize() async {
    final startTime = DateTime.now();
    _logger.info('App initialization started');

    try {
      // Initialize core services in parallel
      await Future.wait([
        _initializeCapabilities(),
        _initializeNavigation(),
        _initializeSession(),
      ]);

      final duration = DateTime.now().difference(startTime).inMilliseconds;
      _logger.info('App initialization completed in ${duration}ms');

      return true;
    } catch (error, stack) {
      _logger.severe('App initialization failed', error, stack);
      return false;
    }
  }

  /// Initialize capabilities
  Future<void> _initializeCapabilities() async {
    try {
      _logger.fine('Initializing capabilities...');
      // TODO: Load capabilities provider
      // await ref.read(capabilitiesProvider.future);
    } catch (error, stack) {
      _logger.warning('Failed to initialize capabilities', error, stack);
      // Non-critical, can continue without capabilities
    }
  }

  /// Initialize navigation tree
  Future<void> _initializeNavigation() async {
    try {
      _logger.fine('Initializing navigation...');
      // TODO: Load navigation provider
      // await ref.read(navigationProvider.future);
    } catch (error, stack) {
      _logger.warning('Failed to initialize navigation', error, stack);
      // Non-critical, can show empty navigation
    }
  }

  /// Initialize session/authentication
  Future<void> _initializeSession() async {
    try {
      _logger.fine('Initializing session...');
      // TODO: Restore session from secure storage
      // This is critical - if it fails, user needs to log in
    } catch (error, stack) {
      _logger.warning('Failed to initialize session', error, stack);
      // Session init failure means user needs to log in
    }
  }

  /// Warm start optimization
  ///
  /// Renders immediately from cache, then refreshes in background.
  /// Target: <100ms to first render from cache
  Future<void> warmStart() async {
    _logger.info('Warm start: rendering from cache');

    // Cache-first rendering happens automatically via providers
    // The providers check cache first, then schedule background refresh

    // Schedule background refresh of all stale data
    // This is handled by BackgroundRefreshCoordinator
  }

  /// Cold start with loading skeletons
  ///
  /// Shows skeleton loaders while data loads for the first time.
  /// Prevents layout shift when data arrives.
  Future<void> coldStart() async {
    _logger.info('Cold start: loading initial data');

    // Show loading skeletons in sidebar and content
    // This is handled by the UI layer (AppShell, PageRenderer)

    // Load initial data
    await initialize();
  }
}
