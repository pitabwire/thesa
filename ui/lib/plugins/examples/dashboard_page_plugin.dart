/// Example page plugin demonstrating custom page rendering.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/core.dart';
import '../../design/design.dart';
import '../../state/actions/action_provider.dart';
import '../../ui_engine/component_renderer.dart';
import '../../widgets/shared/shared.dart';

/// Example dashboard page plugin
///
/// Demonstrates:
/// - Custom page layout mixing custom widgets with generic engine
/// - Reading permissions from page descriptor
/// - Triggering actions via actionProvider
/// - Delegating component rendering to generic engine
Widget buildDashboardPage(
  BuildContext context,
  PageDescriptor descriptor,
  WidgetRef ref,
) {
  return DashboardPagePlugin(descriptor: descriptor);
}

/// Custom dashboard page implementation
class DashboardPagePlugin extends ConsumerWidget {
  const DashboardPagePlugin({
    required this.descriptor,
    super.key,
  });

  final PageDescriptor descriptor;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);

    // Extract components from descriptor
    final components = descriptor.components;

    // Separate metric cards from other components
    final metricComponents = components
        .where((c) => c.type == 'metric' || c.type == 'metric_card')
        .toList();
    final otherComponents =
        components.where((c) => c.type != 'metric' && c.type != 'metric_card').toList();

    return Scaffold(
      appBar: AppBar(
        title: Text(descriptor.title),
        actions: [
          // Render page actions if allowed
          if (descriptor.actions.isNotEmpty)
            IconButton(
              icon: const Icon(Icons.more_vert),
              onPressed: () => _showActionsMenu(context, ref),
            ),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(AppSpacing.space16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Custom welcome section
            Container(
              padding: const EdgeInsets.all(AppSpacing.space24),
              decoration: BoxDecoration(
                gradient: LinearGradient(
                  colors: [
                    theme.colorScheme.primary,
                    theme.colorScheme.primary.withOpacity(0.7),
                  ],
                ),
                borderRadius: BorderRadius.circular(AppBorderRadius.large),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Welcome to ${descriptor.title}',
                    style: theme.textTheme.headlineMedium?.copyWith(
                      color: theme.colorScheme.onPrimary,
                    ),
                  ),
                  const SizedBox(height: AppSpacing.space8),
                  Text(
                    descriptor.description ?? 'Your enterprise dashboard',
                    style: theme.textTheme.bodyLarge?.copyWith(
                      color: theme.colorScheme.onPrimary.withOpacity(0.9),
                    ),
                  ),
                ],
              ),
            ),

            const SizedBox(height: AppSpacing.space24),

            // Metrics row using generic engine
            if (metricComponents.isNotEmpty) ...[
              Text(
                'Key Metrics',
                style: theme.textTheme.titleLarge,
              ),
              const SizedBox(height: AppSpacing.space16),
              GridView.builder(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                  crossAxisCount: 4,
                  mainAxisSpacing: AppSpacing.space16,
                  crossAxisSpacing: AppSpacing.space16,
                  childAspectRatio: 1.5,
                ),
                itemCount: metricComponents.length,
                itemBuilder: (context, index) {
                  // Delegate to generic component renderer
                  return ComponentRenderer(
                    component: metricComponents[index],
                  );
                },
              ),
              const SizedBox(height: AppSpacing.space32),
            ],

            // Other components using generic engine
            if (otherComponents.isNotEmpty) ...[
              Text(
                'Details',
                style: theme.textTheme.titleLarge,
              ),
              const SizedBox(height: AppSpacing.space16),
              ...otherComponents.map((component) {
                return Padding(
                  padding: const EdgeInsets.only(bottom: AppSpacing.space16),
                  child: ComponentRenderer(component: component),
                );
              }),
            ],
          ],
        ),
      ),
    );
  }

  void _showActionsMenu(BuildContext context, WidgetRef ref) {
    showModalBottomSheet(
      context: context,
      builder: (context) => ListView(
        shrinkWrap: true,
        children: descriptor.actions
            .where((action) => action.permission.allowed)
            .map((action) {
          return ListTile(
            leading: action.icon != null
                ? Icon(_parseIcon(action.icon!))
                : const Icon(Icons.touch_app),
            title: Text(action.label),
            onTap: () {
              Navigator.pop(context);
              ref.read(actionProvider(action.actionId).notifier).requestExecution(
                    descriptor: action,
                  );
            },
          );
        }).toList(),
      ),
    );
  }

  IconData _parseIcon(String icon) {
    // Simple icon parser - in production use a proper icon registry
    switch (icon) {
      case 'refresh':
        return Icons.refresh;
      case 'settings':
        return Icons.settings;
      case 'download':
        return Icons.download;
      default:
        return Icons.touch_app;
    }
  }
}
