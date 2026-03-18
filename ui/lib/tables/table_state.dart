/// Table state management.
///
/// Manages data fetching, pagination, sorting, and filtering for tables.
library;

import 'package:freezed_annotation/freezed_annotation.dart';

import '../core/core.dart';

part 'table_state.freezed.dart';
part 'table_state.g.dart';

/// Table state
@freezed
class TableState with _$TableState {
  const factory TableState({
    /// Current data rows
    @Default([]) List<Map<String, dynamic>> rows,

    /// Total count of rows (for pagination)
    @Default(0) int totalCount,

    /// Current page (0-indexed)
    @Default(0) int currentPage,

    /// Page size
    @Default(25) int pageSize,

    /// Current sort column
    String? sortColumn,

    /// Current sort direction
    @Default(SortDirection.asc) SortDirection sortDirection,

    /// Active filters
    @Default({}) Map<String, dynamic> filters,

    /// Selected row IDs
    @Default({}) Set<String> selectedRows,

    /// Loading state
    @Default(false) bool isLoading,

    /// Error state
    String? error,
  }) = _TableState;

  const TableState._();

  /// Check if all rows are selected
  bool get allRowsSelected =>
      selectedRows.isNotEmpty && selectedRows.length == rows.length;

  /// Check if some (but not all) rows are selected
  bool get someRowsSelected =>
      selectedRows.isNotEmpty && selectedRows.length < rows.length;

  /// Get page count
  int get pageCount => (totalCount / pageSize).ceil();

  /// Check if there's a next page
  bool get hasNextPage => currentPage < pageCount - 1;

  /// Check if there's a previous page
  bool get hasPreviousPage => currentPage > 0;
}
