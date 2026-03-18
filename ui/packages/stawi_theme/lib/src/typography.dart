import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

/// Typography system matching BoundaryML's type scale.
///
/// Uses Inter as the primary font (closest match to Geist Sans)
/// and JetBrains Mono for monospace / code.
abstract final class StawiTypography {
  /// Base text theme built on Inter.
  static TextTheme textTheme(Brightness brightness) {
    final color =
        brightness == Brightness.dark ? const Color(0xFFFAFAFA) : const Color(0xFF0A0A0A);

    return GoogleFonts.interTextTheme(
      TextTheme(
        // Display — hero headlines
        displayLarge: TextStyle(
          fontSize: 64,
          fontWeight: FontWeight.w700,
          letterSpacing: -2.5,
          height: 1.05,
          color: color,
        ),
        displayMedium: TextStyle(
          fontSize: 48,
          fontWeight: FontWeight.w700,
          letterSpacing: -2.0,
          height: 1.08,
          color: color,
        ),
        displaySmall: TextStyle(
          fontSize: 36,
          fontWeight: FontWeight.w600,
          letterSpacing: -1.5,
          height: 1.1,
          color: color,
        ),

        // Headline — section titles
        headlineLarge: TextStyle(
          fontSize: 32,
          fontWeight: FontWeight.w600,
          letterSpacing: -1.2,
          height: 1.15,
          color: color,
        ),
        headlineMedium: TextStyle(
          fontSize: 28,
          fontWeight: FontWeight.w600,
          letterSpacing: -0.8,
          height: 1.2,
          color: color,
        ),
        headlineSmall: TextStyle(
          fontSize: 24,
          fontWeight: FontWeight.w600,
          letterSpacing: -0.5,
          height: 1.25,
          color: color,
        ),

        // Title — card titles, nav items
        titleLarge: TextStyle(
          fontSize: 20,
          fontWeight: FontWeight.w600,
          letterSpacing: -0.3,
          height: 1.3,
          color: color,
        ),
        titleMedium: TextStyle(
          fontSize: 16,
          fontWeight: FontWeight.w600,
          letterSpacing: -0.1,
          height: 1.4,
          color: color,
        ),
        titleSmall: TextStyle(
          fontSize: 14,
          fontWeight: FontWeight.w600,
          letterSpacing: 0,
          height: 1.4,
          color: color,
        ),

        // Body — primary content
        bodyLarge: TextStyle(
          fontSize: 17,
          fontWeight: FontWeight.w400,
          letterSpacing: 0,
          height: 1.7,
          color: color,
        ),
        bodyMedium: TextStyle(
          fontSize: 15,
          fontWeight: FontWeight.w400,
          letterSpacing: 0,
          height: 1.6,
          color: color,
        ),
        bodySmall: TextStyle(
          fontSize: 13,
          fontWeight: FontWeight.w400,
          letterSpacing: 0,
          height: 1.5,
          color: color,
        ),

        // Label — buttons, badges, form labels
        labelLarge: TextStyle(
          fontSize: 14,
          fontWeight: FontWeight.w600,
          letterSpacing: 0,
          height: 1.4,
          color: color,
        ),
        labelMedium: TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.w500,
          letterSpacing: 0.3,
          height: 1.4,
          color: color,
        ),
        labelSmall: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.w500,
          letterSpacing: 0.5,
          height: 1.4,
          color: color,
        ),
      ),
    );
  }

  /// Monospace text style for code blocks and terminals.
  static TextStyle mono({
    double fontSize = 14,
    FontWeight fontWeight = FontWeight.w400,
    Color? color,
    double height = 1.8,
  }) {
    return GoogleFonts.jetBrainsMono(
      fontSize: fontSize,
      fontWeight: fontWeight,
      color: color,
      height: height,
    );
  }

  /// Label text in uppercase with letter spacing (for section subtitles).
  static TextStyle overline({
    double fontSize = 12,
    Color? color,
  }) {
    return GoogleFonts.inter(
      fontSize: fontSize,
      fontWeight: FontWeight.w500,
      letterSpacing: 1.0,
      color: color,
    );
  }
}
