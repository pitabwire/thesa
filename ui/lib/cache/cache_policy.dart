/// Cache policy configuration for TTL and staleness detection.
///
/// Defines how long each type of cached data stays "fresh"
/// before being considered stale and needing background refresh.
library;

/// Cache policy configuration
class CachePolicy {
  CachePolicy._();

  /// Default TTL for navigation data (15 minutes)
  ///
  /// Menu items change rarely - typically only when new pages are deployed
  static const Duration navigationTtl = Duration(minutes: 15);

  /// Default TTL for page descriptors (10 minutes)
  ///
  /// Page layouts may change with deployments
  static const Duration pageTtl = Duration(minutes: 10);

  /// Default TTL for schemas (30 minutes)
  ///
  /// Schemas are very stable - they only change when data models are modified
  static const Duration schemaTtl = Duration(minutes: 30);

  /// Default TTL for permissions (5 minutes)
  ///
  /// Security-sensitive - access changes should propagate quickly
  static const Duration permissionTtl = Duration(minutes: 5);

  /// Default TTL for UI decisions (10 minutes)
  ///
  /// Matches page descriptors as they're closely related
  static const Duration uiDecisionTtl = Duration(minutes: 10);

  /// Workflow state never expires (persists until explicitly completed/cancelled)
  ///
  /// Users may start a workflow and return days later
  static const Duration? workflowTtl = null;

  /// Check if a cache entry is fresh (within TTL)
  static bool isFresh(DateTime expiresAt) {
    return DateTime.now().isBefore(expiresAt);
  }

  /// Check if a cache entry is stale (past TTL)
  static bool isStale(DateTime expiresAt) {
    return DateTime.now().isAfter(expiresAt) ||
        DateTime.now().isAtSameMomentAs(expiresAt);
  }

  /// Calculate expiry time from now for a given cache type
  static DateTime calculateExpiryTime(CacheType type) {
    final now = DateTime.now();
    switch (type) {
      case CacheType.navigation:
        return now.add(navigationTtl);
      case CacheType.page:
        return now.add(pageTtl);
      case CacheType.schema:
        return now.add(schemaTtl);
      case CacheType.permission:
        return now.add(permissionTtl);
      case CacheType.uiDecision:
        return now.add(uiDecisionTtl);
      case CacheType.workflow:
        // Workflows never expire
        return DateTime(2099);
    }
  }

  /// Get TTL duration for a cache type
  static Duration? getTtl(CacheType type) {
    switch (type) {
      case CacheType.navigation:
        return navigationTtl;
      case CacheType.page:
        return pageTtl;
      case CacheType.schema:
        return schemaTtl;
      case CacheType.permission:
        return permissionTtl;
      case CacheType.uiDecision:
        return uiDecisionTtl;
      case CacheType.workflow:
        return workflowTtl;
    }
  }
}

/// Types of cached data
enum CacheType {
  navigation,
  page,
  schema,
  permission,
  uiDecision,
  workflow,
}

/// Cache state for decision-making
enum CacheState {
  /// No cached data exists
  empty,

  /// Cached data exists and is within TTL
  fresh,

  /// Cached data exists but is past TTL
  stale,
}

/// Result of a cache lookup
class CacheResult<T> {
  const CacheResult({
    required this.state,
    this.data,
    this.etag,
    this.fetchedAt,
    this.expiresAt,
  });

  /// The state of this cache entry
  final CacheState state;

  /// The cached data (if any)
  final T? data;

  /// ETag for cache validation
  final String? etag;

  /// When this data was fetched
  final DateTime? fetchedAt;

  /// When this data expires
  final DateTime? expiresAt;

  /// Whether this entry is fresh
  bool get isFresh => state == CacheState.fresh;

  /// Whether this entry is stale
  bool get isStale => state == CacheState.stale;

  /// Whether this entry is empty
  bool get isEmpty => state == CacheState.empty;

  /// How old this data is
  Duration? get age {
    if (fetchedAt == null) return null;
    return DateTime.now().difference(fetchedAt!);
  }

  /// How much time until expiry (negative if already expired)
  Duration? get timeUntilExpiry {
    if (expiresAt == null) return null;
    return expiresAt!.difference(DateTime.now());
  }
}
