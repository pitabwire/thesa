/// Badge/status renderer.
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../../widgets/shared/shared.dart';

/// Renders badge/status component
class BadgeRenderer extends StatelessWidget {
  const BadgeRenderer({
    required this.component,
    super.key,
  });

  final ComponentDescriptor component;

  @override
  Widget build(BuildContext context) {
    final label = component.config['label'] as String? ??
        component.config['text'] as String? ??
        component.ui?.tooltip ??
        '';
    final variant = component.config['variant'] as String? ??
        component.config['status'] as String? ??
        'default';

    if (label.isEmpty) {
      return const SizedBox.shrink();
    }

    return AppBadge(
      label: label,
      color: _parseVariantColor(variant),
    );
  }

  Color? _parseVariantColor(String variant) {
    switch (variant.toLowerCase()) {
      case 'success':
      case 'completed':
      case 'active':
        return Colors.green;

      case 'warning':
      case 'pending':
      case 'in_progress':
        return Colors.orange;

      case 'error':
      case 'failed':
      case 'cancelled':
      case 'danger':
        return Colors.red;

      case 'info':
      case 'draft':
        return Colors.blue;

      case 'default':
      case 'neutral':
      default:
        return null;
    }
  }
}
