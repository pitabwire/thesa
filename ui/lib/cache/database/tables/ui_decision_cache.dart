/// UI decision cache for storing server-driven UI decisions.
///
/// These are server decisions about what UI to show based on:
/// - User role/permissions
/// - Tenant configuration
/// - Feature flags
/// - A/B tests
library;

import 'package:drift/drift.dart';

import 'cache_entry.dart';

/// Caches UI decisions from BFF
@DataClassName('UiDecisionCacheEntry')
class UiDecisionCache extends Table with CacheEntryColumns {
  @override
  String get tableName => 'ui_decision_cache';

  @override
  Set<Column> get primaryKey => {id};

  /// Decision key (e.g., "dashboard_layout", "orders_view_mode")
  TextColumn get decisionKey => text()();

  /// Context that influenced this decision (user role, tenant, etc.)
  TextColumn get context => text().nullable()();
}
