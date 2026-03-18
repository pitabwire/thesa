/// Schema cache table for storing data structure definitions.
library;

import 'package:drift/drift.dart';

import 'cache_entry.dart';

/// Caches schemas from BFF
@DataClassName('SchemaCacheEntry')
class SchemaCache extends Table with CacheEntryColumns {
  @override
  String get tableName => 'schema_cache';

  @override
  Set<Column> get primaryKey => {id};

  /// Schema ID for easy lookup
  TextColumn get schemaId => text()();

  /// Reference count (how many pages currently use this schema)
  IntColumn get refCount => integer().withDefault(const Constant(0))();
}
