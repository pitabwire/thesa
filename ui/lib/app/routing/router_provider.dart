/// Router provider for dynamic routing with OIDC auth gating.
library;

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../state/auth/auth_provider.dart';
import '../../state/auth/auth_state.dart';
import '../../state/state.dart';
import '../pages/login_page.dart';
import 'dynamic_route_builder.dart';

part 'router_provider.g.dart';

/// GoRouter provider with auth-gating
@riverpod
GoRouter router(Ref ref) {
  final authAsync = ref.watch(authProvider);
  final navigationAsync = ref.watch(navigationProvider);

  // Determine if user is authenticated
  final isLoggedIn = authAsync.maybeWhen(
    data: (auth) => auth.maybeMap(loggedIn: (_) => true, orElse: () => false),
    orElse: () => false,
  );

  final isLoggingIn = authAsync.maybeWhen(
    data: (auth) => auth.maybeMap(loggingIn: (_) => true, orElse: () => false),
    orElse: () => false,
  );

  // Build routes based on auth state
  final List<RouteBase> routes;

  if (!isLoggedIn) {
    // Not authenticated — show login page
    routes = [
      GoRoute(path: '/', builder: (context, state) => const LoginPage()),
    ];
  } else {
    // Authenticated — build dynamic routes from BFF navigation
    routes = navigationAsync.maybeWhen<List<RouteBase>>(
      data: (navigation) {
        final visibleItems = navigation.items
            .where((item) => item.permission.allowed)
            .toList();
        return DynamicRouteBuilder.buildRoutes(visibleItems);
      },
      orElse: () => [
        // Loading navigation
        GoRoute(
          path: '/',
          builder: (context, state) =>
              const Scaffold(body: Center(child: CircularProgressIndicator())),
        ),
      ],
    );
  }

  return GoRouter(
    routes: routes,
    initialLocation: '/',
    debugLogDiagnostics: true,
    errorBuilder: (context, state) => _ErrorPage(error: state.error),
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
