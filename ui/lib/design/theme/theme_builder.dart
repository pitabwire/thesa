/// Theme builder for constructing ThemeData from design tokens and branding.
library;

import 'package:flutter/material.dart';

import '../branding/branding_config.dart';
import '../tokens/colors.dart';
import '../tokens/typography.dart';

/// Builds Material ThemeData from design tokens
class ThemeBuilder {
  ThemeBuilder._();

  /// Build light theme
  static ThemeData buildLightTheme({
    BrandingConfig branding = BrandingConfig.defaultBranding,
  }) {
    final primaryColor = branding.primaryColor ?? AppColors.lightPrimary;
    final secondaryColor = branding.secondaryColor ?? AppColors.lightSecondary;

    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.light,
      colorScheme: ColorScheme.light(
        primary: primaryColor,
        onPrimary: AppColors.lightOnPrimary,
        secondary: secondaryColor,
        onSecondary: AppColors.lightOnSecondary,
        surface: AppColors.lightSurface,
        onSurface: AppColors.lightOnSurface,
        background: AppColors.lightBackground,
        onBackground: AppColors.lightOnBackground,
        error: AppColors.lightError,
        onError: AppColors.lightOnError,
      ),
      // Typography
      textTheme: _buildTextTheme(
        baseTextTheme: Typography.material2021().black,
        fontFamily: branding.fontFamily,
      ),
      // Apply font family
      fontFamily: branding.fontFamily,
      // Card theme
      cardTheme: CardTheme(
        elevation: 1,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
        ),
      ),
      // AppBar theme
      appBarTheme: AppBarTheme(
        elevation: 0,
        centerTitle: false,
        backgroundColor: primaryColor,
        foregroundColor: AppColors.lightOnPrimary,
      ),
      // Input decoration theme
      inputDecorationTheme: InputDecorationTheme(
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
        ),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 16,
          vertical: 12,
        ),
      ),
      // Button themes
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          padding: const EdgeInsets.symmetric(
            horizontal: 24,
            vertical: 12,
          ),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
          ),
        ),
      ),
      // Divider theme
      dividerTheme: const DividerThemeData(
        color: AppColors.lightDivider,
        thickness: 1,
      ),
    );
  }

  /// Build dark theme
  static ThemeData buildDarkTheme({
    BrandingConfig branding = BrandingConfig.defaultBranding,
  }) {
    final primaryColor = branding.primaryColor != null
        ? _adjustColorForDarkMode(branding.primaryColor!)
        : AppColors.darkPrimary;
    final secondaryColor = branding.secondaryColor != null
        ? _adjustColorForDarkMode(branding.secondaryColor!)
        : AppColors.darkSecondary;

    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.dark,
      colorScheme: ColorScheme.dark(
        primary: primaryColor,
        onPrimary: AppColors.darkOnPrimary,
        secondary: secondaryColor,
        onSecondary: AppColors.darkOnSecondary,
        surface: AppColors.darkSurface,
        onSurface: AppColors.darkOnSurface,
        background: AppColors.darkBackground,
        onBackground: AppColors.darkOnBackground,
        error: AppColors.darkError,
        onError: AppColors.darkOnError,
      ),
      // Typography
      textTheme: _buildTextTheme(
        baseTextTheme: Typography.material2021().white,
        fontFamily: branding.fontFamily,
      ),
      // Apply font family
      fontFamily: branding.fontFamily,
      // Card theme
      cardTheme: CardTheme(
        elevation: 2,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
        ),
      ),
      // AppBar theme
      appBarTheme: AppBarTheme(
        elevation: 0,
        centerTitle: false,
        backgroundColor: primaryColor,
        foregroundColor: AppColors.darkOnPrimary,
      ),
      // Input decoration theme
      inputDecorationTheme: InputDecorationTheme(
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
        ),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 16,
          vertical: 12,
        ),
      ),
      // Button themes
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          padding: const EdgeInsets.symmetric(
            horizontal: 24,
            vertical: 12,
          ),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
          ),
        ),
      ),
      // Divider theme
      dividerTheme: const DividerThemeData(
        color: AppColors.darkDivider,
        thickness: 1,
      ),
    );
  }

  /// Build text theme with custom font family
  static TextTheme _buildTextTheme({
    required TextTheme baseTextTheme,
    String? fontFamily,
  }) {
    if (fontFamily == null) {
      return baseTextTheme;
    }

    return baseTextTheme.apply(fontFamily: fontFamily);
  }

  /// Adjust brand color for dark mode
  ///
  /// Lightens colors for better visibility on dark backgrounds.
  static Color _adjustColorForDarkMode(Color color) {
    final hsl = HSLColor.fromColor(color);

    // Increase lightness for dark mode
    return hsl.withLightness((hsl.lightness * 1.4).clamp(0.0, 1.0)).toColor();
  }
}
