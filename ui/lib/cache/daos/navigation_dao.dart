/// Data Access Object for navigation cache operations.
library;

import 'package:drift/drift.dart';

import '../database/app_database.dart';
import '../database/tables/navigation_cache.dart';

part 'navigation_dao.g.dart';

/// DAO for navigation cache operations
@DriftAccessor(tables: [NavigationCache])
class NavigationDao extends DatabaseAccessor<AppDatabase>
    with _$NavigationDaoMixin {
  /// Creates a new NavigationDao
  NavigationDao(super.db);

  /// Watch the navigation tree (reactive stream)
  Stream<NavigationCacheEntry?> watchNavigation(String id) {
    return (select(navigationCache)..where((t) => t.id.equals(id)))
        .watchSingleOrNull();
  }

  /// Get navigation tree (one-time read)
  Future<NavigationCacheEntry?> getNavigation(String id) {
    return (select(navigationCache)..where((t) => t.id.equals(id)))
        .getSingleOrNull();
  }

  /// Save navigation tree to cache
  Future<void> saveNavigation(NavigationCacheCompanion entry) {
    return into(navigationCache).insertOnConflictUpdate(entry);
  }

  /// Update navigation ETag and refresh timestamp
  Future<void> touchNavigation(String id, {String? etag}) {
    return (update(navigationCache)..where((t) => t.id.equals(id))).write(
      NavigationCacheCompanion(
        fetchedAt: Value(DateTime.now()),
        expiresAt: Value(
          DateTime.now().add(const Duration(minutes: 15)), // TTL: 15 minutes
        ),
        stale: const Value(false),
        etag: etag != null ? Value(etag) : const Value.absent(),
      ),
    );
  }

  /// Mark navigation as stale
  Future<void> markStale(String id) {
    return (update(navigationCache)..where((t) => t.id.equals(id))).write(
      const NavigationCacheCompanion(stale: Value(true)),
    );
  }

  /// Delete navigation entry
  Future<void> deleteNavigation(String id) {
    return (delete(navigationCache)..where((t) => t.id.equals(id))).go();
  }

  /// Clear all navigation cache
  Future<void> clearAll() {
    return delete(navigationCache).go();
  }
}
