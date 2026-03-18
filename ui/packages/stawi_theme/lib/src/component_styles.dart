import 'package:flutter/material.dart';
import 'colors.dart';

/// Styles for Material components (AppBar, NavigationBar, Dialog, etc.).
abstract final class StawiComponentStyles {
  // ──────────────────────────────────────────────
  // AppBar
  // ──────────────────────────────────────────────

  static AppBarTheme appBarTheme(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return AppBarTheme(
      backgroundColor: Colors.transparent,
      foregroundColor: isDark ? StawiColors.darkForeground : StawiColors.lightForeground,
      elevation: 0,
      scrolledUnderElevation: 0,
      centerTitle: false,
      titleTextStyle: TextStyle(
        fontSize: 20,
        fontWeight: FontWeight.w700,
        letterSpacing: -0.3,
        color: isDark ? StawiColors.darkForeground : StawiColors.lightForeground,
      ),
    );
  }

  // ──────────────────────────────────────────────
  // Navigation Bar (bottom)
  // ──────────────────────────────────────────────

  static NavigationBarThemeData navigationBarTheme(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return NavigationBarThemeData(
      backgroundColor: isDark ? StawiColors.darkBackground : StawiColors.lightBackground,
      indicatorColor: StawiColors.emerald.withValues(alpha: 0.15),
      labelBehavior: NavigationDestinationLabelBehavior.alwaysShow,
      elevation: 0,
      surfaceTintColor: Colors.transparent,
      iconTheme: WidgetStateProperty.resolveWith((states) {
        if (states.contains(WidgetState.selected)) {
          return const IconThemeData(color: StawiColors.emerald, size: 24);
        }
        return IconThemeData(
          color: isDark ? StawiColors.darkMutedForeground : StawiColors.lightMutedForeground,
          size: 24,
        );
      }),
      labelTextStyle: WidgetStateProperty.resolveWith((states) {
        if (states.contains(WidgetState.selected)) {
          return const TextStyle(
            fontSize: 12,
            fontWeight: FontWeight.w600,
            color: StawiColors.emerald,
          );
        }
        return TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.w500,
          color: isDark ? StawiColors.darkMutedForeground : StawiColors.lightMutedForeground,
        );
      }),
    );
  }

  // ──────────────────────────────────────────────
  // Navigation Rail (side)
  // ──────────────────────────────────────────────

  static NavigationRailThemeData navigationRailTheme(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return NavigationRailThemeData(
      backgroundColor: isDark ? StawiColors.darkBackground : StawiColors.lightBackground,
      indicatorColor: StawiColors.emerald.withValues(alpha: 0.15),
      elevation: 0,
      selectedIconTheme: const IconThemeData(color: StawiColors.emerald, size: 24),
      unselectedIconTheme: IconThemeData(
        color: isDark ? StawiColors.darkMutedForeground : StawiColors.lightMutedForeground,
        size: 24,
      ),
      selectedLabelTextStyle: const TextStyle(
        fontSize: 12,
        fontWeight: FontWeight.w600,
        color: StawiColors.emerald,
      ),
      unselectedLabelTextStyle: TextStyle(
        fontSize: 12,
        fontWeight: FontWeight.w500,
        color: isDark ? StawiColors.darkMutedForeground : StawiColors.lightMutedForeground,
      ),
    );
  }

  // ──────────────────────────────────────────────
  // Dialog
  // ──────────────────────────────────────────────

  static DialogThemeData dialogTheme(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return DialogThemeData(
      backgroundColor: isDark ? StawiColors.darkCard : StawiColors.lightCard,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(
          color: isDark ? StawiColors.darkBorder : StawiColors.lightBorder,
        ),
      ),
      titleTextStyle: TextStyle(
        fontSize: 18,
        fontWeight: FontWeight.w600,
        letterSpacing: -0.3,
        color: isDark ? StawiColors.darkForeground : StawiColors.lightForeground,
      ),
      contentTextStyle: TextStyle(
        fontSize: 14,
        fontWeight: FontWeight.w400,
        height: 1.6,
        color: isDark ? StawiColors.darkMutedForeground : StawiColors.lightMutedForeground,
      ),
    );
  }

  // ──────────────────────────────────────────────
  // Bottom Sheet
  // ──────────────────────────────────────────────

  static BottomSheetThemeData bottomSheetTheme(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return BottomSheetThemeData(
      backgroundColor: isDark ? StawiColors.darkBackground : StawiColors.lightBackground,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: const BorderRadius.vertical(top: Radius.circular(12)),
        side: BorderSide(
          color: isDark ? StawiColors.darkBorder : StawiColors.lightBorder,
        ),
      ),
      dragHandleColor: isDark ? StawiColors.darkMuted : StawiColors.lightMuted,
      showDragHandle: true,
    );
  }

  // ──────────────────────────────────────────────
  // Divider
  // ──────────────────────────────────────────────

  static DividerThemeData dividerTheme(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return DividerThemeData(
      color: isDark ? StawiColors.darkBorder : StawiColors.lightBorder,
      thickness: 1,
      space: 1,
    );
  }

  // ──────────────────────────────────────────────
  // Chip
  // ──────────────────────────────────────────────

  static ChipThemeData chipTheme(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return ChipThemeData(
      backgroundColor: isDark ? StawiColors.darkMuted : StawiColors.lightMuted,
      selectedColor: StawiColors.emerald.withValues(alpha: 0.15),
      disabledColor: isDark
          ? StawiColors.darkMuted.withValues(alpha: 0.5)
          : StawiColors.lightMuted.withValues(alpha: 0.5),
      labelStyle: TextStyle(
        fontSize: 13,
        fontWeight: FontWeight.w500,
        color: isDark ? StawiColors.darkForeground : StawiColors.lightForeground,
      ),
      side: BorderSide(
        color: isDark ? StawiColors.darkBorder : StawiColors.lightBorder,
      ),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(9999),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
    );
  }

  // ──────────────────────────────────────────────
  // Snackbar
  // ──────────────────────────────────────────────

  static SnackBarThemeData snackBarTheme(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return SnackBarThemeData(
      backgroundColor: isDark ? StawiColors.lightForeground : StawiColors.darkForeground,
      contentTextStyle: TextStyle(
        fontSize: 14,
        fontWeight: FontWeight.w500,
        color: isDark ? StawiColors.darkForeground : StawiColors.lightForeground,
      ),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(10),
      ),
      behavior: SnackBarBehavior.floating,
      elevation: 0,
    );
  }

  // ──────────────────────────────────────────────
  // TabBar
  // ──────────────────────────────────────────────

  static TabBarThemeData tabBarTheme(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return TabBarThemeData(
      indicatorColor: StawiColors.emerald,
      labelColor: isDark ? StawiColors.darkForeground : StawiColors.lightForeground,
      unselectedLabelColor:
          isDark ? StawiColors.darkMutedForeground : StawiColors.lightMutedForeground,
      indicatorSize: TabBarIndicatorSize.tab,
      dividerColor: isDark ? StawiColors.darkBorder : StawiColors.lightBorder,
      labelStyle: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
      unselectedLabelStyle: const TextStyle(fontSize: 14, fontWeight: FontWeight.w500),
    );
  }

  // ──────────────────────────────────────────────
  // Switch / Checkbox / Radio
  // ──────────────────────────────────────────────

  static SwitchThemeData switchTheme(Brightness brightness) {
    return SwitchThemeData(
      thumbColor: WidgetStateProperty.resolveWith((states) {
        if (states.contains(WidgetState.selected)) return Colors.white;
        return StawiColors.gray400;
      }),
      trackColor: WidgetStateProperty.resolveWith((states) {
        if (states.contains(WidgetState.selected)) return StawiColors.emerald;
        final isDark = brightness == Brightness.dark;
        return isDark ? StawiColors.darkMuted : StawiColors.lightMuted;
      }),
      trackOutlineColor: WidgetStateProperty.resolveWith((states) {
        if (states.contains(WidgetState.selected)) return Colors.transparent;
        final isDark = brightness == Brightness.dark;
        return isDark ? StawiColors.darkBorder : StawiColors.lightBorder;
      }),
    );
  }

  static CheckboxThemeData checkboxTheme(Brightness brightness) {
    return CheckboxThemeData(
      fillColor: WidgetStateProperty.resolveWith((states) {
        if (states.contains(WidgetState.selected)) return StawiColors.emerald;
        return Colors.transparent;
      }),
      checkColor: WidgetStateProperty.all(Colors.white),
      side: BorderSide(
        color: brightness == Brightness.dark
            ? StawiColors.darkBorder
            : StawiColors.lightBorder,
        width: 1.5,
      ),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(4),
      ),
    );
  }

  static RadioThemeData radioTheme(Brightness brightness) {
    return RadioThemeData(
      fillColor: WidgetStateProperty.resolveWith((states) {
        if (states.contains(WidgetState.selected)) return StawiColors.emerald;
        return brightness == Brightness.dark
            ? StawiColors.darkBorder
            : StawiColors.lightBorder;
      }),
    );
  }

  // ──────────────────────────────────────────────
  // Progress Indicator
  // ──────────────────────────────────────────────

  static ProgressIndicatorThemeData progressTheme(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return ProgressIndicatorThemeData(
      color: StawiColors.emerald,
      linearTrackColor: isDark ? StawiColors.darkMuted : StawiColors.lightMuted,
      circularTrackColor: isDark ? StawiColors.darkMuted : StawiColors.lightMuted,
    );
  }

  // ──────────────────────────────────────────────
  // Tooltip
  // ──────────────────────────────────────────────

  static TooltipThemeData tooltipTheme(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    return TooltipThemeData(
      decoration: BoxDecoration(
        color: isDark ? StawiColors.lightForeground : StawiColors.darkForeground,
        borderRadius: BorderRadius.circular(6),
      ),
      textStyle: TextStyle(
        fontSize: 12,
        fontWeight: FontWeight.w500,
        color: isDark ? StawiColors.darkForeground : StawiColors.lightForeground,
      ),
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
    );
  }
}
