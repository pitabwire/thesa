/// Data Access Object for schema cache operations.
///
/// Schemas use reference counting to prevent premature eviction.
library;

import 'package:drift/drift.dart';

import '../database/app_database.dart';
import '../database/tables/schema_cache.dart';

part 'schema_dao.g.dart';

/// DAO for schema cache operations
@DriftAccessor(tables: [SchemaCache])
class SchemaDao extends DatabaseAccessor<AppDatabase> with _$SchemaDaoMixin {
  /// Creates a new SchemaDao
  SchemaDao(super.db);

  /// Watch a specific schema (reactive stream)
  Stream<SchemaCacheEntry?> watchSchema(String schemaId) {
    return (select(schemaCache)..where((t) => t.schemaId.equals(schemaId)))
        .watchSingleOrNull();
  }

  /// Get a specific schema (one-time read)
  Future<SchemaCacheEntry?> getSchema(String schemaId) {
    return (select(schemaCache)..where((t) => t.schemaId.equals(schemaId)))
        .getSingleOrNull();
  }

  /// Save schema to cache
  Future<void> saveSchema(SchemaCacheCompanion entry) {
    return into(schemaCache).insertOnConflictUpdate(entry);
  }

  /// Update schema ETag and refresh timestamp
  Future<void> touchSchema(String schemaId, {String? etag}) {
    return (update(schemaCache)..where((t) => t.schemaId.equals(schemaId)))
        .write(
      SchemaCacheCompanion(
        fetchedAt: Value(DateTime.now()),
        expiresAt: Value(
          DateTime.now().add(const Duration(minutes: 30)), // TTL: 30 minutes
        ),
        stale: const Value(false),
        etag: etag != null ? Value(etag) : const Value.absent(),
      ),
    );
  }

  /// Increment reference count for a schema
  Future<void> incrementRefCount(String schemaId) async {
    final schema = await getSchema(schemaId);
    if (schema != null) {
      await (update(schemaCache)..where((t) => t.schemaId.equals(schemaId)))
          .write(
        SchemaCacheCompanion(
          refCount: Value(schema.refCount + 1),
        ),
      );
    }
  }

  /// Decrement reference count for a schema
  Future<void> decrementRefCount(String schemaId) async {
    final schema = await getSchema(schemaId);
    if (schema != null && schema.refCount > 0) {
      await (update(schemaCache)..where((t) => t.schemaId.equals(schemaId)))
          .write(
        SchemaCacheCompanion(
          refCount: Value(schema.refCount - 1),
        ),
      );
    }
  }

  /// Mark schema as stale
  Future<void> markStale(String schemaId) {
    return (update(schemaCache)..where((t) => t.schemaId.equals(schemaId)))
        .write(
      const SchemaCacheCompanion(stale: Value(true)),
    );
  }

  /// Delete schema entry (only if ref count is 0)
  Future<bool> deleteSchema(String schemaId) async {
    final schema = await getSchema(schemaId);
    if (schema != null && schema.refCount == 0) {
      await (delete(schemaCache)..where((t) => t.schemaId.equals(schemaId)))
          .go();
      return true;
    }
    return false;
  }

  /// Clear all schemas (ignores ref count - use with caution)
  Future<void> clearAll() {
    return delete(schemaCache).go();
  }

  /// Get schemas with zero references (candidates for cleanup)
  Future<List<SchemaCacheEntry>> getUnreferencedSchemas() {
    return (select(schemaCache)..where((t) => t.refCount.equals(0))).get();
  }
}
