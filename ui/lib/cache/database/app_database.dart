/// Main Drift database definition for Thesa UI.
///
/// This database implements the offline-first cache architecture with:
/// - Stale-while-revalidate strategy
/// - ETag-based cache validation
/// - TTL-based expiry
/// - Reference counting for shared resources
library;

import 'dart:io';

import 'package:drift/drift.dart';
import 'package:drift/native.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';

import '../daos/navigation_dao.dart';
import '../daos/page_dao.dart';
import '../daos/schema_dao.dart';
import '../daos/workflow_dao.dart';
import 'tables/navigation_cache.dart';
import 'tables/page_cache.dart';
import 'tables/permission_cache.dart';
import 'tables/schema_cache.dart';
import 'tables/ui_decision_cache.dart';
import 'tables/workflow_state.dart';

part 'app_database.g.dart';

/// The main database for Thesa UI
@DriftDatabase(
  tables: [
    NavigationCache,
    PageCache,
    SchemaCache,
    PermissionCache,
    UiDecisionCache,
    WorkflowState,
  ],
  daos: [
    NavigationDao,
    PageDao,
    SchemaDao,
    WorkflowDao,
  ],
)
class AppDatabase extends _$AppDatabase {
  /// Creates the database with the given executor
  AppDatabase(super.e);

  /// Current database schema version
  @override
  int get schemaVersion => 1;

  /// Database migrations
  @override
  MigrationStrategy get migration {
    return MigrationStrategy(
      onCreate: (Migrator m) async {
        // Create all tables
        await m.createAll();
      },
      onUpgrade: (Migrator m, int from, int to) async {
        // Future migrations will go here
        // Example:
        // if (from < 2) {
        //   await m.addColumn(schemaCache, schemaCache.refCount);
        // }
      },
    );
  }

  /// Clear all cache (typically called on logout)
  Future<void> clearAll() async {
    await transaction(() async {
      await delete(navigationCache).go();
      await delete(pageCache).go();
      await delete(schemaCache).go();
      await delete(permissionCache).go();
      await delete(uiDecisionCache).go();
      // Note: We don't clear workflow_state on logout,
      // as workflows should persist
    });
  }

  /// Mark all cache entries as stale (typically called when BFF version changes)
  Future<void> markAllStale() async {
    await transaction(() async {
      await update(navigationCache).write(
        const NavigationCacheCompanion(stale: Value(true)),
      );
      await update(pageCache).write(
        const PageCacheCompanion(stale: Value(true)),
      );
      await update(schemaCache).write(
        const SchemaCacheCompanion(stale: Value(true)),
      );
      await update(permissionCache).write(
        const PermissionCacheCompanion(stale: Value(true)),
      );
      await update(uiDecisionCache).write(
        const UiDecisionCacheCompanion(stale: Value(true)),
      );
    });
  }

  /// Get database size for monitoring
  Future<int> getDatabaseSize() async {
    final dbFolder = await getApplicationDocumentsDirectory();
    final file = File(p.join(dbFolder.path, 'thesa_ui.db'));
    if (await file.exists()) {
      return await file.length();
    }
    return 0;
  }
}

/// Create the database instance
///
/// This is the factory function used throughout the app
Future<AppDatabase> createDatabase() async {
  final dbFolder = await getApplicationDocumentsDirectory();
  final file = File(p.join(dbFolder.path, 'thesa_ui.db'));
  return AppDatabase(NativeDatabase(file));
}
