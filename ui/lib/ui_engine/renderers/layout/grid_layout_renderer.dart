/// Grid layout renderer.
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../../design/design.dart';
import '../../component_renderer.dart';

/// Renders responsive grid layout
class GridLayoutRenderer extends StatelessWidget {
  const GridLayoutRenderer({
    required this.component,
    this.params = const {},
    super.key,
  });

  final ComponentDescriptor component;
  final Map<String, String> params;

  @override
  Widget build(BuildContext context) {
    final layoutConfig = _parseLayoutConfig();
    final children = component.children ?? [];

    if (children.isEmpty) {
      return const SizedBox.shrink();
    }

    // Build grid using GridView
    return Padding(
      padding: EdgeInsets.all(layoutConfig.padding),
      child: GridView.builder(
        shrinkWrap: true,
        physics: const NeverScrollableScrollPhysics(),
        gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
          crossAxisCount: layoutConfig.columns ?? 2,
          crossAxisSpacing: layoutConfig.spacing,
          mainAxisSpacing: layoutConfig.spacing,
          childAspectRatio: 1.0,
        ),
        itemCount: children.length,
        itemBuilder: (context, index) {
          return ComponentRenderer(
            component: children[index],
            params: params,
          );
        },
      ),
    );
  }

  LayoutConfig _parseLayoutConfig() {
    if (component.config['layout'] != null) {
      return LayoutConfig.fromJson(
        component.config['layout'] as Map<String, dynamic>,
      );
    }

    return LayoutConfig(
      type: LayoutType.grid,
      columns: component.config['columns'] as int? ?? 2,
      spacing: (component.config['spacing'] as num?)?.toDouble() ??
          AppSpacing.space16,
      padding: (component.config['padding'] as num?)?.toDouble() ?? 0.0,
    );
  }
}
