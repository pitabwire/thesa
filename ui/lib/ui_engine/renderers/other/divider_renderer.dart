/// Divider/separator renderer.
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../../design/design.dart';

/// Renders divider/separator
class DividerRenderer extends StatelessWidget {
  const DividerRenderer({
    required this.component,
    super.key,
  });

  final ComponentDescriptor component;

  @override
  Widget build(BuildContext context) {
    final type = component.config['type'] as String? ?? 'horizontal';
    final spacing = (component.config['spacing'] as num?)?.toDouble() ??
        AppSpacing.space16;
    final thickness = (component.config['thickness'] as num?)?.toDouble() ?? 1.0;
    final label = component.config['label'] as String? ?? component.ui?.label;

    if (type == 'vertical') {
      return Container(
        width: thickness,
        height: spacing,
        color: Theme.of(context).dividerColor,
      );
    }

    // Horizontal divider
    if (label != null && label.isNotEmpty) {
      // Divider with label
      return Padding(
        padding: EdgeInsets.symmetric(vertical: spacing),
        child: Row(
          children: [
            Expanded(
              child: Divider(thickness: thickness),
            ),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: AppSpacing.space8),
              child: Text(
                label,
                style: Theme.of(context).textTheme.labelSmall,
              ),
            ),
            Expanded(
              child: Divider(thickness: thickness),
            ),
          ],
        ),
      );
    }

    return Padding(
      padding: EdgeInsets.symmetric(vertical: spacing / 2),
      child: Divider(thickness: thickness),
    );
  }
}
