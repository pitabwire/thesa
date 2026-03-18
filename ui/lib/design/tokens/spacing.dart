/// Spacing design tokens for the Thesa UI design system.
///
/// Consistent spacing scale based on 4px base unit.
library;

/// Spacing tokens (based on 4px base unit)
class AppSpacing {
  const AppSpacing._();

  /// 2px - Tight spacing (inside dense components)
  static const double space2 = 2;

  /// 4px - Minimal spacing (icon-to-text gap)
  static const double space4 = 4;

  /// 8px - Compact spacing (between related items)
  static const double space8 = 8;

  /// 12px - Default padding inside components
  static const double space12 = 12;

  /// 16px - Standard spacing between components
  static const double space16 = 16;

  /// 24px - Generous spacing between sections
  static const double space24 = 24;

  /// 32px - Large spacing between page sections
  static const double space32 = 32;

  /// 48px - Extra large spacing (page top margin)
  static const double space48 = 48;

  /// 64px - Maximum spacing (rarely used)
  static const double space64 = 64;
}

/// Border radius tokens
class AppBorderRadius {
  const AppBorderRadius._();

  /// 4px - Small radius (chips, small buttons)
  static const double small = 4;

  /// 8px - Medium radius (cards, dialogs, standard buttons)
  static const double medium = 8;

  /// 12px - Large radius (prominent cards)
  static const double large = 12;

  /// 16px - Extra large radius (special components)
  static const double extraLarge = 16;

  /// 999px - Pill shape (fully rounded)
  static const double pill = 999;
}

/// Elevation (shadow) tokens
class AppElevation {
  const AppElevation._();

  /// No elevation (flat)
  static const double flat = 0;

  /// Minimal elevation (subtle separation)
  static const double low = 1;

  /// Standard elevation (cards)
  static const double medium = 2;

  /// High elevation (dialogs, popovers)
  static const double high = 4;

  /// Very high elevation (modals)
  static const double veryHigh = 8;
}

/// Sizing tokens (common dimensions)
class AppSizing {
  const AppSizing._();

  /// Minimum touch target size (accessibility)
  static const double minTouchTarget = 48;

  /// Icon sizes
  static const double iconSmall = 16;
  static const double iconMedium = 24;
  static const double iconLarge = 32;

  /// Button heights
  static const double buttonSmall = 32;
  static const double buttonMedium = 40;
  static const double buttonLarge = 48;

  /// Input field height
  static const double inputHeight = 48;

  /// Avatar sizes
  static const double avatarSmall = 24;
  static const double avatarMedium = 40;
  static const double avatarLarge = 64;
}
