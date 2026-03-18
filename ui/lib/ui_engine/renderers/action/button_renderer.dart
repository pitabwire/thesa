/// Button renderer.
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../../widgets/shared/shared.dart';

/// Renders button component
class ButtonRenderer extends StatelessWidget {
  const ButtonRenderer({
    required this.component,
    super.key,
  });

  final ComponentDescriptor component;

  @override
  Widget build(BuildContext context) {
    final label = component.config['label'] as String? ??
        component.ui?.label ??
        'Button';
    final icon = component.config['icon'] as String?;
    final variant = _parseVariant(component.config['variant'] as String?);
    final size = _parseSize(component.config['size'] as String?);
    final fullWidth = component.config['fullWidth'] as bool? ?? false;

    final button = AppButton(
      label: label,
      onPressed: () {
        // TODO: Execute action from component.actions[0] or component.config['action']
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Button clicked: $label')),
        );
      },
      variant: variant,
      size: size,
      icon: icon != null ? _parseIcon(icon) : null,
    );

    if (fullWidth) {
      return SizedBox(
        width: double.infinity,
        child: button,
      );
    }

    return button;
  }

  AppButtonVariant _parseVariant(String? variant) {
    switch (variant?.toLowerCase()) {
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
        return AppButtonVariant.primary;
    }
  }

  AppButtonSize _parseSize(String? size) {
    switch (size?.toLowerCase()) {
      case 'small':
        return AppButtonSize.small;
      case 'medium':
        return AppButtonSize.medium;
      case 'large':
        return AppButtonSize.large;
      default:
        return AppButtonSize.medium;
    }
  }

  IconData _parseIcon(String icon) {
    // Simple icon mapping
    switch (icon.toLowerCase()) {
      case 'add':
      case 'plus':
        return Icons.add;
      case 'edit':
        return Icons.edit;
      case 'delete':
        return Icons.delete;
      case 'save':
        return Icons.save;
      case 'cancel':
        return Icons.cancel;
      case 'search':
        return Icons.search;
      case 'filter':
        return Icons.filter_list;
      case 'download':
        return Icons.download;
      case 'upload':
        return Icons.upload;
      case 'refresh':
        return Icons.refresh;
      default:
        return Icons.circle;
    }
  }
}
