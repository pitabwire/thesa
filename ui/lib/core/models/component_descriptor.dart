/// Component descriptors for dynamic UI elements.
///
/// Each component represents a single UI element on a page (table, form, card, etc.).
/// The UI Engine uses the component registry to map component types to widgets.
library;

import 'package:freezed_annotation/freezed_annotation.dart';

import 'action_descriptor.dart';
import 'permission.dart';
import 'schema.dart';
import 'ui_metadata.dart';

part 'component_descriptor.freezed.dart';
part 'component_descriptor.g.dart';

/// Describes a single UI component on a page
@freezed
class ComponentDescriptor with _$ComponentDescriptor {
  const factory ComponentDescriptor({
    /// Component type (determines which widget to use)
    required String type,

    /// Unique identifier for this component instance
    required String id,

    /// Schema reference for this component (if applicable)
    @JsonKey(name: 'schemaRef') String? schemaRef,

    /// Resource this component operates on (e.g., "orders")
    String? resource,

    /// Permission check
    @Default(Permission(allowed: true)) Permission permission,

    /// UI metadata
    UiMetadata? ui,

    /// Component-specific configuration
    @Default({}) Map<String, dynamic> config,

    /// Nested child components (for layout containers)
    List<ComponentDescriptor>? children,

    /// Actions available on this component
    List<ActionDescriptor>? actions,
  }) = _ComponentDescriptor;

  factory ComponentDescriptor.fromJson(Map<String, dynamic> json) =>
      _$ComponentDescriptorFromJson(json);
}

/// Layout configuration for component arrangement
@freezed
class LayoutConfig with _$LayoutConfig {
  const factory LayoutConfig({
    /// Layout type
    required LayoutType type,

    /// Direction (for stack layouts)
    @Default(LayoutDirection.vertical) LayoutDirection direction,

    /// Spacing between components (in pixels)
    @Default(16) double spacing,

    /// Padding around the layout (in pixels)
    @Default(0) double padding,

    /// Number of columns (for grid layouts)
    int? columns,

    /// Minimum column width (for responsive grids)
    double? minColumnWidth,

    /// Maximum column width (for responsive grids)
    double? maxColumnWidth,

    /// Alignment
    LayoutAlignment? alignment,

    /// Whether layout should stretch to fill available space
    @Default(false) bool fill,
  }) = _LayoutConfig;

  factory LayoutConfig.fromJson(Map<String, dynamic> json) =>
      _$LayoutConfigFromJson(json);
}

/// Types of layout containers
enum LayoutType {
  /// Stack components vertically or horizontally
  stack,

  /// Responsive grid
  grid,

  /// Tabbed layout
  tabs,

  /// Split panes
  split,

  /// Scrollable container
  scroll,

  /// Wrap items (flow layout)
  wrap,

  /// Custom layout (requires plugin)
  custom,
}

/// Stack direction
enum LayoutDirection {
  /// Top to bottom
  vertical,

  /// Left to right
  horizontal,
}

/// Alignment options
enum LayoutAlignment {
  /// Align to start (top/left)
  start,

  /// Center alignment
  center,

  /// Align to end (bottom/right)
  end,

  /// Stretch to fill
  stretch,

  /// Space between items
  spaceBetween,

  /// Space around items
  spaceAround,

  /// Space evenly
  spaceEvenly,
}

/// Data table component configuration
@freezed
class DataTableConfig with _$DataTableConfig {
  const factory DataTableConfig({
    /// Table columns
    required List<TableColumn> columns,

    /// Pagination settings
    PaginationConfig? pagination,

    /// Sorting settings
    SortingConfig? sorting,

    /// Filtering settings
    FilterConfig? filter,

    /// Bulk actions
    List<ActionDescriptor>? bulkActions,

    /// Row actions
    List<ActionDescriptor>? rowActions,

    /// Whether rows are selectable
    @Default(false) bool selectable,

    /// Whether to enable virtualization for large datasets
    @Default(true) bool virtualized,

    /// Row height in pixels
    @Default(48.0) double rowHeight,

    /// Whether to show row hover effect
    @Default(true) bool hoverEffect,

    /// Whether to zebra-stripe rows
    @Default(false) bool striped,

    /// Empty state message
    String? emptyMessage,
  }) = _DataTableConfig;

  factory DataTableConfig.fromJson(Map<String, dynamic> json) =>
      _$DataTableConfigFromJson(json);
}

/// Table column definition
@freezed
class TableColumn with _$TableColumn {
  const factory TableColumn({
    /// Field name in the data
    required String field,

    /// Column header label
    required String label,

    /// Whether this column is sortable
    @Default(false) bool sortable,

    /// Whether this column is filterable
    @Default(false) bool filterable,

    /// Priority (1=always visible, 5=hide first on small screens)
    @Default(3) int priority,

    /// Column width (in pixels or flex weight)
    dynamic width,

    /// Minimum width
    double? minWidth,

    /// Maximum width
    double? maxWidth,

    /// Component to render in this column (e.g., "status_badge")
    String? component,

    /// Format hint
    String? format,

    /// Alignment
    @Default(TableColumnAlignment.left) TableColumnAlignment alignment,

    /// Whether this column can be resized
    @Default(true) bool resizable,

    /// Whether this column is pinned (fixed during horizontal scroll)
    @Default(false) bool pinned,

    /// UI metadata
    UiMetadata? ui,
  }) = _TableColumn;

  factory TableColumn.fromJson(Map<String, dynamic> json) =>
      _$TableColumnFromJson(json);
}

/// Column alignment
enum TableColumnAlignment {
  left,
  center,
  right,
}

/// Pagination configuration
@freezed
class PaginationConfig with _$PaginationConfig {
  const factory PaginationConfig({
    /// Pagination type
    @Default(PaginationType.server) PaginationType type,

    /// Default page size
    @Default(25) int defaultPageSize,

    /// Available page sizes
    @Default([10, 25, 50, 100]) List<int> pageSizes,

    /// Whether to show page size selector
    @Default(true) bool showPageSize,

    /// Whether to show total count
    @Default(true) bool showTotal,
  }) = _PaginationConfig;

  factory PaginationConfig.fromJson(Map<String, dynamic> json) =>
      _$PaginationConfigFromJson(json);
}

/// Pagination type
enum PaginationType {
  /// Server-side pagination (preferred for large datasets)
  server,

  /// Client-side pagination (for small datasets)
  client,

  /// Infinite scroll
  infinite,
}

/// Sorting configuration
@freezed
class SortingConfig with _$SortingConfig {
  const factory SortingConfig({
    /// Default sort field
    String? defaultField,

    /// Default sort direction
    @Default(SortDirection.asc) SortDirection defaultDirection,

    /// Whether multi-column sorting is enabled
    @Default(false) bool multiColumn,
  }) = _SortingConfig;

  factory SortingConfig.fromJson(Map<String, dynamic> json) =>
      _$SortingConfigFromJson(json);
}

/// Sort direction
enum SortDirection {
  /// Ascending (A-Z, 0-9)
  asc,

  /// Descending (Z-A, 9-0)
  desc,
}

/// Filter configuration
@freezed
class FilterConfig with _$FilterConfig {
  const factory FilterConfig({
    /// Available filters
    List<FilterDescriptor>? filters,

    /// Whether to show "Clear Filters" button
    @Default(true) bool showClearButton,

    /// Whether filters are collapsible
    @Default(true) bool collapsible,

    /// Whether filters start collapsed
    @Default(false) bool startCollapsed,
  }) = _FilterConfig;

  factory FilterConfig.fromJson(Map<String, dynamic> json) =>
      _$FilterConfigFromJson(json);
}

/// Individual filter descriptor
@freezed
class FilterDescriptor with _$FilterDescriptor {
  const factory FilterDescriptor({
    /// Field to filter on
    required String field,

    /// Filter label
    required String label,

    /// Filter type
    required FilterType type,

    /// Available options (for enum filters)
    List<FieldOption>? options,

    /// Whether multiple values can be selected
    @Default(false) bool multi,

    /// Min/max values (for range filters)
    num? min,
    num? max,

    /// Default value
    dynamic defaultValue,
  }) = _FilterDescriptor;

  factory FilterDescriptor.fromJson(Map<String, dynamic> json) =>
      _$FilterDescriptorFromJson(json);
}

/// Filter types
enum FilterType {
  /// Text search
  text,

  /// Number range
  numberRange,

  /// Date range
  dateRange,

  /// Enum/select
  @JsonValue('enum')
  enumType,

  /// Boolean toggle
  boolean,
}
