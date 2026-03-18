/// Standardized button component with consistent styling.
library;

import 'package:flutter/material.dart';

import '../../design/design.dart';

/// Button variant types
enum AppButtonVariant {
  /// Solid primary color background
  primary,

  /// Outlined with primary color border
  secondary,

  /// Text-only, no border or background
  tertiary,

  /// Solid red background for destructive actions
  destructive,
}

/// Button size options
enum AppButtonSize {
  /// Height 32px, smaller text
  small,

  /// Height 40px, standard text (default)
  medium,

  /// Height 48px, larger text
  large,
}

/// Standardized button with consistent sizing, colors, and states
class AppButton extends StatelessWidget {
  const AppButton({
    required this.onPressed,
    required this.label,
    this.variant = AppButtonVariant.primary,
    this.size = AppButtonSize.medium,
    this.icon,
    this.iconOnly = false,
    this.isLoading = false,
    this.fullWidth = false,
    super.key,
  });

  /// Button press callback (null = disabled)
  final VoidCallback? onPressed;

  /// Button text label
  final String label;

  /// Visual variant
  final AppButtonVariant variant;

  /// Button size
  final AppButtonSize size;

  /// Optional leading icon
  final IconData? icon;

  /// Show only icon (no label)
  final bool iconOnly;

  /// Show loading spinner
  final bool isLoading;

  /// Expand to full width
  final bool fullWidth;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDisabled = onPressed == null || isLoading;

    // Get button height based on size
    final height = switch (size) {
      AppButtonSize.small => AppSizing.buttonSmall,
      AppButtonSize.medium => AppSizing.buttonMedium,
      AppButtonSize.large => AppSizing.buttonLarge,
    };

    // Get text style based on size
    final textStyle = switch (size) {
      AppButtonSize.small => AppTypography.bodySmall,
      AppButtonSize.medium => AppTypography.labelLarge,
      AppButtonSize.large => AppTypography.labelLarge,
    };

    // Get icon size based on button size
    final iconSize = switch (size) {
      AppButtonSize.small => AppSizing.iconSmall,
      AppButtonSize.medium => AppSizing.iconMedium,
      AppButtonSize.large => AppSizing.iconMedium,
    };

    // Build button child (icon + label or loading)
    Widget child;
    if (isLoading) {
      child = SizedBox(
        width: iconSize,
        height: iconSize,
        child: CircularProgressIndicator(
          strokeWidth: 2,
          valueColor: AlwaysStoppedAnimation<Color>(
            _getLoadingColor(theme),
          ),
        ),
      );
    } else if (iconOnly && icon != null) {
      child = Icon(icon, size: iconSize);
    } else if (icon != null) {
      child = Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: iconSize),
          const SizedBox(width: AppSpacing.space8),
          Text(label, style: textStyle),
        ],
      );
    } else {
      child = Text(label, style: textStyle);
    }

    // Build appropriate button type based on variant
    final button = switch (variant) {
      AppButtonVariant.primary => ElevatedButton(
          onPressed: isDisabled ? null : onPressed,
          style: ElevatedButton.styleFrom(
            minimumSize: Size(
              iconOnly ? height : AppSizing.minTouchTarget,
              height,
            ),
          ),
          child: child,
        ),
      AppButtonVariant.secondary => OutlinedButton(
          onPressed: isDisabled ? null : onPressed,
          style: OutlinedButton.styleFrom(
            minimumSize: Size(
              iconOnly ? height : AppSizing.minTouchTarget,
              height,
            ),
          ),
          child: child,
        ),
      AppButtonVariant.tertiary => TextButton(
          onPressed: isDisabled ? null : onPressed,
          style: TextButton.styleFrom(
            minimumSize: Size(
              iconOnly ? height : AppSizing.minTouchTarget,
              height,
            ),
          ),
          child: child,
        ),
      AppButtonVariant.destructive => ElevatedButton(
          onPressed: isDisabled ? null : onPressed,
          style: ElevatedButton.styleFrom(
            backgroundColor: theme.colorScheme.error,
            foregroundColor: theme.colorScheme.onError,
            minimumSize: Size(
              iconOnly ? height : AppSizing.minTouchTarget,
              height,
            ),
          ),
          child: child,
        ),
    };

    // Wrap in SizedBox if full width
    if (fullWidth) {
      return SizedBox(
        width: double.infinity,
        child: button,
      );
    }

    return button;
  }

  /// Get loading indicator color based on variant
  Color _getLoadingColor(ThemeData theme) {
    return switch (variant) {
      AppButtonVariant.primary => theme.colorScheme.onPrimary,
      AppButtonVariant.secondary => theme.colorScheme.primary,
      AppButtonVariant.tertiary => theme.colorScheme.primary,
      AppButtonVariant.destructive => theme.colorScheme.onError,
    };
  }
}
