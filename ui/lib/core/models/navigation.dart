/// Navigation models for dynamic menu generation.
///
/// The BFF provides the complete navigation tree. The UI renders it as:
/// - Sidebar menu
/// - Breadcrumbs
/// - Dynamic routes
library;

import 'package:freezed_annotation/freezed_annotation.dart';

import 'permission.dart';

part 'navigation.freezed.dart';
part 'navigation.g.dart';

/// Complete navigation tree from BFF
@freezed
class NavigationTree with _$NavigationTree {
  const factory NavigationTree({
    /// Navigation items
    required List<NavigationItem> items,

    /// Version (for cache invalidation)
    String? version,

    /// ETag for cache validation
    String? etag,
  }) = _NavigationTree;

  factory NavigationTree.fromJson(Map<String, dynamic> json) =>
      _$NavigationTreeFromJson(json);
}

/// Single navigation item (can have nested children)
@freezed
class NavigationItem with _$NavigationItem {
  const factory NavigationItem({
    /// Unique identifier
    required String id,

    /// Display label
    required String label,

    /// Navigation path (URL)
    String? path,

    /// Page descriptor ID to load for this route
    String? pageId,

    /// Icon identifier
    String? icon,

    /// Badge text (e.g., "5" for notification count)
    String? badge,

    /// Badge color
    String? badgeColor,

    /// Parent item ID (for hierarchical navigation)
    String? parentId,

    /// Child navigation items
    List<NavigationItem>? children,

    /// Permission check
    @Default(Permission(allowed: true)) Permission permission,

    /// Position in the navigation (for ordering)
    @Default(0) int position,

    /// Whether this item is collapsible
    @Default(false) bool collapsible,

    /// Whether this item starts collapsed
    @Default(false) bool defaultCollapsed,

    /// Group this item belongs to (for visual grouping)
    String? group,

    /// Whether this item should be shown at the bottom (e.g., Settings, Logout)
    @Default(false) bool bottomPosition,

    /// Custom navigation behavior (e.g., "external_link")
    String? behavior,

    /// Target for behavior (e.g., external URL)
    String? target,
  }) = _NavigationItem;

  factory NavigationItem.fromJson(Map<String, dynamic> json) =>
      _$NavigationItemFromJson(json);
}
