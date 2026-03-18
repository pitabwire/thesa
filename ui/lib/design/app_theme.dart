/// Application theme configuration.
///
/// Builds Flutter ThemeData from design tokens.
library;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'tokens/colors.dart';
import 'tokens/spacing.dart';
import 'tokens/typography.dart';

/// Application theme builder
class AppTheme {
  const AppTheme._();

  /// Build light theme
  static ThemeData light() {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.light,

      // Color scheme
      colorScheme: const ColorScheme(
        brightness: Brightness.light,
        primary: AppColors.lightPrimary,
        onPrimary: AppColors.lightOnPrimary,
        secondary: AppColors.lightSecondary,
        onSecondary: AppColors.lightOnSecondary,
        error: AppColors.lightError,
        onError: AppColors.lightOnError,
        surface: AppColors.lightSurface,
        onSurface: AppColors.lightOnSurface,
      ),

      // Background color
      scaffoldBackgroundColor: AppColors.lightBackground,

      // App bar theme
      appBarTheme: const AppBarTheme(
        backgroundColor: AppColors.lightPrimary,
        foregroundColor: AppColors.lightOnPrimary,
        elevation: AppElevation.low,
        centerTitle: false,
        systemOverlayStyle: SystemUiOverlayStyle.light, // Light status bar icons
      ),

      // Card theme
      cardTheme: CardThemeData(
        color: AppColors.lightSurface,
        elevation: AppElevation.medium,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppBorderRadius.medium),
        ),
        margin: const EdgeInsets.all(AppSpacing.space8),
      ),

      // Elevated button theme
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: AppColors.lightPrimary,
          foregroundColor: AppColors.lightOnPrimary,
          elevation: AppElevation.low,
          padding: const EdgeInsets.symmetric(
            horizontal: AppSpacing.space16,
            vertical: AppSpacing.space12,
          ),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppBorderRadius.medium),
          ),
          minimumSize: const Size(
            AppSizing.minTouchTarget,
            AppSizing.buttonMedium,
          ),
          textStyle: AppTypography.labelLarge,
        ),
      ),

      // Outlined button theme
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: AppColors.lightPrimary,
          side: const BorderSide(color: AppColors.lightPrimary),
          padding: const EdgeInsets.symmetric(
            horizontal: AppSpacing.space16,
            vertical: AppSpacing.space12,
          ),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppBorderRadius.medium),
          ),
          minimumSize: const Size(
            AppSizing.minTouchTarget,
            AppSizing.buttonMedium,
          ),
          textStyle: AppTypography.labelLarge,
        ),
      ),

      // Text button theme
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          foregroundColor: AppColors.lightPrimary,
          padding: const EdgeInsets.symmetric(
            horizontal: AppSpacing.space16,
            vertical: AppSpacing.space12,
          ),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppBorderRadius.medium),
          ),
          minimumSize: const Size(
            AppSizing.minTouchTarget,
            AppSizing.buttonMedium,
          ),
          textStyle: AppTypography.labelLarge,
        ),
      ),

      // Input decoration theme
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: AppColors.lightSurface,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppBorderRadius.medium),
          borderSide: const BorderSide(color: AppColors.lightBorder),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppBorderRadius.medium),
          borderSide: const BorderSide(color: AppColors.lightBorder),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppBorderRadius.medium),
          borderSide: const BorderSide(color: AppColors.lightPrimary, width: 2),
        ),
        errorBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppBorderRadius.medium),
          borderSide: const BorderSide(color: AppColors.lightError),
        ),
        focusedErrorBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppBorderRadius.medium),
          borderSide: const BorderSide(color: AppColors.lightError, width: 2),
        ),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: AppSpacing.space16,
          vertical: AppSpacing.space12,
        ),
        hintStyle: AppTypography.bodyMedium.copyWith(
          color: AppColors.lightOnSurface.withOpacity(0.6),
        ),
      ),

      // Chip theme
      chipTheme: ChipThemeData(
        backgroundColor: AppColors.lightSurface,
        selectedColor: AppColors.lightPrimary.withOpacity(0.12),
        labelStyle: AppTypography.labelMedium,
        padding: const EdgeInsets.symmetric(
          horizontal: AppSpacing.space12,
          vertical: AppSpacing.space8,
        ),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppBorderRadius.small),
          side: const BorderSide(color: AppColors.lightBorder),
        ),
      ),

      // Dialog theme
      dialogTheme: DialogThemeData(
        backgroundColor: AppColors.lightSurface,
        elevation: AppElevation.high,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppBorderRadius.large),
        ),
      ),

      // Divider theme
      dividerTheme: const DividerThemeData(
        color: AppColors.lightDivider,
        space: AppSpacing.space16,
        thickness: 1,
      ),

      // Typography
      textTheme: TextTheme(
        displayLarge: AppTypography.displayLarge.copyWith(
          color: AppColors.lightOnBackground,
        ),
        headlineMedium: AppTypography.headlineMedium.copyWith(
          color: AppColors.lightOnBackground,
        ),
        titleLarge: AppTypography.titleLarge.copyWith(
          color: AppColors.lightOnSurface,
        ),
        titleMedium: AppTypography.titleMedium.copyWith(
          color: AppColors.lightOnSurface,
        ),
        bodyLarge: AppTypography.bodyLarge.copyWith(
          color: AppColors.lightOnSurface,
        ),
        bodyMedium: AppTypography.bodyMedium.copyWith(
          color: AppColors.lightOnSurface,
        ),
        bodySmall: AppTypography.bodySmall.copyWith(
          color: AppColors.lightOnSurface.withOpacity(0.6),
        ),
        labelLarge: AppTypography.labelLarge.copyWith(
          color: AppColors.lightOnSurface,
        ),
        labelMedium: AppTypography.labelMedium.copyWith(
          color: AppColors.lightOnSurface,
        ),
        labelSmall: AppTypography.labelSmall.copyWith(
          color: AppColors.lightOnSurface.withOpacity(0.6),
        ),
      ),

      // Icon theme
      iconTheme: const IconThemeData(
        color: AppColors.lightOnSurface,
        size: AppSizing.iconMedium,
      ),

      // Focus color
      focusColor: AppColors.lightFocusOverlay,
      hoverColor: AppColors.lightHoverOverlay,
      highlightColor: AppColors.lightPressedOverlay,
    );
  }

  /// Build dark theme
  static ThemeData dark() {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.dark,

      // Color scheme
      colorScheme: const ColorScheme(
        brightness: Brightness.dark,
        primary: AppColors.darkPrimary,
        onPrimary: AppColors.darkOnPrimary,
        secondary: AppColors.darkSecondary,
        onSecondary: AppColors.darkOnSecondary,
        error: AppColors.darkError,
        onError: AppColors.darkOnError,
        surface: AppColors.darkSurface,
        onSurface: AppColors.darkOnSurface,
      ),

      // Background color
      scaffoldBackgroundColor: AppColors.darkBackground,

      // App bar theme
      appBarTheme: const AppBarTheme(
        backgroundColor: AppColors.darkSurface,
        foregroundColor: AppColors.darkOnSurface,
        elevation: AppElevation.low,
        centerTitle: false,
        systemOverlayStyle: SystemUiOverlayStyle.dark, // Dark status bar icons
      ),

      // Card theme
      cardTheme: CardThemeData(
        color: AppColors.darkSurface,
        elevation: AppElevation.medium,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppBorderRadius.medium),
        ),
        margin: const EdgeInsets.all(AppSpacing.space8),
      ),

      // Elevated button theme
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: AppColors.darkPrimary,
          foregroundColor: AppColors.darkOnPrimary,
          elevation: AppElevation.low,
          padding: const EdgeInsets.symmetric(
            horizontal: AppSpacing.space16,
            vertical: AppSpacing.space12,
          ),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppBorderRadius.medium),
          ),
          minimumSize: const Size(
            AppSizing.minTouchTarget,
            AppSizing.buttonMedium,
          ),
          textStyle: AppTypography.labelLarge,
        ),
      ),

      // Outlined button theme
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: AppColors.darkPrimary,
          side: const BorderSide(color: AppColors.darkPrimary),
          padding: const EdgeInsets.symmetric(
            horizontal: AppSpacing.space16,
            vertical: AppSpacing.space12,
          ),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppBorderRadius.medium),
          ),
          minimumSize: const Size(
            AppSizing.minTouchTarget,
            AppSizing.buttonMedium,
          ),
          textStyle: AppTypography.labelLarge,
        ),
      ),

      // Text button theme
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          foregroundColor: AppColors.darkPrimary,
          padding: const EdgeInsets.symmetric(
            horizontal: AppSpacing.space16,
            vertical: AppSpacing.space12,
          ),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppBorderRadius.medium),
          ),
          minimumSize: const Size(
            AppSizing.minTouchTarget,
            AppSizing.buttonMedium,
          ),
          textStyle: AppTypography.labelLarge,
        ),
      ),

      // Input decoration theme
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: AppColors.darkSurface,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppBorderRadius.medium),
          borderSide: const BorderSide(color: AppColors.darkBorder),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppBorderRadius.medium),
          borderSide: const BorderSide(color: AppColors.darkBorder),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppBorderRadius.medium),
          borderSide: const BorderSide(color: AppColors.darkPrimary, width: 2),
        ),
        errorBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppBorderRadius.medium),
          borderSide: const BorderSide(color: AppColors.darkError),
        ),
        focusedErrorBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppBorderRadius.medium),
          borderSide: const BorderSide(color: AppColors.darkError, width: 2),
        ),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: AppSpacing.space16,
          vertical: AppSpacing.space12,
        ),
        hintStyle: AppTypography.bodyMedium.copyWith(
          color: AppColors.darkOnSurface.withOpacity(0.6),
        ),
      ),

      // Chip theme
      chipTheme: ChipThemeData(
        backgroundColor: AppColors.darkSurface,
        selectedColor: AppColors.darkPrimary.withOpacity(0.24),
        labelStyle: AppTypography.labelMedium,
        padding: const EdgeInsets.symmetric(
          horizontal: AppSpacing.space12,
          vertical: AppSpacing.space8,
        ),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppBorderRadius.small),
          side: const BorderSide(color: AppColors.darkBorder),
        ),
      ),

      // Dialog theme
      dialogTheme: DialogThemeData(
        backgroundColor: AppColors.darkSurface,
        elevation: AppElevation.high,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppBorderRadius.large),
        ),
      ),

      // Divider theme
      dividerTheme: const DividerThemeData(
        color: AppColors.darkDivider,
        space: AppSpacing.space16,
        thickness: 1,
      ),

      // Typography
      textTheme: TextTheme(
        displayLarge: AppTypography.displayLarge.copyWith(
          color: AppColors.darkOnBackground,
        ),
        headlineMedium: AppTypography.headlineMedium.copyWith(
          color: AppColors.darkOnBackground,
        ),
        titleLarge: AppTypography.titleLarge.copyWith(
          color: AppColors.darkOnSurface,
        ),
        titleMedium: AppTypography.titleMedium.copyWith(
          color: AppColors.darkOnSurface,
        ),
        bodyLarge: AppTypography.bodyLarge.copyWith(
          color: AppColors.darkOnSurface,
        ),
        bodyMedium: AppTypography.bodyMedium.copyWith(
          color: AppColors.darkOnSurface,
        ),
        bodySmall: AppTypography.bodySmall.copyWith(
          color: AppColors.darkOnSurface.withOpacity(0.6),
        ),
        labelLarge: AppTypography.labelLarge.copyWith(
          color: AppColors.darkOnSurface,
        ),
        labelMedium: AppTypography.labelMedium.copyWith(
          color: AppColors.darkOnSurface,
        ),
        labelSmall: AppTypography.labelSmall.copyWith(
          color: AppColors.darkOnSurface.withOpacity(0.6),
        ),
      ),

      // Icon theme
      iconTheme: const IconThemeData(
        color: AppColors.darkOnSurface,
        size: AppSizing.iconMedium,
      ),

      // Focus color
      focusColor: AppColors.darkFocusOverlay,
      hoverColor: AppColors.darkHoverOverlay,
      highlightColor: AppColors.darkPressedOverlay,
    );
  }
}
