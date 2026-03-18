/// Table controller for managing table state and data.
library;

import 'package:flutter/foundation.dart';

import '../core/core.dart';
import 'table_state.dart';

/// Controller for managing table data and state
class TableController extends ChangeNotifier {
  TableController({
    required this.tableConfig,
    required this.fetchData,
    int initialPageSize = 25,
  }) {
    _state = TableState(
      pageSize: tableConfig.pagination?.defaultPageSize ?? initialPageSize,
      sortColumn: tableConfig.sorting?.defaultField,
      sortDirection: tableConfig.sorting?.defaultDirection ?? SortDirection.asc,
    );
  }

  /// Table configuration from BFF
  final DataTableConfig tableConfig;

  /// Function to fetch data from BFF
  final Future<TableDataResponse> Function(TableDataRequest request) fetchData;

  /// Current state
  late TableState _state;
  TableState get state => _state;

  /// Load data
  Future<void> loadData() async {
    _updateState(_state.copyWith(isLoading: true, error: null));

    try {
      final request = TableDataRequest(
        page: _state.currentPage,
        pageSize: _state.pageSize,
        sortColumn: _state.sortColumn,
        sortDirection: _state.sortDirection,
        filters: _state.filters,
      );

      final response = await fetchData(request);

      _updateState(_state.copyWith(
        rows: response.rows,
        totalCount: response.totalCount,
        isLoading: false,
      ));
    } catch (error) {
      _updateState(_state.copyWith(
        isLoading: false,
        error: error.toString(),
      ));
    }
  }

  /// Change page
  Future<void> goToPage(int page) async {
    if (page == _state.currentPage) return;
    _updateState(_state.copyWith(currentPage: page));
    await loadData();
  }

  /// Next page
  Future<void> nextPage() async {
    if (_state.hasNextPage) {
      await goToPage(_state.currentPage + 1);
    }
  }

  /// Previous page
  Future<void> previousPage() async {
    if (_state.hasPreviousPage) {
      await goToPage(_state.currentPage - 1);
    }
  }

  /// Change page size
  Future<void> changePageSize(int pageSize) async {
    _updateState(_state.copyWith(
      pageSize: pageSize,
      currentPage: 0, // Reset to first page
    ));
    await loadData();
  }

  /// Sort by column
  Future<void> sortBy(String column) async {
    SortDirection direction = SortDirection.asc;

    // Toggle direction if clicking same column
    if (_state.sortColumn == column) {
      direction = _state.sortDirection == SortDirection.asc
          ? SortDirection.desc
          : SortDirection.asc;
    }

    _updateState(_state.copyWith(
      sortColumn: column,
      sortDirection: direction,
    ));

    await loadData();
  }

  /// Apply filters
  Future<void> applyFilters(Map<String, dynamic> filters) async {
    _updateState(_state.copyWith(
      filters: filters,
      currentPage: 0, // Reset to first page
    ));
    await loadData();
  }

  /// Clear filters
  Future<void> clearFilters() async {
    await applyFilters({});
  }

  /// Select row
  void selectRow(String rowId) {
    final newSelection = Set<String>.from(_state.selectedRows)..add(rowId);
    _updateState(_state.copyWith(selectedRows: newSelection));
  }

  /// Deselect row
  void deselectRow(String rowId) {
    final newSelection = Set<String>.from(_state.selectedRows)..remove(rowId);
    _updateState(_state.copyWith(selectedRows: newSelection));
  }

  /// Toggle row selection
  void toggleRowSelection(String rowId) {
    if (_state.selectedRows.contains(rowId)) {
      deselectRow(rowId);
    } else {
      selectRow(rowId);
    }
  }

  /// Select all rows
  void selectAllRows() {
    final allIds = _state.rows
        .map((row) => row['id']?.toString() ?? '')
        .where((id) => id.isNotEmpty)
        .toSet();
    _updateState(_state.copyWith(selectedRows: allIds));
  }

  /// Deselect all rows
  void deselectAllRows() {
    _updateState(_state.copyWith(selectedRows: {}));
  }

  /// Toggle all rows selection
  void toggleAllRows() {
    if (_state.allRowsSelected) {
      deselectAllRows();
    } else {
      selectAllRows();
    }
  }

  /// Refresh data
  Future<void> refresh() async {
    await loadData();
  }

  void _updateState(TableState newState) {
    _state = newState;
    notifyListeners();
  }

  @override
  void dispose() {
    super.dispose();
  }
}

/// Request for fetching table data
class TableDataRequest {
  const TableDataRequest({
    required this.page,
    required this.pageSize,
    this.sortColumn,
    this.sortDirection = SortDirection.asc,
    this.filters = const {},
  });

  final int page;
  final int pageSize;
  final String? sortColumn;
  final SortDirection sortDirection;
  final Map<String, dynamic> filters;
}

/// Response from BFF with table data
class TableDataResponse {
  const TableDataResponse({
    required this.rows,
    required this.totalCount,
  });

  final List<Map<String, dynamic>> rows;
  final int totalCount;
}
