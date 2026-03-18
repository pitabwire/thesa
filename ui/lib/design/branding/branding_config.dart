/// Enterprise branding configuration model.
library;

import 'package:flutter/material.dart';
import 'package:freezed_annotation/freezed_annotation.dart';

part 'branding_config.freezed.dart';
part 'branding_config.g.dart';

/// Enterprise branding configuration
///
/// Allows deployments to customize:
/// - Primary and secondary colors
/// - Logo asset path
/// - Font family
/// - Default theme mode
/// - Login background image
///
/// What is NOT customizable (hardcoded for consistency):
/// - Status colors (red=error, green=success, etc.)
/// - Spacing scale
/// - Typography scale
@freezed
class BrandingConfig with _$BrandingConfig {
  const factory BrandingConfig({
    /// Primary brand color hex (e.g., '#1565C0')
    String? primaryColorHex,

    /// Secondary brand color hex (e.g., '#7B1FA2')
    String? secondaryColorHex,

    /// Path to logo asset (e.g., 'assets/images/logo.png')
    String? logoAssetPath,

    /// Font family name (must be registered in pubspec.yaml)
    String? fontFamily,

    /// Default theme mode (light, dark, or system)
    @Default(ThemeMode.system) ThemeMode defaultThemeMode,

    /// Path to login background image
    String? loginBackgroundPath,

    /// App title override
    String? appTitle,

    /// Whether to show branding in sidebar
    @Default(true) bool showBrandingInSidebar,
  }) = _BrandingConfig;

  const BrandingConfig._();

  /// Get primary color from hex string
  Color? get primaryColor =>
      primaryColorHex != null ? _parseColor(primaryColorHex!) : null;

  /// Get secondary color from hex string
  Color? get secondaryColor =>
      secondaryColorHex != null ? _parseColor(secondaryColorHex!) : null;

  /// Parse hex color string (e.g., '#1565C0' or '1565C0')
  static Color _parseColor(String hex) {
    final hexCode = hex.replaceAll('#', '');
    return Color(int.parse('FF$hexCode', radix: 16));
  }

  factory BrandingConfig.fromJson(Map<String, dynamic> json) =>
      _$BrandingConfigFromJson(json);

  /// Default branding (Thesa UI defaults)
  static const BrandingConfig defaultBranding = BrandingConfig();
}
