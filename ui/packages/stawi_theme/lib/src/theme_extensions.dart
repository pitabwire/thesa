import 'package:flutter/material.dart';
import 'colors.dart';

/// Custom semantic color tokens that extend Material's [ColorScheme].
///
/// Access via `Theme.of(context).extension<StawiColorTokens>()`.
///
/// ```dart
/// final tokens = Theme.of(context).extension<StawiColorTokens>()!;
/// Text('Subtle', style: TextStyle(color: tokens.mutedForeground));
/// ```
class StawiColorTokens extends ThemeExtension<StawiColorTokens> {
  const StawiColorTokens({
    required this.background,
    required this.foreground,
    required this.card,
    required this.cardForeground,
    required this.muted,
    required this.mutedForeground,
    required this.accent,
    required this.accentForeground,
    required this.border,
    required this.input,
    required this.ring,
    required this.destructive,
    required this.destructiveForeground,
    required this.secondary,
    required this.secondaryForeground,
  });

  final Color background;
  final Color foreground;
  final Color card;
  final Color cardForeground;
  final Color muted;
  final Color mutedForeground;
  final Color accent;
  final Color accentForeground;
  final Color border;
  final Color input;
  final Color ring;
  final Color destructive;
  final Color destructiveForeground;
  final Color secondary;
  final Color secondaryForeground;

  /// Light mode tokens.
  static const light = StawiColorTokens(
    background: StawiColors.lightBackground,
    foreground: StawiColors.lightForeground,
    card: StawiColors.lightCard,
    cardForeground: StawiColors.lightCardForeground,
    muted: StawiColors.lightMuted,
    mutedForeground: StawiColors.lightMutedForeground,
    accent: StawiColors.lightAccent,
    accentForeground: StawiColors.lightAccentForeground,
    border: StawiColors.lightBorder,
    input: StawiColors.lightInput,
    ring: StawiColors.lightRing,
    destructive: StawiColors.lightDestructive,
    destructiveForeground: StawiColors.lightDestructiveForeground,
    secondary: StawiColors.lightSecondary,
    secondaryForeground: StawiColors.lightSecondaryForeground,
  );

  /// Dark mode tokens.
  static const dark = StawiColorTokens(
    background: StawiColors.darkBackground,
    foreground: StawiColors.darkForeground,
    card: StawiColors.darkCard,
    cardForeground: StawiColors.darkCardForeground,
    muted: StawiColors.darkMuted,
    mutedForeground: StawiColors.darkMutedForeground,
    accent: StawiColors.darkAccent,
    accentForeground: StawiColors.darkAccentForeground,
    border: StawiColors.darkBorder,
    input: StawiColors.darkInput,
    ring: StawiColors.darkRing,
    destructive: StawiColors.darkDestructive,
    destructiveForeground: StawiColors.darkDestructiveForeground,
    secondary: StawiColors.darkSecondary,
    secondaryForeground: StawiColors.darkSecondaryForeground,
  );

  @override
  StawiColorTokens copyWith({
    Color? background,
    Color? foreground,
    Color? card,
    Color? cardForeground,
    Color? muted,
    Color? mutedForeground,
    Color? accent,
    Color? accentForeground,
    Color? border,
    Color? input,
    Color? ring,
    Color? destructive,
    Color? destructiveForeground,
    Color? secondary,
    Color? secondaryForeground,
  }) {
    return StawiColorTokens(
      background: background ?? this.background,
      foreground: foreground ?? this.foreground,
      card: card ?? this.card,
      cardForeground: cardForeground ?? this.cardForeground,
      muted: muted ?? this.muted,
      mutedForeground: mutedForeground ?? this.mutedForeground,
      accent: accent ?? this.accent,
      accentForeground: accentForeground ?? this.accentForeground,
      border: border ?? this.border,
      input: input ?? this.input,
      ring: ring ?? this.ring,
      destructive: destructive ?? this.destructive,
      destructiveForeground:
          destructiveForeground ?? this.destructiveForeground,
      secondary: secondary ?? this.secondary,
      secondaryForeground: secondaryForeground ?? this.secondaryForeground,
    );
  }

  @override
  StawiColorTokens lerp(ThemeExtension<StawiColorTokens>? other, double t) {
    if (other is! StawiColorTokens) return this;
    return StawiColorTokens(
      background: Color.lerp(background, other.background, t)!,
      foreground: Color.lerp(foreground, other.foreground, t)!,
      card: Color.lerp(card, other.card, t)!,
      cardForeground: Color.lerp(cardForeground, other.cardForeground, t)!,
      muted: Color.lerp(muted, other.muted, t)!,
      mutedForeground: Color.lerp(mutedForeground, other.mutedForeground, t)!,
      accent: Color.lerp(accent, other.accent, t)!,
      accentForeground:
          Color.lerp(accentForeground, other.accentForeground, t)!,
      border: Color.lerp(border, other.border, t)!,
      input: Color.lerp(input, other.input, t)!,
      ring: Color.lerp(ring, other.ring, t)!,
      destructive: Color.lerp(destructive, other.destructive, t)!,
      destructiveForeground:
          Color.lerp(destructiveForeground, other.destructiveForeground, t)!,
      secondary: Color.lerp(secondary, other.secondary, t)!,
      secondaryForeground:
          Color.lerp(secondaryForeground, other.secondaryForeground, t)!,
    );
  }
}

/// Spacing and radius constants.
///
/// ```dart
/// final spacing = Theme.of(context).extension<StawiSpacing>()!;
/// Padding(padding: EdgeInsets.all(spacing.md));
/// ```
class StawiSpacing extends ThemeExtension<StawiSpacing> {
  const StawiSpacing({
    this.xs = 4.0,
    this.sm = 8.0,
    this.md = 16.0,
    this.lg = 24.0,
    this.xl = 32.0,
    this.xxl = 48.0,
    this.xxxl = 64.0,
    this.radiusSm = 6.0,
    this.radiusMd = 8.0,
    this.radiusLg = 10.0,
    this.radiusXl = 14.0,
    this.radiusFull = 9999.0,
  });

  final double xs;
  final double sm;
  final double md;
  final double lg;
  final double xl;
  final double xxl;
  final double xxxl;
  final double radiusSm;
  final double radiusMd;
  final double radiusLg;
  final double radiusXl;
  final double radiusFull;

  BorderRadius get borderRadiusSm => BorderRadius.circular(radiusSm);
  BorderRadius get borderRadiusMd => BorderRadius.circular(radiusMd);
  BorderRadius get borderRadiusLg => BorderRadius.circular(radiusLg);
  BorderRadius get borderRadiusXl => BorderRadius.circular(radiusXl);
  BorderRadius get borderRadiusFull => BorderRadius.circular(radiusFull);

  @override
  StawiSpacing copyWith({
    double? xs,
    double? sm,
    double? md,
    double? lg,
    double? xl,
    double? xxl,
    double? xxxl,
    double? radiusSm,
    double? radiusMd,
    double? radiusLg,
    double? radiusXl,
    double? radiusFull,
  }) {
    return StawiSpacing(
      xs: xs ?? this.xs,
      sm: sm ?? this.sm,
      md: md ?? this.md,
      lg: lg ?? this.lg,
      xl: xl ?? this.xl,
      xxl: xxl ?? this.xxl,
      xxxl: xxxl ?? this.xxxl,
      radiusSm: radiusSm ?? this.radiusSm,
      radiusMd: radiusMd ?? this.radiusMd,
      radiusLg: radiusLg ?? this.radiusLg,
      radiusXl: radiusXl ?? this.radiusXl,
      radiusFull: radiusFull ?? this.radiusFull,
    );
  }

  @override
  StawiSpacing lerp(ThemeExtension<StawiSpacing>? other, double t) {
    if (other is! StawiSpacing) return this;
    return StawiSpacing(
      xs: _lerpDouble(xs, other.xs, t),
      sm: _lerpDouble(sm, other.sm, t),
      md: _lerpDouble(md, other.md, t),
      lg: _lerpDouble(lg, other.lg, t),
      xl: _lerpDouble(xl, other.xl, t),
      xxl: _lerpDouble(xxl, other.xxl, t),
      xxxl: _lerpDouble(xxxl, other.xxxl, t),
      radiusSm: _lerpDouble(radiusSm, other.radiusSm, t),
      radiusMd: _lerpDouble(radiusMd, other.radiusMd, t),
      radiusLg: _lerpDouble(radiusLg, other.radiusLg, t),
      radiusXl: _lerpDouble(radiusXl, other.radiusXl, t),
      radiusFull: _lerpDouble(radiusFull, other.radiusFull, t),
    );
  }

  static double _lerpDouble(double a, double b, double t) => a + (b - a) * t;
}
