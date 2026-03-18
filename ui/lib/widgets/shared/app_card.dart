/// Surface container with optional header, body, and footer.
library;

import 'package:flutter/material.dart';

import '../../design/design.dart';

/// Card component with header, body, and footer sections
class AppCard extends StatelessWidget {
  const AppCard({
    required this.body,
    this.title,
    this.subtitle,
    this.actions,
    this.footer,
    this.elevation,
    this.padding,
    this.onTap,
    super.key,
  });

  /// Card body content (required)
  final Widget body;

  /// Optional card title (shown in header)
  final String? title;

  /// Optional card subtitle (shown in header below title)
  final String? subtitle;

  /// Optional action widgets (shown in header trailing)
  final List<Widget>? actions;

  /// Optional footer widget
  final Widget? footer;

  /// Card elevation (default: medium)
  final double? elevation;

  /// Body padding (default: space16)
  final EdgeInsetsGeometry? padding;

  /// Optional tap handler (makes card interactive)
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final hasHeader = title != null || actions != null;

    // Build header if title or actions exist
    Widget? header;
    if (hasHeader) {
      header = Padding(
        padding: const EdgeInsets.all(AppSpacing.space16),
        child: Row(
          children: [
            // Title and subtitle column
            if (title != null)
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title!,
                      style: theme.textTheme.titleLarge,
                    ),
                    if (subtitle != null) ...[
                      const SizedBox(height: AppSpacing.space4),
                      Text(
                        subtitle!,
                        style: theme.textTheme.bodySmall,
                      ),
                    ],
                  ],
                ),
              ),
            // Actions
            if (actions != null) ...[
              if (title != null) const SizedBox(width: AppSpacing.space8),
              ...actions!,
            ],
          ],
        ),
      );
    }

    // Build card content
    final content = Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (header != null) ...[
          header,
          const Divider(height: 1),
        ],
        Padding(
          padding: padding ??
              const EdgeInsets.all(AppSpacing.space16),
          child: body,
        ),
        if (footer != null) ...[
          const Divider(height: 1),
          Padding(
            padding: const EdgeInsets.all(AppSpacing.space16),
            child: footer,
          ),
        ],
      ],
    );

    // Build card with optional tap handler
    return Card(
      elevation: elevation ?? AppElevation.medium,
      child: onTap != null
          ? InkWell(
              onTap: onTap,
              borderRadius: BorderRadius.circular(AppBorderRadius.medium),
              child: content,
            )
          : content,
    );
  }
}
