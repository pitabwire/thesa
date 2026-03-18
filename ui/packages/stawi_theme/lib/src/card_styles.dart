import 'package:flutter/material.dart';
import 'colors.dart';

/// Card and container styles.
abstract final class StawiCardStyles {
  static const _radius = 10.0;

  /// Default card theme.
  static CardThemeData cardTheme(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return CardThemeData(
      color: isDark ? StawiColors.darkCard : StawiColors.lightCard,
      elevation: 0,
      margin: EdgeInsets.zero,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(_radius),
        side: BorderSide(
          color: isDark ? StawiColors.darkBorder : StawiColors.lightBorder,
        ),
      ),
    );
  }

  /// Bordered container decoration (for code blocks, terminals, etc.).
  static BoxDecoration bordered(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return BoxDecoration(
      color: isDark ? StawiColors.darkCard : StawiColors.lightCard,
      borderRadius: BorderRadius.circular(_radius),
      border: Border.all(
        color: isDark ? StawiColors.darkBorder : StawiColors.lightBorder,
      ),
    );
  }

  /// Muted container decoration (for backgrounds, sections).
  static BoxDecoration muted(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return BoxDecoration(
      color: isDark ? StawiColors.darkMuted : StawiColors.lightMuted,
      borderRadius: BorderRadius.circular(_radius - 2),
    );
  }

  /// Hover-aware container decoration for bento cards.
  static BoxDecoration bento(Brightness brightness, {bool hovered = false}) {
    final isDark = brightness == Brightness.dark;
    return BoxDecoration(
      color: hovered
          ? (isDark ? StawiColors.darkMuted.withValues(alpha: 0.3) : StawiColors.lightMuted.withValues(alpha: 0.3))
          : Colors.transparent,
    );
  }

  /// CTA gradient decoration.
  static BoxDecoration ctaGradient() {
    return BoxDecoration(
      gradient: StawiColors.ctaGradient,
      borderRadius: BorderRadius.circular(12),
      border: Border.all(color: Colors.white.withValues(alpha: 0.1)),
    );
  }

  /// Status pill decoration (for badges, status indicators).
  static BoxDecoration statusPill(Color color) {
    return BoxDecoration(
      color: color.withValues(alpha: 0.1),
      borderRadius: BorderRadius.circular(9999),
      border: Border.all(color: color.withValues(alpha: 0.3)),
    );
  }
}
