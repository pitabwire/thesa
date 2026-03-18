/// Core component rendering engine.
///
/// Maps ComponentDescriptor types to actual Flutter widgets.
/// Uses the plugin system to allow custom component renderers.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../core/core.dart';
import '../plugins/plugin_provider.dart';
import '../widgets/shared/shared.dart';
import 'renderers/renderers.dart';

/// Renders a component based on its descriptor
class ComponentRenderer extends ConsumerWidget {
  const ComponentRenderer({
    required this.component,
    this.params = const {},
    super.key,
  });

  /// Component descriptor to render
  final ComponentDescriptor component;

  /// Route/page parameters
  final Map<String, String> params;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Check permission first
    if (!component.permission.allowed) {
      return const SizedBox.shrink();
    }

    // Wrap in ErrorBoundary to prevent single component failures from crashing page
    return ErrorBoundary(
      componentId: component.id,
      componentType: component.type,
      child: _buildComponentWidget(context, ref),
    );
  }

  Widget _buildComponentWidget(BuildContext context, WidgetRef ref) {
    // Check plugin registry first
    final pluginRegistry = ref.watch(pluginRegistryProvider);
    if (pluginRegistry.hasComponentPlugin(component.type)) {
      final builder = pluginRegistry.getComponentPlugin(component.type)!;
      return builder(component, ref);
    }

    // Route to built-in renderer based on component type
    switch (component.type.toLowerCase()) {
      // Layout components
      case 'stack':
      case 'layout':
        return LayoutRenderer(component: component, params: params);

      case 'grid':
        return GridLayoutRenderer(component: component, params: params);

      case 'tabs':
        return TabsRenderer(component: component, params: params);

      // Content components
      case 'text':
      case 'heading':
      case 'paragraph':
        return TextRenderer(component: component);

      case 'card':
        return CardRenderer(component: component, params: params);

      case 'metric':
      case 'stat':
        return MetricRenderer(component: component);

      case 'list':
        return ListRenderer(component: component, params: params);

      // Data components
      case 'table':
      case 'data_table':
        return DataTableRenderer(component: component, params: params);

      case 'form':
        return FormRenderer(component: component, params: params);

      // Action components
      case 'button':
        return ButtonRenderer(component: component);

      case 'action_group':
        return ActionGroupRenderer(component: component);

      // Media components
      case 'image':
        return ImageRenderer(component: component);

      case 'icon':
        return IconRenderer(component: component);

      // Status components
      case 'badge':
      case 'status':
        return BadgeRenderer(component: component);

      case 'progress':
        return ProgressRenderer(component: component);

      case 'alert':
      case 'notification':
        return AlertRenderer(component: component);

      // Workflow components
      case 'workflow':
      case 'stepper':
        return WorkflowRenderer(component: component, params: params);

      // Divider
      case 'divider':
      case 'separator':
        return DividerRenderer(component: component);

      // Unknown type - check for plugin renderer
      default:
        return _buildPluginOrFallback(context, ref);
    }
  }

  Widget _buildPluginOrFallback(BuildContext context, WidgetRef ref) {
    // TODO: Check plugin registry for custom renderer
    // final pluginRegistry = ref.read(pluginRegistryProvider);
    // final customRenderer = pluginRegistry.getComponentRenderer(component.type);
    // if (customRenderer != null) {
    //   return customRenderer(component, params);
    // }

    // Fallback to placeholder for unknown types
    return AppCard(
      title: 'Unknown Component: ${component.type}',
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Component ID: ${component.id}'),
          if (component.config.isNotEmpty)
            Text('Config: ${component.config.toString()}'),
        ],
      ),
    );
  }
}

/// Error boundary to catch rendering errors
class ErrorBoundary extends StatelessWidget {
  const ErrorBoundary({
    required this.child,
    required this.componentId,
    required this.componentType,
    super.key,
  });

  final Widget child;
  final String componentId;
  final String componentType;

  @override
  Widget build(BuildContext context) {
    return ErrorWidget.withDetails(
      message: 'Error rendering $componentType ($componentId)',
      error: child,
    );
  }
}
