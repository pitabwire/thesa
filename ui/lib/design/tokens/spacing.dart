/// Spacing design tokens for the Thesa UI design system.
///
/// Consistent spacing scale based on 4px base unit.
library;

/// Spacing tokens (based on 4px base unit)
class AppSpacing {
  const AppSpacing._();

  /// 2px - Tight spacing (inside dense components)
  static const double space2 = 2.0;

  /// 4px - Minimal spacing (icon-to-text gap)
  static const double space4 = 4.0;

  /// 8px - Compact spacing (between related items)
  static const double space8 = 8.0;

  /// 12px - Default padding inside components
  static const double space12 = 12.0;

  /// 16px - Standard spacing between components
  static const double space16 = 16.0;

  /// 24px - Generous spacing between sections
  static const double space24 = 24.0;

  /// 32px - Large spacing between page sections
  static const double space32 = 32.0;

  /// 48px - Extra large spacing (page top margin)
  static const double space48 = 48.0;

  /// 64px - Maximum spacing (rarely used)
  static const double space64 = 64.0;
}

/// Border radius tokens
class AppBorderRadius {
  const AppBorderRadius._();

  /// 4px - Small radius (chips, small buttons)
  static const double small = 4.0;

  /// 8px - Medium radius (cards, dialogs, standard buttons)
  static const double medium = 8.0;

  /// 12px - Large radius (prominent cards)
  static const double large = 12.0;

  /// 16px - Extra large radius (special components)
  static const double extraLarge = 16.0;

  /// 999px - Pill shape (fully rounded)
  static const double pill = 999.0;
}

/// Elevation (shadow) tokens
class AppElevation {
  const AppElevation._();

  /// No elevation (flat)
  static const double flat = 0.0;

  /// Minimal elevation (subtle separation)
  static const double low = 1.0;

  /// Standard elevation (cards)
  static const double medium = 2.0;

  /// High elevation (dialogs, popovers)
  static const double high = 4.0;

  /// Very high elevation (modals)
  static const double veryHigh = 8.0;
}

/// Sizing tokens (common dimensions)
class AppSizing {
  const AppSizing._();

  /// Minimum touch target size (accessibility)
  static const double minTouchTarget = 48.0;

  /// Icon sizes
  static const double iconSmall = 16.0;
  static const double iconMedium = 24.0;
  static const double iconLarge = 32.0;

  /// Button heights
  static const double buttonSmall = 32.0;
  static const double buttonMedium = 40.0;
  static const double buttonLarge = 48.0;

  /// Input field height
  static const double inputHeight = 48.0;

  /// Avatar sizes
  static const double avatarSmall = 24.0;
  static const double avatarMedium = 40.0;
  static const double avatarLarge = 64.0;
}
