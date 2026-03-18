/// Tabbed layout renderer.
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../component_renderer.dart';

/// Renders tabbed layout
class TabsRenderer extends StatelessWidget {
  const TabsRenderer({
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

    return DefaultTabController(
      length: children.length,
      child: Column(
        children: [
          TabBar(
            isScrollable: true,
            tabs: children.map((child) {
              return Tab(
                text: child.ui?.label ?? child.type,
                icon: child.ui?.icon != null ? Icon(_parseIcon(child.ui!.icon!)) : null,
              );
            }).toList(),
          ),
          Expanded(
            child: TabBarView(
              children: children.map((child) {
                return ComponentRenderer(
                  component: child,
                  params: params,
                );
              }).toList(),
            ),
          ),
        ],
      ),
    );
  }

  IconData _parseIcon(String icon) {
    // Simple icon mapping - in production this would be more comprehensive
    switch (icon.toLowerCase()) {
      case 'home':
        return Icons.home;
      case 'settings':
        return Icons.settings;
      case 'person':
      case 'user':
        return Icons.person;
      case 'list':
        return Icons.list;
      case 'table':
        return Icons.table_chart;
      default:
        return Icons.folder;
    }
  }
}
