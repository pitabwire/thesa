/// Data table renderer (placeholder).
///
/// Full implementation will be in Task 11 (Implement dynamic table engine).
/// For now, provides basic table rendering with columns from BFF descriptor.
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../../design/design.dart';
import '../../../widgets/shared/shared.dart';

/// Renders data table component (basic version)
class DataTableRenderer extends StatelessWidget {
  const DataTableRenderer({
    required this.component,
    this.params = const {},
    super.key,
  });

  final ComponentDescriptor component;
  final Map<String, String> params;

  @override
  Widget build(BuildContext context) {
    // Parse table config
    DataTableConfig? tableConfig;
    if (component.config['table'] != null) {
      try {
        tableConfig = DataTableConfig.fromJson(
          component.config['table'] as Map<String, dynamic>,
        );
      } catch (e) {
        // Invalid config
      }
    }

    if (tableConfig == null) {
      return AppCard(
        title: 'Data Table: ${component.id}',
        body: const Text('Table configuration missing or invalid'),
      );
    }

    // TODO: Load actual data from BFF
    // For now, show placeholder with column headers
    return AppCard(
      title: component.ui?.label ?? 'Data Table',
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'Columns: ${tableConfig.columns.map((c) => c.label).join(", ")}',
            style: AppTypography.bodySmall,
          ),
          const SizedBox(height: AppSpacing.space8),
          Text(
            'Pagination: ${tableConfig.pagination?.defaultPageSize ?? "Not configured"}',
            style: AppTypography.bodySmall,
          ),
          const SizedBox(height: AppSpacing.space16),
          const Center(
            child: Text(
              'Full table implementation in Task 11',
              style: TextStyle(fontStyle: FontStyle.italic),
            ),
          ),
        ],
      ),
    );
  }
}
