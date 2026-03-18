import 'package:flutter/material.dart';

/// Complete color system based on BoundaryML's design tokens.
///
/// Provides both light and dark palettes with semantic naming.
/// Colors are mapped from the OKLCH-based CSS custom properties
/// used in the BoundaryML site.
abstract final class StawiColors {
  // ──────────────────────────────────────────────
  // Brand / Accent
  // ──────────────────────────────────────────────

  /// Primary emerald green accent.
  static const Color emerald = Color(0xFF10B981);
  static const Color emeraldLight = Color(0xFF34D399);
  static const Color emeraldDark = Color(0xFF059669);

  /// Secondary indigo accent.
  static const Color indigo = Color(0xFF6366F1);
  static const Color indigoLight = Color(0xFF818CF8);
  static const Color indigoDark = Color(0xFF4F46E5);

  /// Amber accent (warnings, highlights).
  static const Color amber = Color(0xFFF59E0B);
  static const Color amberLight = Color(0xFFFBBF24);
  static const Color amberDark = Color(0xFFD97706);

  /// Rose accent (errors, destructive).
  static const Color rose = Color(0xFFEF4444);
  static const Color roseLight = Color(0xFFF87171);
  static const Color roseDark = Color(0xFFDC2626);

  /// Sky blue accent (info, links).
  static const Color sky = Color(0xFF60A5FA);
  static const Color skyLight = Color(0xFF93C5FD);
  static const Color skyDark = Color(0xFF3B82F6);

  /// Purple accent (charts, highlights).
  static const Color purple = Color(0xFFC084FC);
  static const Color purpleLight = Color(0xFFD8B4FE);
  static const Color purpleDark = Color(0xFFA855F7);

  /// Pink accent (charts).
  static const Color pink = Color(0xFFF472B6);

  // ──────────────────────────────────────────────
  // Neutral palette
  // ──────────────────────────────────────────────

  static const Color white = Color(0xFFFFFFFF);
  static const Color black = Color(0xFF000000);

  static const Color gray50 = Color(0xFFFAFAFA);
  static const Color gray100 = Color(0xFFF5F5F5);
  static const Color gray200 = Color(0xFFE5E5E5);
  static const Color gray300 = Color(0xFFD4D4D4);
  static const Color gray400 = Color(0xFFA3A3A3);
  static const Color gray500 = Color(0xFF737373);
  static const Color gray600 = Color(0xFF525252);
  static const Color gray700 = Color(0xFF404040);
  static const Color gray800 = Color(0xFF262626);
  static const Color gray900 = Color(0xFF171717);
  static const Color gray950 = Color(0xFF0A0A0A);

  // ──────────────────────────────────────────────
  // Semantic – Light mode
  // ──────────────────────────────────────────────

  static const Color lightBackground = white;
  static const Color lightForeground = gray950;
  static const Color lightCard = white;
  static const Color lightCardForeground = gray950;
  static const Color lightPrimary = gray900;
  static const Color lightPrimaryForeground = gray50;
  static const Color lightSecondary = emerald;
  static const Color lightSecondaryForeground = white;
  static const Color lightMuted = gray100;
  static const Color lightMutedForeground = gray500;
  static const Color lightAccent = gray100;
  static const Color lightAccentForeground = gray900;
  static const Color lightBorder = gray200;
  static const Color lightInput = gray200;
  static const Color lightRing = gray950;
  static const Color lightDestructive = rose;
  static const Color lightDestructiveForeground = gray50;

  // ──────────────────────────────────────────────
  // Semantic – Dark mode
  // ──────────────────────────────────────────────

  static const Color darkBackground = black;
  static const Color darkForeground = gray50;
  static const Color darkCard = gray900;
  static const Color darkCardForeground = gray50;
  static const Color darkPrimary = gray50;
  static const Color darkPrimaryForeground = gray900;
  static const Color darkSecondary = emerald;
  static const Color darkSecondaryForeground = gray50;
  static const Color darkMuted = gray800;
  static const Color darkMutedForeground = gray300;
  static const Color darkAccent = gray800;
  static const Color darkAccentForeground = gray50;
  static const Color darkBorder = gray800;
  static const Color darkInput = gray800;
  static const Color darkRing = gray300;
  static const Color darkDestructive = roseDark;
  static const Color darkDestructiveForeground = gray50;

  // ──────────────────────────────────────────────
  // Chart palette
  // ──────────────────────────────────────────────

  static const List<Color> chartColors = [
    emerald,
    indigo,
    amber,
    purple,
    sky,
    pink,
    rose,
    Color(0xFF14B8A6), // teal
  ];

  // ──────────────────────────────────────────────
  // Gradient presets
  // ──────────────────────────────────────────────

  /// Hero text gradient (foreground fading to transparent).
  static LinearGradient heroTextGradient(Brightness brightness) {
    final base = brightness == Brightness.dark ? white : gray950;
    return LinearGradient(
      begin: Alignment.topLeft,
      end: Alignment.bottomRight,
      colors: [base, base.withValues(alpha: 0.4)],
      stops: const [0.3, 1.0],
    );
  }

  /// CTA section gradient.
  static const LinearGradient ctaGradient = LinearGradient(
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
    colors: [emeraldDark, indigo],
  );

  /// Logo icon gradient.
  static const LinearGradient brandGradient = LinearGradient(
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
    colors: [emerald, indigo],
  );

  /// Step number gradient.
  static const LinearGradient stepGradient = LinearGradient(
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
    colors: [emerald, indigo],
  );

  // ──────────────────────────────────────────────
  // Material ColorScheme factories
  // ──────────────────────────────────────────────

  /// Light mode [ColorScheme].
  static ColorScheme lightScheme() => const ColorScheme(
        brightness: Brightness.light,
        primary: lightPrimary,
        onPrimary: lightPrimaryForeground,
        secondary: lightSecondary,
        onSecondary: lightSecondaryForeground,
        error: lightDestructive,
        onError: lightDestructiveForeground,
        surface: lightBackground,
        onSurface: lightForeground,
        surfaceContainerHighest: lightCard,
        outline: lightBorder,
        outlineVariant: lightInput,
      );

  /// Dark mode [ColorScheme].
  static ColorScheme darkScheme() => const ColorScheme(
        brightness: Brightness.dark,
        primary: darkPrimary,
        onPrimary: darkPrimaryForeground,
        secondary: darkSecondary,
        onSecondary: darkSecondaryForeground,
        error: darkDestructive,
        onError: darkDestructiveForeground,
        surface: darkBackground,
        onSurface: darkForeground,
        surfaceContainerHighest: darkCard,
        outline: darkBorder,
        outlineVariant: darkInput,
      );
}
