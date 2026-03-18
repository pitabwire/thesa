/// List renderer.
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../component_renderer.dart';

/// Renders list of items
class ListRenderer extends StatelessWidget {
  const ListRenderer({
    required this.component,
    this.params = const {},
    super.key,
  });

  final ComponentDescriptor component;
  final Map<String, String> params;

  @override
  Widget build(BuildContext context) {
    final children = component.children ?? [];

    if (children.isEmpty) {
      return const SizedBox.shrink();
    }

    final dividers = component.config['dividers'] as bool? ?? true;

    if (dividers) {
      return ListView.separated(
        shrinkWrap: true,
        physics: const NeverScrollableScrollPhysics(),
        itemCount: children.length,
        separatorBuilder: (context, index) => const Divider(),
        itemBuilder: (context, index) {
          return ComponentRenderer(
            component: children[index],
            params: params,
          );
        },
      );
    }

    return ListView.builder(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      itemCount: children.length,
      itemBuilder: (context, index) {
        return ComponentRenderer(
          component: children[index],
          params: params,
        );
      },
    );
  }
}
