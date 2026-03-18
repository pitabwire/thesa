/// Navigation cache table for storing the sidebar menu tree.
library;

import 'package:drift/drift.dart';

import 'cache_entry.dart';

/// Caches navigation trees from BFF
@DataClassName('NavigationCacheEntry')
class NavigationCache extends Table with CacheEntryColumns {
  @override
  String get tableName => 'navigation_cache';

  @override
  Set<Column> get primaryKey => {id};
}
