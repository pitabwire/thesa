/// Page cache table for storing page descriptors.
library;

import 'package:drift/drift.dart';

import 'cache_entry.dart';

/// Caches page descriptors from BFF
@DataClassName('PageCacheEntry')
class PageCache extends Table with CacheEntryColumns {
  @override
  String get tableName => 'page_cache';

  @override
  Set<Column> get primaryKey => {id};

  /// Page ID for easy lookup
  TextColumn get pageId => text()();

  /// Page title for display
  TextColumn get title => text()();
}
