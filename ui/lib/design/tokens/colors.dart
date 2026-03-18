/// Color design tokens for the Thesa UI design system.
///
/// All colors have light and dark mode variants.
/// Use semantic names (colorPrimary) not descriptive names (colorBlue).
library;

import 'package:flutter/material.dart';

/// Color tokens organized by light/dark theme
class AppColors {
  const AppColors._();

  // ============================================================
  // LIGHT MODE COLORS
  // ============================================================

  static const lightPrimary = Color(0xFF1565C0); // Blue
  static const lightOnPrimary = Color(0xFFFFFFFF); // White
  static const lightSecondary = Color(0xFF7B1FA2); // Purple
  static const lightOnSecondary = Color(0xFFFFFFFF); // White
  static const lightSurface = Color(0xFFFFFFFF); // White
  static const lightOnSurface = Color(0xFF212121); // Near-black
  static const lightBackground = Color(0xFFFAFAFA); // Light grey
  static const lightOnBackground = Color(0xFF212121); // Near-black
  static const lightError = Color(0xFFD32F2F); // Red
  static const lightOnError = Color(0xFFFFFFFF); // White
  static const lightSuccess = Color(0xFF388E3C); // Green
  static const lightOnSuccess = Color(0xFFFFFFFF); // White
  static const lightWarning = Color(0xFFF57C00); // Orange
  static const lightOnWarning = Color(0xFFFFFFFF); // White
  static const lightInfo = Color(0xFF1976D2); // Blue
  static const lightOnInfo = Color(0xFFFFFFFF); // White

  // Border and divider colors
  static const lightBorder = Color(0xFFE0E0E0); // Light grey
  static const lightDivider = Color(0xFFBDBDBD); // Medium grey

  // Overlay colors (for hover, focus, etc.)
  static const lightHoverOverlay = Color(0x0A000000); // 4% black
  static const lightFocusOverlay = Color(0x1F000000); // 12% black
  static const lightPressedOverlay = Color(0x29000000); // 16% black

  // ============================================================
  // DARK MODE COLORS
  // ============================================================

  static const darkPrimary = Color(0xFF90CAF9); // Light blue
  static const darkOnPrimary = Color(0xFF0D1B2A); // Dark blue
  static const darkSecondary = Color(0xFFCE93D8); // Light purple
  static const darkOnSecondary = Color(0xFF2D0A37); // Dark purple
  static const darkSurface = Color(0xFF1E1E1E); // Dark grey
  static const darkOnSurface = Color(0xFFE0E0E0); // Light grey
  static const darkBackground = Color(0xFF121212); // Near-black
  static const darkOnBackground = Color(0xFFE0E0E0); // Light grey
  static const darkError = Color(0xFFEF9A9A); // Light red
  static const darkOnError = Color(0xFF1B0000); // Dark red
  static const darkSuccess = Color(0xFFA5D6A7); // Light green
  static const darkOnSuccess = Color(0xFF0A1F0B); // Dark green
  static const darkWarning = Color(0xFFFFB74D); // Light orange
  static const darkOnWarning = Color(0xFF2E1500); // Dark orange
  static const darkInfo = Color(0xFF64B5F6); // Light blue
  static const darkOnInfo = Color(0xFF001D35); // Dark blue

  // Border and divider colors
  static const darkBorder = Color(0xFF3E3E3E); // Medium grey
  static const darkDivider = Color(0xFF5E5E5E); // Lighter grey

  // Overlay colors (for hover, focus, etc.)
  static const darkHoverOverlay = Color(0x0AFFFFFF); // 4% white
  static const darkFocusOverlay = Color(0x1FFFFFFF); // 12% white
  static const darkPressedOverlay = Color(0x29FFFFFF); // 16% white

  // ============================================================
  // STATUS COLORS (used for badges, chips, etc.)
  // ============================================================

  /// Get status color based on status value
  static Color getStatusColor(
    String status, {
    required bool isDark,
  }) {
    switch (status.toLowerCase()) {
      case 'completed':
      case 'delivered':
      case 'success':
      case 'approved':
        return isDark ? darkSuccess : lightSuccess;

      case 'active':
      case 'processing':
      case 'shipped':
      case 'in_transit':
        return isDark ? darkInfo : lightInfo;

      case 'pending':
      case 'waiting':
      case 'review':
        return isDark ? darkWarning : lightWarning;

      case 'cancelled':
      case 'failed':
      case 'rejected':
      case 'error':
        return isDark ? darkError : lightError;

      case 'draft':
      case 'inactive':
        return isDark ? darkOnSurface : lightOnSurface;

      default:
        // Unknown status - use neutral color
        return isDark ? darkOnSurface : lightOnSurface;
    }
  }

  /// Get color for status text (contrast color)
  static Color getStatusTextColor(
    String status, {
    required bool isDark,
  }) {
    final bgColor = getStatusColor(status, isDark: isDark);

    // For light backgrounds, use dark text
    // For dark backgrounds, use light text
    final luminance = bgColor.computeLuminance();
    return luminance > 0.5
        ? (isDark ? darkOnSurface : lightOnSurface)
        : Color(0xFFFFFFFF);
  }
}
