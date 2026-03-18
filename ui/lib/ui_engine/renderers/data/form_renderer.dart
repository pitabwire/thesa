/// Form renderer (placeholder).
///
/// Full implementation will be in Task 10 (Implement dynamic form engine).
/// For now, provides basic form rendering with schema reference.
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../../design/design.dart';
import '../../../widgets/shared/shared.dart';

/// Renders form component (basic version)
class FormRenderer extends StatelessWidget {
  const FormRenderer({
    required this.component,
    this.params = const {},
    super.key,
  });

  final ComponentDescriptor component;
  final Map<String, String> params;

  @override
  Widget build(BuildContext context) {
    final schemaRef = component.schemaRef;
    final resource = component.resource;

    return AppCard(
      title: component.ui?.label ?? 'Form',
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (schemaRef != null) ...[
            Text(
              'Schema: $schemaRef',
              style: AppTypography.bodySmall,
            ),
            const SizedBox(height: AppSpacing.space8),
          ],
          if (resource != null) ...[
            Text(
              'Resource: $resource',
              style: AppTypography.bodySmall,
            ),
            const SizedBox(height: AppSpacing.space8),
          ],
          const SizedBox(height: AppSpacing.space16),
          const Center(
            child: Text(
              'Full form implementation in Task 10',
              style: TextStyle(fontStyle: FontStyle.italic),
            ),
          ),
        ],
      ),
    );
  }
}
