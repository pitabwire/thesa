/// Plugin registration entry point.
///
/// This file is called during app initialization to register all plugins.
/// Add your plugin registrations here.
library;

import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'examples/dashboard_page_plugin.dart';
import 'examples/invoice_schema_plugin.dart';
import 'examples/map_component_plugin.dart';
import 'plugin_provider.dart';
import 'plugin_registry.dart';

/// Register all application plugins
///
/// Called from main.dart before runApp().
/// Plugins are registered in the order listed here.
void registerPlugins(WidgetRef ref) {
  final registry = ref.read(pluginRegistryProvider);

  // Register example plugins (remove in production)
  registerExamplePlugins(registry);

  // Register your custom plugins here
  // registry.registerPage('my-page', myPageBuilder);
  // registry.registerComponent('my-component', myComponentBuilder);
  // registry.registerSchemaRenderer('my-schema', mySchemaBuilder);
}

/// Register example plugins for demonstration
void registerExamplePlugins(PluginRegistry registry) {
  registry
    // Example page plugin
    ..registerPage('dashboard', buildDashboardPage)
    // Example component plugin
    ..registerComponent('map', buildMapComponent)
    // Example schema renderer plugin
    ..registerSchemaRenderer('invoice_schema', buildInvoiceForm);
}
