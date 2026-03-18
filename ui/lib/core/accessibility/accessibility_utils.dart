/// Accessibility utilities and helpers.
library;

import 'package:flutter/material.dart';

/// Accessibility utilities
class AccessibilityUtils {
  AccessibilityUtils._();

  /// Check if reduce motion is enabled
  static bool reduceMotionEnabled(BuildContext context) {
    return MediaQuery.disableAnimationsOf(context);
  }

  /// Get animation duration based on reduce motion setting
  ///
  /// Returns Duration.zero if reduce motion is enabled, otherwise returns the provided duration.
  static Duration getAnimationDuration(
    BuildContext context,
    Duration normalDuration,
  ) {
    return reduceMotionEnabled(context) ? Duration.zero : normalDuration;
  }

  /// Get font scale factor
  static double getFontScale(BuildContext context) {
    return MediaQuery.textScalerOf(context).scale(1.0);
  }

  /// Check if large text is enabled (scale >= 1.3)
  static bool isLargeTextEnabled(BuildContext context) {
    return getFontScale(context) >= 1.3;
  }

  /// Get accessible touch target size
  ///
  /// Ensures minimum 48x48 logical pixels for interactive elements.
  static Size getMinimumTouchTarget() {
    return const Size(48, 48);
  }

  /// Wrap size to meet minimum touch target
  static Size ensureMinimumTouchTarget(Size size) {
    final minSize = getMinimumTouchTarget();
    return Size(
      size.width < minSize.width ? minSize.width : size.width,
      size.height < minSize.height ? minSize.height : size.height,
    );
  }

  /// Create semantic label for screen readers
  ///
  /// Combines multiple strings into a readable label.
  static String createSemanticLabel(List<String> parts) {
    return parts.where((part) => part.isNotEmpty).join(', ');
  }

  /// Announce message to screen readers
  ///
  /// Creates a live region announcement.
  static void announce(BuildContext context, String message) {
    // Use SemanticsService to announce
    SemanticsService.announce(message, TextDirection.ltr);
  }
}

/// Minimum touch target size constant
const double kMinimumTouchTarget = 48.0;

/// WCAG AA contrast ratio thresholds
class ContrastRatios {
  ContrastRatios._();

  /// Minimum contrast for normal text (4.5:1)
  static const double normalText = 4.5;

  /// Minimum contrast for large text (3:1)
  static const double largeText = 3.0;

  /// Minimum contrast for UI components (3:1)
  static const double uiComponents = 3.0;
}

/// Calculate luminance of a color
double _calculateLuminance(Color color) {
  return color.computeLuminance();
}

/// Calculate contrast ratio between two colors
///
/// Returns a value where 1.0 is no contrast and 21.0 is maximum contrast.
/// WCAG AA requires 4.5:1 for normal text and 3:1 for large text.
double calculateContrastRatio(Color foreground, Color background) {
  final lum1 = _calculateLuminance(foreground);
  final lum2 = _calculateLuminance(background);

  final lighter = lum1 > lum2 ? lum1 : lum2;
  final darker = lum1 > lum2 ? lum2 : lum1;

  return (lighter + 0.05) / (darker + 0.05);
}

/// Check if color combination meets WCAG AA for normal text
bool meetsWCAGAANormalText(Color foreground, Color background) {
  return calculateContrastRatio(foreground, background) >= ContrastRatios.normalText;
}

/// Check if color combination meets WCAG AA for large text
bool meetsWCAGAALargeText(Color foreground, Color background) {
  return calculateContrastRatio(foreground, background) >= ContrastRatios.largeText;
}
