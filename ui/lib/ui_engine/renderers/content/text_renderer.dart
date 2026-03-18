/// Text content renderer.
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../../design/design.dart';

/// Renders text, heading, and paragraph components
class TextRenderer extends StatelessWidget {
  const TextRenderer({
    required this.component,
    super.key,
  });

  final ComponentDescriptor component;

  @override
  Widget build(BuildContext context) {
    final text = component.config['text'] as String? ??
        component.ui?.label ??
        '';

    if (text.isEmpty) {
      return const SizedBox.shrink();
    }

    final style = _getTextStyle(component.type, component.config);
    final align = _getTextAlign(component.config['align'] as String?);

    return Text(
      text,
      style: style,
      textAlign: align,
    );
  }

  TextStyle _getTextStyle(String type, Map<String, dynamic> config) {
    // Check for explicit style in config
    final styleString = config['style'] as String?;
    if (styleString != null) {
      return _parseStyleString(styleString);
    }

    // Default styles based on type
    switch (type.toLowerCase()) {
      case 'heading':
        final level = config['level'] as int? ?? 1;
        switch (level) {
          case 1:
            return AppTypography.displayLarge;
          case 2:
            return AppTypography.displayMedium;
          case 3:
            return AppTypography.displaySmall;
          case 4:
            return AppTypography.headlineMedium;
          case 5:
            return AppTypography.headlineSmall;
          default:
            return AppTypography.titleLarge;
        }

      case 'paragraph':
        return AppTypography.bodyLarge;

      case 'text':
      default:
        return AppTypography.bodyMedium;
    }
  }

  TextStyle _parseStyleString(String style) {
    switch (style.toLowerCase()) {
      case 'display_large':
        return AppTypography.displayLarge;
      case 'display_medium':
        return AppTypography.displayMedium;
      case 'display_small':
        return AppTypography.displaySmall;
      case 'headline_large':
        return AppTypography.headlineLarge;
      case 'headline_medium':
        return AppTypography.headlineMedium;
      case 'headline_small':
        return AppTypography.headlineSmall;
      case 'title_large':
        return AppTypography.titleLarge;
      case 'title_medium':
        return AppTypography.titleMedium;
      case 'title_small':
        return AppTypography.titleSmall;
      case 'body_large':
        return AppTypography.bodyLarge;
      case 'body_medium':
        return AppTypography.bodyMedium;
      case 'body_small':
        return AppTypography.bodySmall;
      case 'label_large':
        return AppTypography.labelLarge;
      case 'label_medium':
        return AppTypography.labelMedium;
      case 'label_small':
        return AppTypography.labelSmall;
      default:
        return AppTypography.bodyMedium;
    }
  }

  TextAlign _getTextAlign(String? align) {
    switch (align?.toLowerCase()) {
      case 'left':
        return TextAlign.left;
      case 'center':
        return TextAlign.center;
      case 'right':
        return TextAlign.right;
      case 'justify':
        return TextAlign.justify;
      default:
        return TextAlign.left;
    }
  }
}
