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
        component.ui?.label ??
        '';
    final variant = component.config['variant'] as String? ??
        component.config['status'] as String? ??
        'default';

    if (label.isEmpty) {
      return const SizedBox.shrink();
    }

    return AppBadge(
      label: label,
      variant: _parseVariant(variant),
    );
  }

  AppBadgeVariant _parseVariant(String variant) {
    switch (variant.toLowerCase()) {
      case 'success':
      case 'completed':
      case 'active':
        return AppBadgeVariant.success;

      case 'warning':
      case 'pending':
      case 'in_progress':
        return AppBadgeVariant.warning;

      case 'error':
      case 'failed':
      case 'cancelled':
      case 'danger':
        return AppBadgeVariant.error;

      case 'info':
      case 'draft':
        return AppBadgeVariant.info;

      case 'default':
      case 'neutral':
      default:
        return AppBadgeVariant.neutral;
    }
  }
}
