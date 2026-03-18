/// Permission cache table for storing user capabilities and permissions.
library;

import 'package:drift/drift.dart';

import 'cache_entry.dart';

/// Caches permissions and capabilities from BFF
@DataClassName('PermissionCacheEntry')
class PermissionCache extends Table with CacheEntryColumns {
  @override
  String get tableName => 'permission_cache';

  @override
  Set<Column> get primaryKey => {id};

  /// User ID this permission set belongs to
  TextColumn get userId => text()();

  /// Tenant/organization ID
  TextColumn get tenantId => text().nullable()();
}
