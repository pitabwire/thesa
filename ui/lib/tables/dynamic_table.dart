/// Dynamic table widget with pagination, sorting, and filtering.
library;

import 'package:data_table_2/data_table_2.dart';
import 'package:flutter/material.dart';

import '../core/core.dart';
import '../design/design.dart';
import '../widgets/shared/shared.dart';
import 'table_controller.dart';

/// Dynamic table that renders from table config
class DynamicTable extends StatefulWidget {
  const DynamicTable({
    required this.controller,
    this.onRowTap,
    this.onRowAction,
    this.onBulkAction,
    super.key,
  });

  final TableController controller;
  final void Function(Map<String, dynamic> row)? onRowTap;
  final void Function(ActionDescriptor action, Map<String, dynamic> row)?
      onRowAction;
  final void Function(ActionDescriptor action, Set<String> rowIds)?
      onBulkAction;

  @override
  State<DynamicTable> createState() => _DynamicTableState();
}

class _DynamicTableState extends State<DynamicTable> {
  @override
  void initState() {
    super.initState();
    widget.controller.addListener(_onControllerUpdate);
    // Load initial data
    WidgetsBinding.instance.addPostFrameCallback((_) {
      widget.controller.loadData();
    });
  }

  @override
  void dispose() {
    widget.controller.removeListener(_onControllerUpdate);
    super.dispose();
  }

  void _onControllerUpdate() {
    setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    final state = widget.controller.state;
    final config = widget.controller.tableConfig;

    if (state.error != null) {
      return AppErrorWidget(
        error: state.error!,
        onRetry: widget.controller.refresh,
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        // Bulk actions bar (shown when rows selected)
        if (state.selectedRows.isNotEmpty) ...[
          _buildBulkActionsBar(context),
          const SizedBox(height: AppSpacing.space8),
        ],

        // Table
        Expanded(
          child: Card(
            child: Column(
              children: [
                // Table content
                Expanded(
                  child: state.isLoading && state.rows.isEmpty
                      ? const Center(child: AppLoadingIndicator())
                      : state.rows.isEmpty
                          ? Center(
                              child: AppEmptyState(
                                message: config.emptyMessage ?? 'No data available',
                                icon: Icons.table_chart,
                              ),
                            )
                          : _buildDataTable(context),
                ),

                // Pagination
                if (config.pagination != null) ...[
                  const Divider(height: 1),
                  _buildPagination(context),
                ],
              ],
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildDataTable(BuildContext context) {
    final state = widget.controller.state;
    final config = widget.controller.tableConfig;

    return DataTable2(
      columnSpacing: 12,
      horizontalMargin: 12,
      minWidth: 600,
      isHorizontalScrollBarVisible: true,
      isVerticalScrollBarVisible: true,
      fixedTopRows: 1,
      columns: _buildColumns(context),
      rows: _buildRows(context),
      headingRowColor: MaterialStateProperty.all(
        Theme.of(context).colorScheme.surfaceVariant.withOpacity(0.5),
      ),
      headingRowHeight: 56,
      dataRowHeight: config.rowHeight,
      showCheckboxColumn: config.selectable,
    );
  }

  List<DataColumn2> _buildColumns(BuildContext context) {
    final config = widget.controller.tableConfig;
    final state = widget.controller.state;
    final columns = <DataColumn2>[];

    for (final column in config.columns) {
      columns.add(
        DataColumn2(
          label: Text(
            column.label,
            style: AppTypography.labelMedium.copyWith(
              fontWeight: FontWeight.bold,
            ),
          ),
          size: _getColumnSize(column),
          onSort: column.sortable
              ? (columnIndex, ascending) {
                  widget.controller.sortBy(column.field);
                }
              : null,
          numeric: column.alignment == TableColumnAlignment.right,
        ),
      );
    }

    // Add row actions column if configured
    if (config.rowActions != null && config.rowActions!.isNotEmpty) {
      columns.add(
        DataColumn2(
          label: const Text('Actions'),
          size: ColumnSize.S,
          fixedWidth: 100,
        ),
      );
    }

    return columns;
  }

  ColumnSize _getColumnSize(TableColumn column) {
    if (column.width != null) {
      if (column.width is String) {
        final widthStr = column.width.toString().toLowerCase();
        if (widthStr == 'l') return ColumnSize.L;
        if (widthStr == 'm') return ColumnSize.M;
        if (widthStr == 's') return ColumnSize.S;
      } else if (column.width is int) {
        final widthNum = column.width as int;
        if (widthNum > 200) return ColumnSize.L;
        if (widthNum > 100) return ColumnSize.M;
        return ColumnSize.S;
      }
    }
    return ColumnSize.M;
  }

  List<DataRow2> _buildRows(BuildContext context) {
    final state = widget.controller.state;
    final config = widget.controller.tableConfig;

    return state.rows.map((row) {
      final rowId = row['id']?.toString() ?? '';
      final isSelected = state.selectedRows.contains(rowId);

      return DataRow2(
        selected: isSelected,
        onSelectChanged: config.selectable
            ? (_) => widget.controller.toggleRowSelection(rowId)
            : null,
        onTap: widget.onRowTap != null ? () => widget.onRowTap!(row) : null,
        cells: [
          ...config.columns.map((column) {
            return DataCell(_buildCell(context, row, column));
          }),
          // Row actions
          if (config.rowActions != null && config.rowActions!.isNotEmpty)
            DataCell(_buildRowActions(context, row)),
        ],
      );
    }).toList();
  }

  Widget _buildCell(
    BuildContext context,
    Map<String, dynamic> row,
    TableColumn column,
  ) {
    final value = row[column.field];

    // Use custom component renderer if specified
    if (column.component != null) {
      return _buildComponentCell(context, value, column.component!);
    }

    // Default text rendering
    return Text(
      _formatValue(value, column.format),
      style: AppTypography.bodyMedium,
      overflow: TextOverflow.ellipsis,
    );
  }

  Widget _buildComponentCell(
    BuildContext context,
    dynamic value,
    String componentType,
  ) {
    switch (componentType.toLowerCase()) {
      case 'badge':
      case 'status_badge':
        return AppBadge(
          label: value?.toString() ?? '',
          variant: _getBadgeVariant(value?.toString() ?? ''),
        );

      case 'avatar':
        return AppAvatar(
          name: value?.toString() ?? '',
          size: 32,
        );

      default:
        return Text(value?.toString() ?? '');
    }
  }

  AppBadgeVariant _getBadgeVariant(String status) {
    switch (status.toLowerCase()) {
      case 'active':
      case 'completed':
      case 'success':
        return AppBadgeVariant.success;
      case 'pending':
      case 'in_progress':
        return AppBadgeVariant.warning;
      case 'failed':
      case 'error':
      case 'cancelled':
        return AppBadgeVariant.error;
      case 'draft':
        return AppBadgeVariant.info;
      default:
        return AppBadgeVariant.neutral;
    }
  }

  String _formatValue(dynamic value, String? format) {
    if (value == null) return '-';

    // TODO: Implement formatting based on format hint
    // (currency, date, number, etc.)

    return value.toString();
  }

  Widget _buildRowActions(BuildContext context, Map<String, dynamic> row) {
    final config = widget.controller.tableConfig;
    final actions = config.rowActions!
        .where((action) => action.permission.allowed)
        .toList();

    if (actions.isEmpty) {
      return const SizedBox.shrink();
    }

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: actions.map((action) {
        return IconButton(
          icon: Icon(_getActionIcon(action.icon)),
          iconSize: 20,
          tooltip: action.label,
          onPressed: () {
            widget.onRowAction?.call(action, row);
          },
        );
      }).toList(),
    );
  }

  IconData _getActionIcon(String? icon) {
    if (icon == null) return Icons.more_vert;

    switch (icon.toLowerCase()) {
      case 'edit':
        return Icons.edit;
      case 'delete':
        return Icons.delete;
      case 'view':
        return Icons.visibility;
      case 'download':
        return Icons.download;
      default:
        return Icons.circle;
    }
  }

  Widget _buildBulkActionsBar(BuildContext context) {
    final state = widget.controller.state;
    final config = widget.controller.tableConfig;
    final bulkActions = config.bulkActions
            ?.where((action) => action.permission.allowed)
            .toList() ??
        [];

    return Card(
      color: Theme.of(context).colorScheme.primaryContainer,
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.space12),
        child: Row(
          children: [
            Text(
              '${state.selectedRows.length} selected',
              style: AppTypography.labelMedium.copyWith(
                color: Theme.of(context).colorScheme.onPrimaryContainer,
              ),
            ),
            const Spacer(),
            ...bulkActions.map((action) {
              return Padding(
                padding: const EdgeInsets.only(left: AppSpacing.space8),
                child: AppButton(
                  label: action.label,
                  onPressed: () {
                    widget.onBulkAction?.call(action, state.selectedRows);
                  },
                  variant: AppButtonVariant.secondary,
                  size: AppButtonSize.small,
                ),
              );
            }),
            const SizedBox(width: AppSpacing.space8),
            AppButton(
              label: 'Clear',
              onPressed: widget.controller.deselectAllRows,
              variant: AppButtonVariant.tertiary,
              size: AppButtonSize.small,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildPagination(BuildContext context) {
    final state = widget.controller.state;
    final config = widget.controller.tableConfig;
    final pagination = config.pagination!;

    return Padding(
      padding: const EdgeInsets.all(AppSpacing.space12),
      child: Row(
        children: [
          // Page size selector
          if (pagination.showPageSize) ...[
            Text(
              'Rows per page:',
              style: AppTypography.bodySmall,
            ),
            const SizedBox(width: AppSpacing.space8),
            DropdownButton<int>(
              value: state.pageSize,
              items: pagination.pageSizes.map((size) {
                return DropdownMenuItem(
                  value: size,
                  child: Text(size.toString()),
                );
              }).toList(),
              onChanged: (value) {
                if (value != null) {
                  widget.controller.changePageSize(value);
                }
              },
              underline: const SizedBox.shrink(),
            ),
          ],

          const Spacer(),

          // Total count
          if (pagination.showTotal)
            Text(
              '${state.currentPage * state.pageSize + 1}-${((state.currentPage + 1) * state.pageSize).clamp(0, state.totalCount)} of ${state.totalCount}',
              style: AppTypography.bodySmall,
            ),

          const SizedBox(width: AppSpacing.space16),

          // Navigation buttons
          IconButton(
            icon: const Icon(Icons.chevron_left),
            onPressed: state.hasPreviousPage
                ? widget.controller.previousPage
                : null,
            tooltip: 'Previous page',
          ),
          IconButton(
            icon: const Icon(Icons.chevron_right),
            onPressed: state.hasNextPage ? widget.controller.nextPage : null,
            tooltip: 'Next page',
          ),
        ],
      ),
    );
  }
}
