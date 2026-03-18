/// Action group renderer.
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../../design/design.dart';
import '../../../widgets/shared/shared.dart';

/// Renders group of action buttons
class ActionGroupRenderer extends StatelessWidget {
  const ActionGroupRenderer({
    required this.component,
    super.key,
  });

  final ComponentDescriptor component;

  @override
  Widget build(BuildContext context) {
    final actions = component.actions ?? [];
    final filteredActions = actions
        .where((action) => action.permission.allowed)
        .toList();

    if (filteredActions.isEmpty) {
      return const SizedBox.shrink();
    }

    final spacing = (component.config['spacing'] as num?)?.toDouble() ??
        AppSpacing.space8;
    final direction = component.config['direction'] as String? ?? 'horizontal';

    final buttons = filteredActions.map((action) {
      return AppButton(
        label: action.label,
        onPressed: () {
          // TODO: Execute action
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Action: ${action.label}')),
          );
        },
        variant: _parseButtonVariant(action.style),
        size: AppButtonSize.medium,
      );
    }).toList();

    if (direction == 'vertical') {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          for (var i = 0; i < buttons.length; i++) ...[
            if (i > 0) SizedBox(height: spacing),
            buttons[i],
          ],
        ],
      );
    }

    return Wrap(
      spacing: spacing,
      runSpacing: spacing,
      children: buttons,
    );
  }

  AppButtonVariant _parseButtonVariant(String? style) {
    switch (style?.toLowerCase()) {
      case 'primary':
        return AppButtonVariant.primary;
      case 'secondary':
        return AppButtonVariant.secondary;
      case 'tertiary':
        return AppButtonVariant.tertiary;
      case 'destructive':
      case 'danger':
        return AppButtonVariant.destructive;
      default:
        return AppButtonVariant.secondary;
    }
  }
}
