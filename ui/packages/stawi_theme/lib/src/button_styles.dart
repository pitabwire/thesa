import 'package:flutter/material.dart';
import 'colors.dart';

/// Button styles matching BoundaryML's button variants.
///
/// Provides factories for primary, outline, ghost, destructive, and link
/// button styles in both light and dark modes.
abstract final class StawiButtonStyles {
  static const _radiusFull = 9999.0;
  static const _radiusLg = 10.0;

  // ──────────────────────────────────────────────
  // Elevated / Primary — green CTA
  // ──────────────────────────────────────────────

  /// Primary filled button (green CTA style).
  static ButtonStyle primary(Brightness brightness) {
    return ElevatedButton.styleFrom(
      backgroundColor: StawiColors.emerald,
      foregroundColor: Colors.white,
      elevation: 0,
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(_radiusFull),
      ),
      textStyle: const TextStyle(
        fontSize: 14,
        fontWeight: FontWeight.w600,
        letterSpacing: 0,
      ),
    ).copyWith(
      overlayColor: WidgetStateProperty.resolveWith((states) {
        if (states.contains(WidgetState.hovered)) {
          return StawiColors.emeraldDark.withValues(alpha: 0.15);
        }
        if (states.contains(WidgetState.pressed)) {
          return StawiColors.emeraldDark.withValues(alpha: 0.25);
        }
        return null;
      }),
    );
  }

  /// Large primary button variant.
  static ButtonStyle primaryLarge(Brightness brightness) {
    return primary(brightness).copyWith(
      padding: WidgetStateProperty.all(
        const EdgeInsets.symmetric(horizontal: 32, vertical: 16),
      ),
      textStyle: WidgetStateProperty.all(
        const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
      ),
    );
  }

  // ──────────────────────────────────────────────
  // Outlined
  // ──────────────────────────────────────────────

  /// Outlined button with border.
  static ButtonStyle outline(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return OutlinedButton.styleFrom(
      foregroundColor: isDark ? StawiColors.gray50 : StawiColors.gray900,
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
      side: BorderSide(
        color: isDark ? StawiColors.darkBorder : StawiColors.lightBorder,
      ),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(_radiusFull),
      ),
      textStyle: const TextStyle(
        fontSize: 14,
        fontWeight: FontWeight.w600,
        letterSpacing: 0,
      ),
    ).copyWith(
      overlayColor: WidgetStateProperty.resolveWith((states) {
        final muted = isDark ? StawiColors.darkMuted : StawiColors.lightMuted;
        if (states.contains(WidgetState.hovered)) {
          return muted.withValues(alpha: 0.5);
        }
        return null;
      }),
    );
  }

  /// Large outlined button variant.
  static ButtonStyle outlineLarge(Brightness brightness) {
    return outline(brightness).copyWith(
      padding: WidgetStateProperty.all(
        const EdgeInsets.symmetric(horizontal: 32, vertical: 16),
      ),
      textStyle: WidgetStateProperty.all(
        const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
      ),
    );
  }

  // ──────────────────────────────────────────────
  // Ghost
  // ──────────────────────────────────────────────

  /// Ghost (text-only) button, no border or fill.
  static ButtonStyle ghost(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return TextButton.styleFrom(
      foregroundColor:
          isDark ? StawiColors.darkMutedForeground : StawiColors.lightMutedForeground,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(_radiusLg),
      ),
      textStyle: const TextStyle(
        fontSize: 14,
        fontWeight: FontWeight.w500,
        letterSpacing: 0,
      ),
    ).copyWith(
      overlayColor: WidgetStateProperty.resolveWith((states) {
        final muted = isDark ? StawiColors.darkMuted : StawiColors.lightMuted;
        if (states.contains(WidgetState.hovered)) {
          return muted.withValues(alpha: 0.5);
        }
        return null;
      }),
    );
  }

  // ──────────────────────────────────────────────
  // Destructive
  // ──────────────────────────────────────────────

  /// Destructive (red) filled button.
  static ButtonStyle destructive(Brightness brightness) {
    return ElevatedButton.styleFrom(
      backgroundColor: StawiColors.rose,
      foregroundColor: Colors.white,
      elevation: 0,
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(_radiusFull),
      ),
      textStyle: const TextStyle(
        fontSize: 14,
        fontWeight: FontWeight.w600,
      ),
    );
  }

  // ──────────────────────────────────────────────
  // Secondary — soft fill
  // ──────────────────────────────────────────────

  /// Secondary button with muted background.
  static ButtonStyle secondary(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return ElevatedButton.styleFrom(
      backgroundColor: isDark ? StawiColors.darkMuted : StawiColors.lightMuted,
      foregroundColor: isDark ? StawiColors.darkForeground : StawiColors.lightForeground,
      elevation: 0,
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(_radiusFull),
      ),
      textStyle: const TextStyle(
        fontSize: 14,
        fontWeight: FontWeight.w600,
      ),
    );
  }

  // ──────────────────────────────────────────────
  // Icon button
  // ──────────────────────────────────────────────

  /// Bordered icon button (like mobile menu toggle).
  static ButtonStyle iconOutline(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return IconButton.styleFrom(
      foregroundColor: isDark ? StawiColors.darkForeground : StawiColors.lightForeground,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(6),
        side: BorderSide(
          color: isDark ? StawiColors.darkBorder : StawiColors.lightBorder,
        ),
      ),
      minimumSize: const Size(32, 32),
      padding: const EdgeInsets.all(4),
    );
  }
}
