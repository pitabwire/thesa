import 'package:flutter/material.dart';
import 'button_styles.dart';
import 'card_styles.dart';
import 'colors.dart';
import 'component_styles.dart';
import 'input_styles.dart';
import 'theme_extensions.dart';
import 'typography.dart';

/// Main entry point for the Stawi theme.
///
/// ```dart
/// MaterialApp(
///   theme: StawiTheme.light(),
///   darkTheme: StawiTheme.dark(),
///   themeMode: ThemeMode.dark,
/// )
/// ```
///
/// Override any color by passing a custom [StawiColorTokens]:
/// ```dart
/// StawiTheme.dark(
///   overrideTokens: StawiColorTokens.dark.copyWith(
///     secondary: Colors.blue,
///   ),
/// )
/// ```
abstract final class StawiTheme {
  /// Creates the dark [ThemeData].
  static ThemeData dark({
    StawiColorTokens? overrideTokens,
    StawiSpacing? overrideSpacing,
  }) =>
      _build(
        Brightness.dark,
        overrideTokens ?? StawiColorTokens.dark,
        overrideSpacing ?? const StawiSpacing(),
      );

  /// Creates the light [ThemeData].
  static ThemeData light({
    StawiColorTokens? overrideTokens,
    StawiSpacing? overrideSpacing,
  }) =>
      _build(
        Brightness.light,
        overrideTokens ?? StawiColorTokens.light,
        overrideSpacing ?? const StawiSpacing(),
      );

  static ThemeData _build(
    Brightness brightness,
    StawiColorTokens tokens,
    StawiSpacing spacing,
  ) {
    final colorScheme = brightness == Brightness.dark
        ? StawiColors.darkScheme()
        : StawiColors.lightScheme();

    final textTheme = StawiTypography.textTheme(brightness);

    return ThemeData(
      useMaterial3: true,
      brightness: brightness,
      colorScheme: colorScheme,
      textTheme: textTheme,
      scaffoldBackgroundColor: tokens.background,
      canvasColor: tokens.background,

      // ── Extensions ────────────────────────────
      extensions: [tokens, spacing],

      // ── AppBar ────────────────────────────────
      appBarTheme: StawiComponentStyles.appBarTheme(brightness),

      // ── Cards ─────────────────────────────────
      cardTheme: StawiCardStyles.cardTheme(brightness),

      // ── Buttons ───────────────────────────────
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: StawiButtonStyles.primary(brightness),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: StawiButtonStyles.outline(brightness),
      ),
      textButtonTheme: TextButtonThemeData(
        style: StawiButtonStyles.ghost(brightness),
      ),
      iconButtonTheme: IconButtonThemeData(
        style: StawiButtonStyles.iconOutline(brightness),
      ),

      // ── Inputs ────────────────────────────────
      inputDecorationTheme: StawiInputStyles.inputTheme(brightness),

      // ── Navigation ────────────────────────────
      navigationBarTheme:
          StawiComponentStyles.navigationBarTheme(brightness),
      navigationRailTheme:
          StawiComponentStyles.navigationRailTheme(brightness),

      // ── Dialogs / Sheets ──────────────────────
      dialogTheme: StawiComponentStyles.dialogTheme(brightness),
      bottomSheetTheme: StawiComponentStyles.bottomSheetTheme(brightness),

      // ── Misc Components ───────────────────────
      dividerTheme: StawiComponentStyles.dividerTheme(brightness),
      chipTheme: StawiComponentStyles.chipTheme(brightness),
      snackBarTheme: StawiComponentStyles.snackBarTheme(brightness),
      tabBarTheme: StawiComponentStyles.tabBarTheme(brightness),
      switchTheme: StawiComponentStyles.switchTheme(brightness),
      checkboxTheme: StawiComponentStyles.checkboxTheme(brightness),
      radioTheme: StawiComponentStyles.radioTheme(brightness),
      progressIndicatorTheme:
          StawiComponentStyles.progressTheme(brightness),
      tooltipTheme: StawiComponentStyles.tooltipTheme(brightness),

      // ── Scrollbar ─────────────────────────────
      scrollbarTheme: ScrollbarThemeData(
        thumbColor: WidgetStateProperty.all(
          tokens.mutedForeground.withValues(alpha: 0.3),
        ),
        radius: const Radius.circular(9999),
        thickness: WidgetStateProperty.all(6),
      ),

      // ── Splash / ink ──────────────────────────
      splashFactory: InkSparkle.splashFactory,
    );
  }
}

/// Convenience extension to access [StawiColorTokens] and [StawiSpacing] from
/// any [BuildContext].
///
/// ```dart
/// final tokens = context.stawiColors;
/// final spacing = context.stawiSpacing;
/// ```
extension StawiThemeContext on BuildContext {
  /// Shorthand for `Theme.of(this).extension<StawiColorTokens>()!`.
  StawiColorTokens get stawiColors =>
      Theme.of(this).extension<StawiColorTokens>()!;

  /// Shorthand for `Theme.of(this).extension<StawiSpacing>()!`.
  StawiSpacing get stawiSpacing =>
      Theme.of(this).extension<StawiSpacing>()!;

  /// Whether the current theme is dark.
  bool get isDarkMode => Theme.of(this).brightness == Brightness.dark;
}
