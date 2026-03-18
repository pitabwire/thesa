/// Data Access Object for page cache operations.
library;

import 'package:drift/drift.dart';

import '../database/app_database.dart';
import '../database/tables/page_cache.dart';

part 'page_dao.g.dart';

/// DAO for page cache operations
@DriftAccessor(tables: [PageCache])
class PageDao extends DatabaseAccessor<AppDatabase> with _$PageDaoMixin {
  /// Creates a new PageDao
  PageDao(super.db);

  /// Watch a specific page (reactive stream)
  Stream<PageCacheEntry?> watchPage(String pageId) {
    return (select(pageCache)..where((t) => t.pageId.equals(pageId)))
        .watchSingleOrNull();
  }

  /// Get a specific page (one-time read)
  Future<PageCacheEntry?> getPage(String pageId) {
    return (select(pageCache)..where((t) => t.pageId.equals(pageId)))
        .getSingleOrNull();
  }

  /// Save page to cache
  Future<void> savePage(PageCacheCompanion entry) {
    return into(pageCache).insertOnConflictUpdate(entry);
  }

  /// Update page ETag and refresh timestamp
  Future<void> touchPage(String pageId, {String? etag}) {
    return (update(pageCache)..where((t) => t.pageId.equals(pageId))).write(
      PageCacheCompanion(
        fetchedAt: Value(DateTime.now()),
        expiresAt: Value(
          DateTime.now().add(const Duration(minutes: 10)), // TTL: 10 minutes
        ),
        stale: const Value(false),
        etag: etag != null ? Value(etag) : const Value.absent(),
      ),
    );
  }

  /// Mark page as stale
  Future<void> markStale(String pageId) {
    return (update(pageCache)..where((t) => t.pageId.equals(pageId))).write(
      const PageCacheCompanion(stale: Value(true)),
    );
  }

  /// Delete page entry
  Future<void> deletePage(String pageId) {
    return (delete(pageCache)..where((t) => t.pageId.equals(pageId))).go();
  }

  /// Delete all pages
  Future<void> clearAll() {
    return delete(pageCache).go();
  }

  /// Get all cached page IDs
  Future<List<String>> getAllPageIds() async {
    final pages = await select(pageCache).get();
    return pages.map((p) => p.pageId).toList();
  }
}
