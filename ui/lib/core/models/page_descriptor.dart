/// Page descriptors for complete page definitions.
///
/// A page descriptor contains everything needed to render a page:
/// - Layout configuration
/// - List of components to render
/// - Page-level actions
/// - Metadata (title, breadcrumbs, etc.)
library;

import 'package:freezed_annotation/freezed_annotation.dart';

import 'action_descriptor.dart';
import 'component_descriptor.dart';
import 'permission.dart';
import 'ui_metadata.dart';

part 'page_descriptor.freezed.dart';
part 'page_descriptor.g.dart';

/// Complete description of a page and its contents
@freezed
class PageDescriptor with _$PageDescriptor {
  const factory PageDescriptor({
    /// Unique identifier for this page
    required String pageId,

    /// Page title (shown in header and breadcrumbs)
    required String title,

    /// Page subtitle
    String? subtitle,

    /// Layout configuration
    LayoutConfig? layout,

    /// Components to render on this page
    @Default([]) List<ComponentDescriptor> components,

    /// Page-level actions (e.g., "Create New")
    @Default([]) List<ActionDescriptor> actions,

    /// Permission check for this page
    @Default(Permission(allowed: true)) Permission permission,

    /// UI metadata
    UiMetadata? ui,

    /// Breadcrumb items
    List<BreadcrumbItem>? breadcrumbs,

    /// Page type hint
    PageType? pageType,

    /// Whether this page supports full-screen mode
    @Default(false) bool fullscreenable,

    /// Whether this page should refresh automatically
    RefreshConfig? autoRefresh,

    /// Custom metadata
    Map<String, dynamic>? metadata,

    /// Version of this descriptor (for cache invalidation)
    String? version,

    /// ETag for cache validation
    String? etag,
  }) = _PageDescriptor;

  factory PageDescriptor.fromJson(Map<String, dynamic> json) =>
      _$PageDescriptorFromJson(json);
}

/// Breadcrumb navigation item
@freezed
class BreadcrumbItem with _$BreadcrumbItem {
  const factory BreadcrumbItem({
    /// Label text
    required String label,

    /// Navigation path
    String? path,

    /// Icon
    String? icon,
  }) = _BreadcrumbItem;

  factory BreadcrumbItem.fromJson(Map<String, dynamic> json) =>
      _$BreadcrumbItemFromJson(json);
}

/// Page type hints for optimized rendering
enum PageType {
  /// List of resources with table
  resourceList,

  /// Detail view of a single resource
  resourceDetail,

  /// Dashboard with metrics and charts
  dashboard,

  /// Form for creating/editing
  form,

  /// Workflow/wizard
  workflow,

  /// Custom page type
  custom,
}

/// Auto-refresh configuration
@freezed
class RefreshConfig with _$RefreshConfig {
  const factory RefreshConfig({
    /// Whether auto-refresh is enabled
    @Default(false) bool enabled,

    /// Refresh interval in seconds
    @Default(60) int intervalSeconds,

    /// Whether to show a refresh indicator
    @Default(false) bool showIndicator,

    /// Whether to pause refresh when page is not visible
    @Default(true) bool pauseWhenHidden,
  }) = _RefreshConfig;

  factory RefreshConfig.fromJson(Map<String, dynamic> json) =>
      _$RefreshConfigFromJson(json);
}
