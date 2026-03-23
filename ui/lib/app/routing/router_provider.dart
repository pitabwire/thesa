/// Router provider for dynamic routing with OIDC auth gating.
///
/// Uses a single long-lived GoRouter instance with redirect-based auth gating
/// to avoid recreating the router (and losing StatefulShellRoute state)
/// on every auth/navigation state change.
library;

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../state/state.dart';
import '../pages/login_page.dart';
import '../pages/page_renderer.dart';
import '../shell/app_shell.dart';

part 'router_provider.g.dart';

/// Notifier that triggers GoRouter redirect when auth or navigation changes.
class _RouterRefreshNotifier extends ChangeNotifier {
  void notify() => notifyListeners();
}

/// GoRouter provider — kept alive to preserve StatefulShellRoute state.
@Riverpod(keepAlive: true)
GoRouter router(Ref ref) {
  final refreshNotifier = _RouterRefreshNotifier();

  // Listen for auth and navigation changes and trigger redirect re-evaluation.
  ref
    ..listen(authProvider, (_, _) => refreshNotifier.notify())
    ..listen(navigationProvider, (_, _) => refreshNotifier.notify());

  return GoRouter(
    initialLocation: '/',
    debugLogDiagnostics: true,
    refreshListenable: refreshNotifier,
    redirect: (context, state) {
      final authAsync = ref.read(authProvider);
      final isLoggedIn = authAsync.maybeWhen(
        data: (auth) =>
            auth.maybeMap(loggedIn: (_) => true, orElse: () => false),
        orElse: () => false,
      );

      final location = state.uri.path;
      final isAuthRoute = location == '/login' || location == '/auth/callback';

      if (!isLoggedIn && !isAuthRoute) {
        return '/login';
      }
      if (isLoggedIn && isAuthRoute) {
        return '/';
      }
      return null;
    },
    routes: [
      // Auth routes (outside shell)
      GoRoute(
        path: '/login',
        builder: (context, state) => const LoginPage(),
      ),
      GoRoute(
        path: '/auth/callback',
        builder: (context, state) => const LoginPage(),
      ),

      // App shell with dynamic page routes
      ShellRoute(
        builder: (context, state, child) => AppShell(child: child),
        routes: [
          GoRoute(
            path: '/',
            redirect: (context, state) {
              // Redirect root to first visible navigation item
              final navAsync = ref.read(navigationProvider);
              return navAsync.maybeWhen(
                data: (nav) {
                  final visible = nav.items
                      .where((item) => item.permission.allowed)
                      .toList();
                  if (visible.isNotEmpty && visible.first.path != null) {
                    return visible.first.path;
                  }
                  return null;
                },
                orElse: () => null,
              );
            },
            builder: (context, state) => const Scaffold(
              body: Center(child: CircularProgressIndicator()),
            ),
          ),

          // Catch-all route for dynamic pages — renders any /<pageId> path
          GoRoute(
            path: '/:pageId',
            pageBuilder: (context, state) {
              final pageId = state.pathParameters['pageId']!;
              return NoTransitionPage(
                key: state.pageKey,
                child: PageRenderer(
                  pageId: _resolvePageId(ref, pageId),
                  params: state.pathParameters,
                ),
              );
            },
          ),
        ],
      ),
    ],
    errorBuilder: (context, state) => _ErrorPage(error: state.error),
  );
}

/// Resolve a URL path segment to a page ID using the navigation tree.
///
/// Navigation items have path: "/tenants" and pageId: "tenants.list".
/// This maps the path segment "tenants" to the pageId "tenants.list".
String _resolvePageId(Ref ref, String pathSegment) {
  final navAsync = ref.read(navigationProvider);
  return navAsync.maybeWhen(
    data: (nav) {
      for (final item in nav.items) {
        if (item.path == '/$pathSegment' && item.pageId != null) {
          return item.pageId!;
        }
        if (item.children != null) {
          for (final child in item.children!) {
            if (child.path == '/$pathSegment' && child.pageId != null) {
              return child.pageId!;
            }
          }
        }
      }
      return pathSegment;
    },
    orElse: () => pathSegment,
  );
}

/// Error page for routing errors
class _ErrorPage extends StatelessWidget {
  const _ErrorPage({required this.error});

  final Exception? error;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.error_outline, size: 64),
            const SizedBox(height: 16),
            const Text('Page Not Found'),
            if (error != null) ...[
              const SizedBox(height: 8),
              Text(
                error.toString(),
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ],
          ],
        ),
      ),
    );
  }
}
