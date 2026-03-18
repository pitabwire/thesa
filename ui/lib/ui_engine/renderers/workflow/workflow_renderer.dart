/// Workflow/stepper renderer (placeholder).
///
/// Full workflow implementation uses workflowProvider from state layer.
/// This provides basic stepper visualization.
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../../design/design.dart';
import '../../../widgets/shared/shared.dart';

/// Renders workflow/stepper component (basic version)
class WorkflowRenderer extends StatelessWidget {
  const WorkflowRenderer({
    required this.component,
    this.params = const {},
    super.key,
  });

  final ComponentDescriptor component;
  final Map<String, String> params;

  @override
  Widget build(BuildContext context) {
    final workflowId = component.config['workflowId'] as String?;
    final steps = component.config['steps'] as List<dynamic>? ?? [];

    return AppCard(
      title: component.ui?.label ?? 'Workflow',
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (workflowId != null) ...[
            Text(
              'Workflow ID: $workflowId',
              style: AppTypography.bodySmall,
            ),
            const SizedBox(height: AppSpacing.space8),
          ],
          if (steps.isNotEmpty) ...[
            Text(
              'Steps: ${steps.length}',
              style: AppTypography.bodySmall,
            ),
            const SizedBox(height: AppSpacing.space16),
            // Show step indicators
            Row(
              children: [
                for (var i = 0; i < steps.length; i++) ...[
                  if (i > 0)
                    Expanded(
                      child: Container(
                        height: 2,
                        color: Theme.of(context).colorScheme.outlineVariant,
                      ),
                    ),
                  Container(
                    width: 32,
                    height: 32,
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      color: i == 0
                          ? Theme.of(context).colorScheme.primary
                          : Theme.of(context).colorScheme.surfaceVariant,
                    ),
                    child: Center(
                      child: Text(
                        '${i + 1}',
                        style: TextStyle(
                          color: i == 0
                              ? Theme.of(context).colorScheme.onPrimary
                              : Theme.of(context).colorScheme.onSurfaceVariant,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                    ),
                  ),
                ],
              ],
            ),
            const SizedBox(height: AppSpacing.space16),
          ],
          const Center(
            child: Text(
              'Full workflow implementation uses workflowProvider',
              style: TextStyle(fontStyle: FontStyle.italic),
            ),
          ),
        ],
      ),
    );
  }
}
