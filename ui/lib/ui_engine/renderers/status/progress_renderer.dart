/// Progress indicator renderer.
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../../design/design.dart';

/// Renders progress indicator
class ProgressRenderer extends StatelessWidget {
  const ProgressRenderer({
    required this.component,
    super.key,
  });

  final ComponentDescriptor component;

  @override
  Widget build(BuildContext context) {
    final type = component.config['type'] as String? ?? 'linear';
    final value = (component.config['value'] as num?)?.toDouble();
    final label = component.config['label'] as String? ?? component.ui?.label;
    final showPercentage = component.config['showPercentage'] as bool? ?? true;

    if (type == 'circular') {
      return Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          SizedBox(
            width: 48,
            height: 48,
            child: value != null
                ? CircularProgressIndicator(value: value / 100)
                : const CircularProgressIndicator(),
          ),
          if (label != null) ...[
            const SizedBox(height: AppSpacing.space8),
            Text(label, style: AppTypography.labelSmall),
          ],
          if (showPercentage && value != null) ...[
            const SizedBox(height: 4),
            Text(
              '${value.toInt()}%',
              style: AppTypography.labelSmall.copyWith(
                fontWeight: FontWeight.bold,
              ),
            ),
          ],
        ],
      );
    }

    // Linear progress
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        if (label != null) ...[
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(label, style: AppTypography.labelMedium),
              if (showPercentage && value != null)
                Text(
                  '${value.toInt()}%',
                  style: AppTypography.labelSmall.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
            ],
          ),
          const SizedBox(height: AppSpacing.space8),
        ],
        ClipRRect(
          borderRadius: BorderRadius.circular(4),
          child: SizedBox(
            height: 8,
            child: value != null
                ? LinearProgressIndicator(value: value / 100)
                : const LinearProgressIndicator(),
          ),
        ),
      ],
    );
  }
}
