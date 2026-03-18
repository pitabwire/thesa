/// Metric/stat renderer.
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../../design/design.dart';

/// Renders metric/stat component (number with label)
class MetricRenderer extends StatelessWidget {
  const MetricRenderer({
    required this.component,
    super.key,
  });

  final ComponentDescriptor component;

  @override
  Widget build(BuildContext context) {
    final value = component.config['value']?.toString() ?? '0';
    final label = component.config['label'] as String? ??
        component.ui?.label ??
        '';
    final delta = component.config['delta']?.toString();
    final trend = component.config['trend'] as String?; // 'up', 'down', 'neutral'

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.space16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            if (label.isNotEmpty)
              Text(
                label,
                style: AppTypography.labelMedium.copyWith(
                  color: Theme.of(context).colorScheme.onSurfaceVariant,
                ),
              ),
            const SizedBox(height: AppSpacing.space8),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Text(
                  value,
                  style: AppTypography.displaySmall.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                if (delta != null) _buildDelta(context, delta, trend),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildDelta(BuildContext context, String delta, String? trend) {
    Color color;
    IconData icon;

    switch (trend?.toLowerCase()) {
      case 'up':
        color = Theme.of(context).colorScheme.error;
        icon = Icons.trending_up;
        break;
      case 'down':
        color = Theme.of(context).colorScheme.primary;
        icon = Icons.trending_down;
        break;
      case 'neutral':
      default:
        color = Theme.of(context).colorScheme.onSurfaceVariant;
        icon = Icons.trending_flat;
    }

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 16, color: color),
        const SizedBox(width: 4),
        Text(
          delta,
          style: AppTypography.labelSmall.copyWith(color: color),
        ),
      ],
    );
  }
}
