/// Application shell with sidebar and content area.
library;

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../design/design.dart';
import 'sidebar.dart';

/// Main app shell with persistent sidebar and dynamic content area
class AppShell extends StatelessWidget {
  const AppShell({
    required this.child,
    super.key,
  });

  /// Content area (current route)
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Row(
        children: [
          // Sidebar (persistent)
          const AppSidebar(),

          // Content area (changes with route)
          Expanded(
            child: child,
          ),
        ],
      ),
    );
  }
}

/// Shell route for go_router StatefulShellRoute
class AppShellRoute extends StatefulShellRoute {
  AppShellRoute({
    required super.branches,
  }) : super(
          navigatorContainerBuilder: (context, navigationShell, children) {
            return AppShell(
              child: navigationShell,
            );
          },
        );
}
