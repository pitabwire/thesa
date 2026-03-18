import 'package:flutter/material.dart';
import '../theme.dart';

/// A centered section header with title and subtitle.
///
/// ```dart
/// StawiSectionHeader(
///   title: 'Built for Production',
///   subtitle: 'Everything you need at scale',
/// )
/// ```
class StawiSectionHeader extends StatelessWidget {
  const StawiSectionHeader({
    super.key,
    required this.title,
    this.subtitle,
    this.titleStyle,
    this.crossAxisAlignment = CrossAxisAlignment.center,
    this.padding = const EdgeInsets.only(bottom: 32),
  });

  final String title;
  final String? subtitle;
  final TextStyle? titleStyle;
  final CrossAxisAlignment crossAxisAlignment;
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context) {
    final tokens = context.stawiColors;
    final textAlign = crossAxisAlignment == CrossAxisAlignment.center
        ? TextAlign.center
        : TextAlign.start;

    return Padding(
      padding: padding,
      child: Column(
        crossAxisAlignment: crossAxisAlignment,
        children: [
          Text(
            title,
            textAlign: textAlign,
            style: titleStyle ??
                TextStyle(
                  fontSize: 32,
                  fontWeight: FontWeight.w600,
                  letterSpacing: -1.2,
                  height: 1.15,
                  color: tokens.foreground,
                ),
          ),
          if (subtitle != null) ...[
            const SizedBox(height: 8),
            Text(
              subtitle!,
              textAlign: textAlign,
              style: TextStyle(
                fontSize: 17,
                fontWeight: FontWeight.w500,
                color: tokens.mutedForeground,
              ),
            ),
          ],
        ],
      ),
    );
  }
}
