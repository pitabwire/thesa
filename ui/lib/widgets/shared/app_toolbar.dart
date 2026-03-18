/// Toolbar widget for page actions.
library;

import 'package:flutter/material.dart';

import '../../core/core.dart';
import '../../design/design.dart';
import 'app_button.dart';

/// Toolbar widget with actions
class AppToolbar extends StatelessWidget {
  const AppToolbar({
    required this.actions,
    this.title,
    this.onActionPressed,
    super.key,
  });

  final String? title;
  final List<ActionDescriptor> actions;
  final void Function(ActionDescriptor action)? onActionPressed;

  @override
  Widget build(BuildContext context) {
    final visibleActions = actions
        .where((action) => action.permission.allowed)
        .toList();

    if (visibleActions.isEmpty && title == null) {
      return const SizedBox.shrink();
    }

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.space12),
        child: Row(
          children: [
            if (title != null) ...[
              Text(
                title!,
                style: AppTypography.titleMedium,
              ),
              const Spacer(),
            ],
            ...visibleActions.map((action) {
              return Padding(
                padding: const EdgeInsets.only(left: AppSpacing.space8),
                child: AppButton(
                  label: action.label,
                  onPressed: () => onActionPressed?.call(action),
                  variant: _getVariant(action.style),
                  icon: action.icon != null ? _parseIcon(action.icon!) : null,
                  size: AppButtonSize.small,
                ),
              );
            }),
          ],
        ),
      ),
    );
  }

  AppButtonVariant _getVariant(String? style) {
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

  IconData _parseIcon(String icon) {
    switch (icon.toLowerCase()) {
      case 'add':
      case 'plus':
        return Icons.add;
      case 'edit':
        return Icons.edit;
      case 'delete':
        return Icons.delete;
      case 'download':
        return Icons.download;
      case 'upload':
        return Icons.upload;
      case 'refresh':
        return Icons.refresh;
      case 'filter':
        return Icons.filter_list;
      default:
        return Icons.circle;
    }
  }
}
