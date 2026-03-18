/// Base cache entry structure used by all cache tables.
///
/// All cache tables follow the same pattern:
/// - Primary key for lookups
/// - JSON payload for the actual data
/// - ETag for HTTP cache validation
/// - Timestamps for TTL management
/// - Stale flag for visual indicators
library;

import 'package:drift/drift.dart';

/// Base mixin for all cache tables
mixin CacheEntryColumns on Table {
  /// Unique identifier for this cache entry
  TextColumn get id => text()();

  /// The actual data, stored as JSON
  TextColumn get payload => text()();

  /// ETag from the server for cache validation
  TextColumn get etag => text().nullable()();

  /// Schema/API version number
  IntColumn get version => integer().nullable()();

  /// When this data was last fetched from the server
  DateTimeColumn get fetchedAt => dateTime()();

  /// When this data expires (TTL expiry)
  DateTimeColumn get expiresAt => dateTime()();

  /// Whether this data is known to be stale
  BoolColumn get stale => boolean().withDefault(const Constant(false))();
}
