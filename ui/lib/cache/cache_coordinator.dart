/// Cache coordinator - the decision-maker for cache vs network.
///
/// Implements the stale-while-revalidate strategy:
/// 1. Check cache first
/// 2. If fresh → return it (no network)
/// 3. If stale → return it AND refresh in background
/// 4. If empty → fetch from network
library;

import 'dart:convert';

import 'package:drift/drift.dart';
import 'package:logging/logging.dart';

import '../core/core.dart';
import 'cache_policy.dart';
import 'database/app_database.dart';

/// Coordinates cache operations and network fetches
class CacheCoordinator {
  CacheCoordinator(this._database);

  final AppDatabase _database;
  final _logger = Logger('CacheCoordinator');

  /// Get navigation with cache-first strategy
  Future<CacheResult<NavigationTree>> getNavigation(
    String id, {
    required Future<NavigationTree> Function() fetchFromNetwork,
  }) async {
    final cached = await _database.navigationDao.getNavigation(id);

    if (cached == null) {
      // CASE C: Cache is empty
      _logger.info('Navigation cache miss: $id');
      final data = await fetchFromNetwork();
      await _saveNavigation(id, data);
      return CacheResult(
        state: CacheState.empty,
        data: data,
      );
    }

    // Parse cached data
    final data = NavigationTree.fromJson(
      jsonDecode(cached.payload) as Map<String, dynamic>,
    );

    if (CachePolicy.isFresh(cached.expiresAt)) {
      // CASE A: Cache is fresh
      _logger.fine('Navigation cache hit (fresh): $id');
      return CacheResult(
        state: CacheState.fresh,
        data: data,
        etag: cached.etag,
        fetchedAt: cached.fetchedAt,
        expiresAt: cached.expiresAt,
      );
    }

    // CASE B: Cache is stale
    _logger.info('Navigation cache hit (stale): $id');

    // Return stale data immediately
    final result = CacheResult(
      state: CacheState.stale,
      data: data,
      etag: cached.etag,
      fetchedAt: cached.fetchedAt,
      expiresAt: cached.expiresAt,
    );

    // Fire background refresh (don't await)
    _refreshNavigationInBackground(id, fetchFromNetwork);

    return result;
  }

  /// Get page with cache-first strategy
  Future<CacheResult<PageDescriptor>> getPage(
    String pageId, {
    required Future<PageDescriptor> Function() fetchFromNetwork,
  }) async {
    final cached = await _database.pageDao.getPage(pageId);

    if (cached == null) {
      _logger.info('Page cache miss: $pageId');
      final data = await fetchFromNetwork();
      await _savePage(pageId, data);
      return CacheResult(
        state: CacheState.empty,
        data: data,
      );
    }

    final data = PageDescriptor.fromJson(
      jsonDecode(cached.payload) as Map<String, dynamic>,
    );

    if (CachePolicy.isFresh(cached.expiresAt)) {
      _logger.fine('Page cache hit (fresh): $pageId');
      return CacheResult(
        state: CacheState.fresh,
        data: data,
        etag: cached.etag,
        fetchedAt: cached.fetchedAt,
        expiresAt: cached.expiresAt,
      );
    }

    _logger.info('Page cache hit (stale): $pageId');

    _refreshPageInBackground(pageId, fetchFromNetwork);

    return CacheResult(
      state: CacheState.stale,
      data: data,
      etag: cached.etag,
      fetchedAt: cached.fetchedAt,
      expiresAt: cached.expiresAt,
    );
  }

  /// Get schema with cache-first strategy
  Future<CacheResult<Schema>> getSchema(
    String schemaId, {
    required Future<Schema> Function() fetchFromNetwork,
  }) async {
    final cached = await _database.schemaDao.getSchema(schemaId);

    if (cached == null) {
      _logger.info('Schema cache miss: $schemaId');
      final data = await fetchFromNetwork();
      await _saveSchema(schemaId, data);
      return CacheResult(
        state: CacheState.empty,
        data: data,
      );
    }

    final data = Schema.fromJson(
      jsonDecode(cached.payload) as Map<String, dynamic>,
    );

    if (CachePolicy.isFresh(cached.expiresAt)) {
      _logger.fine('Schema cache hit (fresh): $schemaId');
      return CacheResult(
        state: CacheState.fresh,
        data: data,
        etag: cached.etag,
        fetchedAt: cached.fetchedAt,
        expiresAt: cached.expiresAt,
      );
    }

    _logger.info('Schema cache hit (stale): $schemaId');

    _refreshSchemaInBackground(schemaId, fetchFromNetwork);

    return CacheResult(
      state: CacheState.stale,
      data: data,
      etag: cached.etag,
      fetchedAt: cached.fetchedAt,
      expiresAt: cached.expiresAt,
    );
  }

  /// Save navigation to cache
  Future<void> _saveNavigation(String id, NavigationTree data) async {
    final now = DateTime.now();
    await _database.navigationDao.saveNavigation(
      NavigationCacheCompanion.insert(
        id: id,
        payload: jsonEncode(data.toJson()),
        etag: Value(data.etag),
        version: Value(data.version != null ? int.tryParse(data.version!) : null),
        fetchedAt: now,
        expiresAt: CachePolicy.calculateExpiryTime(CacheType.navigation),
        stale: const Value(false),
      ),
    );
  }

  /// Save page to cache
  Future<void> _savePage(String pageId, PageDescriptor data) async {
    final now = DateTime.now();
    await _database.pageDao.savePage(
      PageCacheCompanion.insert(
        id: pageId,
        pageId: pageId,
        title: data.title,
        payload: jsonEncode(data.toJson()),
        etag: Value(data.etag),
        version: Value(data.version != null ? int.tryParse(data.version!) : null),
        fetchedAt: now,
        expiresAt: CachePolicy.calculateExpiryTime(CacheType.page),
        stale: const Value(false),
      ),
    );
  }

  /// Save schema to cache
  Future<void> _saveSchema(String schemaId, Schema data) async {
    final now = DateTime.now();
    await _database.schemaDao.saveSchema(
      SchemaCacheCompanion.insert(
        id: schemaId,
        schemaId: schemaId,
        payload: jsonEncode(data.toJson()),
        version: Value(data.version != null ? int.tryParse(data.version!) : null),
        fetchedAt: now,
        expiresAt: CachePolicy.calculateExpiryTime(CacheType.schema),
        stale: const Value(false),
      ),
    );
  }

  /// Background refresh for navigation
  Future<void> _refreshNavigationInBackground(
    String id,
    Future<NavigationTree> Function() fetchFromNetwork,
  ) async {
    try {
      final data = await fetchFromNetwork();
      await _saveNavigation(id, data);
      _logger.info('Navigation refreshed in background: $id');
    } catch (e, stack) {
      _logger.warning('Background navigation refresh failed: $id', e, stack);
      // Don't rethrow - background refresh failures are non-critical
    }
  }

  /// Background refresh for page
  Future<void> _refreshPageInBackground(
    String pageId,
    Future<PageDescriptor> Function() fetchFromNetwork,
  ) async {
    try {
      final data = await fetchFromNetwork();
      await _savePage(pageId, data);
      _logger.info('Page refreshed in background: $pageId');
    } catch (e, stack) {
      _logger.warning('Background page refresh failed: $pageId', e, stack);
    }
  }

  /// Background refresh for schema
  Future<void> _refreshSchemaInBackground(
    String schemaId,
    Future<Schema> Function() fetchFromNetwork,
  ) async {
    try {
      final data = await fetchFromNetwork();
      await _saveSchema(schemaId, data);
      _logger.info('Schema refreshed in background: $schemaId');
    } catch (e, stack) {
      _logger.warning('Background schema refresh failed: $schemaId', e, stack);
    }
  }

  /// Invalidate specific cache entry
  Future<void> invalidate(CacheType type, String id) async {
    switch (type) {
      case CacheType.navigation:
        await _database.navigationDao.markStale(id);
      case CacheType.page:
        await _database.pageDao.markStale(id);
      case CacheType.schema:
        await _database.schemaDao.markStale(id);
      case CacheType.permission:
        // Permission cache DAO not yet created
        break;
      case CacheType.uiDecision:
        // UI decision cache DAO not yet created
        break;
      case CacheType.workflow:
        // Workflows don't get invalidated
        break;
    }
  }

  /// Invalidate all cache (e.g., on BFF version change)
  Future<void> invalidateAll() async {
    await _database.markAllStale();
    _logger.info('All caches marked as stale');
  }

  /// Clear all cache (e.g., on logout)
  Future<void> clearAll() async {
    await _database.clearAll();
    _logger.info('All caches cleared');
  }
}
