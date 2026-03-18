/// Small labeled element for tags, filters, and selections.
library;

import 'package:flutter/material.dart';

import '../../design/design.dart';

/// Chip variant types
enum AppChipVariant {
  /// Filter chip with optional remove button
  filter,

  /// Toggleable selection chip
  selection,

  /// Read-only info chip
  info,

  /// Status chip with color coding
  status,
}

/// Chip component for tags, filters, and labels
class AppChip extends StatelessWidget {
  const AppChip({
    required this.label,
    this.variant = AppChipVariant.info,
    this.icon,
    this.onDeleted,
    this.onSelected,
    this.isSelected = false,
    this.statusValue,
    super.key,
  });

  /// Chip label text
  final String label;

  /// Chip variant
  final AppChipVariant variant;

  /// Optional leading icon
  final IconData? icon;

  /// Delete callback (for filter chips)
  final VoidCallback? onDeleted;

  /// Selection callback (for selection chips)
  final ValueChanged<bool>? onSelected;

  /// Whether chip is selected (for selection chips)
  final bool isSelected;

  /// Status value (for status chips) - determines color
  final String? statusValue;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    return switch (variant) {
      AppChipVariant.filter => FilterChip(
          label: Text(label),
          avatar: icon != null ? Icon(icon, size: AppSizing.iconSmall) : null,
          onDeleted: onDeleted,
          deleteIcon: const Icon(Icons.close, size: AppSizing.iconSmall),
          onSelected: onSelected != null ? (_) => onSelected!(true) : null,
        ),
      AppChipVariant.selection => FilterChip(
          label: Text(label),
          avatar: icon != null ? Icon(icon, size: AppSizing.iconSmall) : null,
          selected: isSelected,
          onSelected: onSelected,
          checkmarkColor: theme.colorScheme.primary,
        ),
      AppChipVariant.info => Chip(
          label: Text(label),
          avatar: icon != null
              ? Icon(icon, size: AppSizing.iconSmall)
              : null,
        ),
      AppChipVariant.status => _StatusChip(
          label: label,
          statusValue: statusValue ?? label,
          isDark: isDark,
        ),
    };
  }
}

/// Status chip with semantic color coding
class _StatusChip extends StatelessWidget {
  const _StatusChip({
    required this.label,
    required this.statusValue,
    required this.isDark,
  });

  final String label;
  final String statusValue;
  final bool isDark;

  @override
  Widget build(BuildContext context) {
    final statusColor = AppColors.getStatusColor(
      statusValue,
      isDark: isDark,
    );
    final textColor = AppColors.getStatusTextColor(
      statusValue,
      isDark: isDark,
    );

    return Container(
      padding: const EdgeInsets.symmetric(
        horizontal: AppSpacing.space12,
        vertical: AppSpacing.space4,
      ),
      decoration: BoxDecoration(
        color: statusColor.withOpacity(0.12),
        borderRadius: BorderRadius.circular(AppBorderRadius.small),
        border: Border.all(
          color: statusColor,
          width: 1,
        ),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          // Status indicator dot
          Container(
            width: 6,
            height: 6,
            decoration: BoxDecoration(
              color: statusColor,
              shape: BoxShape.circle,
            ),
          ),
          const SizedBox(width: AppSpacing.space8),
          // Label
          Text(
            label,
            style: AppTypography.labelMedium.copyWith(
              color: textColor,
            ),
          ),
        ],
      ),
    );
  }
}
