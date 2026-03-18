/// Dynamic route generation from navigation tree.
library;

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/core.dart';
import '../pages/page_renderer.dart';
import '../shell/app_shell.dart';

/// Builds GoRouter routes from navigation tree
class DynamicRouteBuilder {
  const DynamicRouteBuilder._();

  /// Generate routes from navigation items
  static List<RouteBase> buildRoutes(List<NavigationItem> items) {
    final routes = <GoRoute>[];

    // Process each top-level navigation item
    for (final item in items) {
      if (!item.permission.allowed) continue;

      // Add route if item has a pageId
      if (item.pageId != null && item.path != null) {
        routes.add(_buildRoute(item));
      }

      // Add child routes
      if (item.children != null) {
        for (final child in item.children!) {
          if (!child.permission.allowed) continue;
          if (child.pageId != null && child.path != null) {
            routes.add(_buildRoute(child));
          }
        }
      }
    }

    // Wrap all routes in shell
    return [
      AppShellRoute(
        branches: [
          StatefulShellBranch(
            routes: routes.isEmpty
                ? [
                    // Fallback route if no navigation items
                    GoRoute(
                      path: '/',
                      builder: (context, state) => const _EmptyStatePage(),
                    ),
                  ]
                : [
                    // Redirect root to first route
                    GoRoute(
                      path: '/',
                      redirect: (context, state) =>
                          routes.first.path,
                    ),
                    ...routes,
                  ],
          ),
        ],
      ),
    ];
  }

  /// Build single route from navigation item
  static GoRoute _buildRoute(NavigationItem item) {
    return GoRoute(
      path: item.path!,
      pageBuilder: (context, state) {
        return NoTransitionPage(
          key: state.pageKey,
          child: PageRenderer(
            pageId: item.pageId!,
            params: state.pathParameters,
          ),
        );
      },
    );
  }
}

/// Empty state when no navigation items available
class _EmptyStatePage extends StatelessWidget {
  const _EmptyStatePage();

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.route, size: 64),
            SizedBox(height: 16),
            Text('No navigation routes configured'),
          ],
        ),
      ),
    );
  }
}
