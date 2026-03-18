/// Schema caching for memoization within a session.
library;

import 'package:logger/logger.dart';

import '../core.dart';

/// Schema cache for memoizing resolved schemas
///
/// Once a schema is resolved (with $ref expansion), the resolved version
/// is cached for the session. This avoids re-resolving the same schema
/// when navigating back to a page.
///
/// Uses schema version/ETag to invalidate memoized schemas when the BFF
/// updates them.
class SchemaCache {
  SchemaCache();

  final Logger _logger = Logger();

  /// Cache of resolved schemas keyed by schema ID
  final Map<String, _CachedSchema> _cache = {};

  /// Get a cached schema if available and not stale
  Schema? get(String schemaId, {String? version}) {
    final cached = _cache[schemaId];

    if (cached == null) {
      _logger.fine('Schema cache miss: $schemaId');
      return null;
    }

    // Check if version matches (if provided)
    if (version != null && cached.version != version) {
      _logger.fine(
        'Schema cache invalidated for $schemaId '
        '(cached: ${cached.version}, requested: $version)',
      );
      _cache.remove(schemaId);
      return null;
    }

    _logger.fine('Schema cache hit: $schemaId');
    return cached.schema;
  }

  /// Put a schema in the cache
  void put(String schemaId, Schema schema, {String? version}) {
    _cache[schemaId] = _CachedSchema(
      schema: schema,
      version: version,
      cachedAt: DateTime.now(),
    );
    _logger.fine('Schema cached: $schemaId (version: $version)');
  }

  /// Invalidate a specific schema
  void invalidate(String schemaId) {
    _cache.remove(schemaId);
    _logger.fine('Schema invalidated: $schemaId');
  }

  /// Invalidate all cached schemas
  void invalidateAll() {
    final count = _cache.length;
    _cache.clear();
    _logger.info('All schemas invalidated (count: $count)');
  }

  /// Get cache statistics
  Map<String, dynamic> getStats() {
    return {
      'total_cached': _cache.length,
      'cache_keys': _cache.keys.toList(),
    };
  }
}

/// Internal cached schema wrapper
class _CachedSchema {
  _CachedSchema({
    required this.schema,
    required this.version,
    required this.cachedAt,
  });

  final Schema schema;
  final String? version;
  final DateTime cachedAt;
}
