import 'package:flutter/material.dart';
import 'colors.dart';

/// Input decoration and form field styles.
abstract final class StawiInputStyles {
  static const _radius = 10.0;

  /// Standard input decoration theme.
  static InputDecorationTheme inputTheme(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    final border = isDark ? StawiColors.darkBorder : StawiColors.lightBorder;
    final input = isDark ? StawiColors.darkInput : StawiColors.lightInput;
    final muted =
        isDark ? StawiColors.darkMutedForeground : StawiColors.lightMutedForeground;
    final foreground =
        isDark ? StawiColors.darkForeground : StawiColors.lightForeground;
    final ring = isDark ? StawiColors.darkRing : StawiColors.lightRing;

    return InputDecorationTheme(
      filled: true,
      fillColor: Colors.transparent,
      contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      hintStyle: TextStyle(
        color: muted,
        fontSize: 14,
        fontWeight: FontWeight.w400,
      ),
      labelStyle: TextStyle(
        color: muted,
        fontSize: 14,
        fontWeight: FontWeight.w500,
      ),
      floatingLabelStyle: TextStyle(
        color: foreground,
        fontSize: 14,
        fontWeight: FontWeight.w500,
      ),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(_radius),
        borderSide: BorderSide(color: border),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(_radius),
        borderSide: BorderSide(color: input),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(_radius),
        borderSide: BorderSide(color: ring, width: 2),
      ),
      errorBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(_radius),
        borderSide: const BorderSide(color: StawiColors.rose),
      ),
      focusedErrorBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(_radius),
        borderSide: const BorderSide(color: StawiColors.rose, width: 2),
      ),
      disabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(_radius),
        borderSide: BorderSide(color: border.withValues(alpha: 0.5)),
      ),
      errorStyle: const TextStyle(
        color: StawiColors.rose,
        fontSize: 12,
        fontWeight: FontWeight.w500,
      ),
    );
  }

  /// Search field decoration (pill-shaped).
  static InputDecoration searchDecoration({
    required Brightness brightness,
    String hintText = 'Search...',
  }) {
    final isDark = brightness == Brightness.dark;
    final muted =
        isDark ? StawiColors.darkMutedForeground : StawiColors.lightMutedForeground;
    final border = isDark ? StawiColors.darkBorder : StawiColors.lightBorder;

    return InputDecoration(
      hintText: hintText,
      hintStyle: TextStyle(color: muted, fontSize: 14),
      prefixIcon: Icon(Icons.search, color: muted, size: 20),
      filled: true,
      fillColor: Colors.transparent,
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(9999),
        borderSide: BorderSide(color: border),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(9999),
        borderSide: BorderSide(color: border),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(9999),
        borderSide: BorderSide(color: StawiColors.emerald, width: 2),
      ),
    );
  }
}
