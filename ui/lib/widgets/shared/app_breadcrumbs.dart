/// Breadcrumbs navigation widget.
library;

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/core.dart';
import '../../design/design.dart';

/// Breadcrumbs widget
class AppBreadcrumbs extends StatelessWidget {
  const AppBreadcrumbs({
    required this.items,
    super.key,
  });

  final List<BreadcrumbItem> items;

  @override
  Widget build(BuildContext context) {
    if (items.isEmpty) {
      return const SizedBox.shrink();
    }

    return Row(
      children: [
        for (var i = 0; i < items.length; i++) ...[
          if (i > 0) ...[
            const SizedBox(width: AppSpacing.space8),
            Icon(
              Icons.chevron_right,
              size: 16,
              color: Theme.of(context).colorScheme.onSurfaceVariant,
            ),
            const SizedBox(width: AppSpacing.space8),
          ],
          _buildItem(context, items[i], isLast: i == items.length - 1),
        ],
      ],
    );
  }

  Widget _buildItem(BuildContext context, BreadcrumbItem item,
      {required bool isLast}) {
    final textStyle = isLast
        ? AppTypography.bodySmall
        : AppTypography.bodySmall.copyWith(
            color: Theme.of(context).colorScheme.primary,
          );

    if (item.path != null && !isLast) {
      return InkWell(
        onTap: () => context.go(item.path!),
        borderRadius: BorderRadius.circular(4),
        child: Padding(
          padding: const EdgeInsets.symmetric(
            horizontal: 4,
            vertical: 2,
          ),
          child: Text(item.label, style: textStyle),
        ),
      );
    }

    return Padding(
      padding: const EdgeInsets.symmetric(
        horizontal: 4,
        vertical: 2,
      ),
      child: Text(item.label, style: textStyle),
    );
  }
}
