/// Badge widget for counts and notifications.
library;

import 'package:flutter/material.dart';

import '../../design/design.dart';

/// Badge component for notification counts and indicators
class AppBadge extends StatelessWidget {
  const AppBadge({
    required this.label,
    this.color,
    this.textColor,
    this.small = false,
    super.key,
  });

  /// Badge label text
  final String label;

  /// Background color (defaults to primary)
  final Color? color;

  /// Text color (defaults to onPrimary)
  final Color? textColor;

  /// Use small size
  final bool small;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final bgColor = color ?? theme.colorScheme.primary;
    final fgColor = textColor ?? theme.colorScheme.onPrimary;

    return Container(
      padding: EdgeInsets.symmetric(
        horizontal: small ? AppSpacing.space4 : AppSpacing.space8,
        vertical: small ? AppSpacing.space2 : AppSpacing.space4,
      ),
      decoration: BoxDecoration(
        color: bgColor,
        borderRadius: BorderRadius.circular(AppBorderRadius.pill),
      ),
      child: Text(
        label,
        style: (small
                ? AppTypography.labelSmall
                : AppTypography.labelMedium)
            .copyWith(color: fgColor),
      ),
    );
  }
}

/// Dot badge (indicator only, no text)
class AppDotBadge extends StatelessWidget {
  const AppDotBadge({
    this.color,
    this.size = 8.0,
    super.key,
  });

  /// Dot color (defaults to error color)
  final Color? color;

  /// Dot size
  final double size;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final dotColor = color ?? theme.colorScheme.error;

    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        color: dotColor,
        shape: BoxShape.circle,
      ),
    );
  }
}

/// Badge positioned on another widget
class AppBadgedWidget extends StatelessWidget {
  const AppBadgedWidget({
    required this.child,
    required this.badge,
    this.offset = const Offset(0, 0),
    super.key,
  });

  /// Child widget to badge
  final Widget child;

  /// Badge widget
  final Widget badge;

  /// Badge position offset
  final Offset offset;

  @override
  Widget build(BuildContext context) {
    return Stack(
      clipBehavior: Clip.none,
      children: [
        child,
        Positioned(
          right: offset.dx,
          top: offset.dy,
          child: badge,
        ),
      ],
    );
  }
}
