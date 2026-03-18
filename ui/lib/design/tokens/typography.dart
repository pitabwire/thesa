/// Typography design tokens for the Thesa UI design system.
///
/// Defines the text style hierarchy used throughout the app.
library;

import 'package:flutter/material.dart';

/// Typography scale following Material Design 3 naming
class AppTypography {
  const AppTypography._();

  /// Font family - defaults to system font
  /// Can be overridden for enterprise branding
  static const String fontFamily = 'Roboto'; // System default

  // ============================================================
  // DISPLAY STYLES (largest text)
  // ============================================================

  /// Display large - 32px bold
  /// Usage: Main page titles on dashboards
  static const displayLarge = TextStyle(
    fontFamily: fontFamily,
    fontSize: 32,
    height: 1.25, // line-height: 40px
    fontWeight: FontWeight.w700, // Bold
    letterSpacing: 0,
  );

  // ============================================================
  // HEADLINE STYLES
  // ============================================================

  /// Headline medium - 24px semi-bold
  /// Usage: Page titles
  static const headlineMedium = TextStyle(
    fontFamily: fontFamily,
    fontSize: 24,
    height: 1.33, // line-height: 32px
    fontWeight: FontWeight.w600, // Semi-bold
    letterSpacing: 0,
  );

  // ============================================================
  // TITLE STYLES
  // ============================================================

  /// Title large - 20px semi-bold
  /// Usage: Section headings, card titles
  static const titleLarge = TextStyle(
    fontFamily: fontFamily,
    fontSize: 20,
    height: 1.4, // line-height: 28px
    fontWeight: FontWeight.w600, // Semi-bold
    letterSpacing: 0,
  );

  /// Title medium - 16px semi-bold
  /// Usage: Sub-section headings
  static const titleMedium = TextStyle(
    fontFamily: fontFamily,
    fontSize: 16,
    height: 1.5, // line-height: 24px
    fontWeight: FontWeight.w600, // Semi-bold
    letterSpacing: 0.15,
  );

  // ============================================================
  // BODY STYLES
  // ============================================================

  /// Body large - 16px regular
  /// Usage: Primary body text
  static const bodyLarge = TextStyle(
    fontFamily: fontFamily,
    fontSize: 16,
    height: 1.5, // line-height: 24px
    fontWeight: FontWeight.w400, // Regular
    letterSpacing: 0.5,
  );

  /// Body medium - 14px regular
  /// Usage: Secondary body text, table cells
  static const bodyMedium = TextStyle(
    fontFamily: fontFamily,
    fontSize: 14,
    height: 1.43, // line-height: 20px
    fontWeight: FontWeight.w400, // Regular
    letterSpacing: 0.25,
  );

  /// Body small - 12px regular
  /// Usage: Captions, timestamps, help text
  static const bodySmall = TextStyle(
    fontFamily: fontFamily,
    fontSize: 12,
    height: 1.33, // line-height: 16px
    fontWeight: FontWeight.w400, // Regular
    letterSpacing: 0.4,
  );

  // ============================================================
  // LABEL STYLES
  // ============================================================

  /// Label large - 14px medium
  /// Usage: Button text
  static const labelLarge = TextStyle(
    fontFamily: fontFamily,
    fontSize: 14,
    height: 1.43, // line-height: 20px
    fontWeight: FontWeight.w500, // Medium
    letterSpacing: 0.1,
  );

  /// Label medium - 12px medium
  /// Usage: Badge text, tag text
  static const labelMedium = TextStyle(
    fontFamily: fontFamily,
    fontSize: 12,
    height: 1.33, // line-height: 16px
    fontWeight: FontWeight.w500, // Medium
    letterSpacing: 0.5,
  );

  /// Label small - 11px medium
  /// Usage: Overline text, tiny labels
  static const labelSmall = TextStyle(
    fontFamily: fontFamily,
    fontSize: 11,
    height: 1.45, // line-height: 16px
    fontWeight: FontWeight.w500, // Medium
    letterSpacing: 0.5,
  );
}
