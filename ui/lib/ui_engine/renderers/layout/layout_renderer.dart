/// Stack layout renderer (vertical/horizontal).
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../../design/design.dart';
import '../../component_renderer.dart';

/// Renders stack layout (vertical or horizontal)
class LayoutRenderer extends StatelessWidget {
  const LayoutRenderer({
    required this.component,
    this.params = const {},
    super.key,
  });

  final ComponentDescriptor component;
  final Map<String, String> params;

  @override
  Widget build(BuildContext context) {
    // Parse layout config from component config
    final layoutConfig = _parseLayoutConfig();

    // Render child components
    final children = (component.children ?? [])
        .map((child) => Padding(
              padding: EdgeInsets.only(bottom: layoutConfig.spacing),
              child: ComponentRenderer(component: child, params: params),
            ))
        .toList();

    if (children.isEmpty) {
      return const SizedBox.shrink();
    }

    // Build layout based on direction
    final layout = layoutConfig.direction == LayoutDirection.horizontal
        ? Row(
            mainAxisAlignment: _getMainAxisAlignment(layoutConfig.alignment),
            crossAxisAlignment: _getCrossAxisAlignment(layoutConfig.alignment),
            children: children,
          )
        : Column(
            mainAxisAlignment: _getMainAxisAlignment(layoutConfig.alignment),
            crossAxisAlignment: _getCrossAxisAlignment(layoutConfig.alignment),
            children: children,
          );

    return Padding(
      padding: EdgeInsets.all(layoutConfig.padding),
      child: layout,
    );
  }

  LayoutConfig _parseLayoutConfig() {
    if (component.config['layout'] != null) {
      return LayoutConfig.fromJson(
        component.config['layout'] as Map<String, dynamic>,
      );
    }

    // Default config
    return LayoutConfig(
      type: LayoutType.stack,
      direction: component.config['direction'] == 'horizontal'
          ? LayoutDirection.horizontal
          : LayoutDirection.vertical,
      spacing: (component.config['spacing'] as num?)?.toDouble() ??
          AppSpacing.space16,
      padding: (component.config['padding'] as num?)?.toDouble() ?? 0.0,
    );
  }

  MainAxisAlignment _getMainAxisAlignment(LayoutAlignment? alignment) {
    switch (alignment) {
      case LayoutAlignment.start:
        return MainAxisAlignment.start;
      case LayoutAlignment.center:
        return MainAxisAlignment.center;
      case LayoutAlignment.end:
        return MainAxisAlignment.end;
      case LayoutAlignment.spaceBetween:
        return MainAxisAlignment.spaceBetween;
      case LayoutAlignment.spaceAround:
        return MainAxisAlignment.spaceAround;
      case LayoutAlignment.spaceEvenly:
        return MainAxisAlignment.spaceEvenly;
      default:
        return MainAxisAlignment.start;
    }
  }

  CrossAxisAlignment _getCrossAxisAlignment(LayoutAlignment? alignment) {
    switch (alignment) {
      case LayoutAlignment.start:
        return CrossAxisAlignment.start;
      case LayoutAlignment.center:
        return CrossAxisAlignment.center;
      case LayoutAlignment.end:
        return CrossAxisAlignment.end;
      case LayoutAlignment.stretch:
        return CrossAxisAlignment.stretch;
      default:
        return CrossAxisAlignment.start;
    }
  }
}
