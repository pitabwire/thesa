/// Plugin Registry for custom renderers and pages.
///
/// Allows developers to register custom implementations that override
/// the default generic rendering for specific pages, components, or schemas.
library;

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:logging/logging.dart';

import '../core/core.dart';

final _logger = Logger('PluginRegistry');

/// Page plugin builder signature
typedef PagePluginBuilder = Widget Function(
  BuildContext context,
  PageDescriptor descriptor,
  WidgetRef ref,
);

/// Component plugin builder signature
typedef ComponentPluginBuilder = Widget Function(
  ComponentDescriptor descriptor,
  WidgetRef ref,
);

/// Schema renderer plugin builder signature
typedef SchemaRendererPluginBuilder = Widget Function(
  Schema schema,
  WidgetRef ref,
  ValueChanged<Map<String, dynamic>> onSaved,
);

/// Plugin Registry for managing custom renderers
///
/// Resolution order:
/// 1. Check plugin registry for custom implementation
/// 2. Fall back to built-in renderer
/// 3. Show placeholder for unknown types
///
/// Example:
/// ```dart
/// final registry = PluginRegistry();
/// registry.registerPage('dashboard', (context, descriptor, ref) {
///   return CustomDashboardPage(descriptor: descriptor);
/// });
/// ```
class PluginRegistry {
  PluginRegistry();

  /// Registered page plugins (pageId -> builder)
  final Map<String, PagePluginBuilder> _pagePlugins = {};

  /// Registered component plugins (componentType -> builder)
  final Map<String, ComponentPluginBuilder> _componentPlugins = {};

  /// Registered schema renderer plugins (schemaId -> builder)
  final Map<String, SchemaRendererPluginBuilder> _schemaRendererPlugins = {};

  // ========== Page Plugins ==========

  /// Register a custom page renderer
  ///
  /// The builder receives the full PageDescriptor so plugins can:
  /// - Read permissions from descriptor.permission.allowed
  /// - Access page metadata
  /// - Render page actions
  /// - Mix custom widgets with generic engine delegation
  ///
  /// If a page plugin is already registered for this pageId, it will be
  /// replaced and a warning will be logged.
  void registerPage(String pageId, PagePluginBuilder builder) {
    if (_pagePlugins.containsKey(pageId)) {
      _logger.warning(
        'Replacing existing page plugin for: $pageId (last one wins)',
      );
    }
    _pagePlugins[pageId] = builder;
    _logger.info('Registered page plugin: $pageId');
  }

  /// Check if a page plugin is registered
  bool hasPagePlugin(String pageId) => _pagePlugins.containsKey(pageId);

  /// Get a page plugin builder
  PagePluginBuilder? getPagePlugin(String pageId) => _pagePlugins[pageId];

  /// Unregister a page plugin
  void unregisterPage(String pageId) {
    if (_pagePlugins.remove(pageId) != null) {
      _logger.info('Unregistered page plugin: $pageId');
    }
  }

  // ========== Component Plugins ==========

  /// Register a custom component renderer
  ///
  /// The builder receives the ComponentDescriptor with all config,
  /// metadata, and actions. Plugins MUST check `allowed` flags.
  ///
  /// Example:
  /// ```dart
  /// registry.registerComponent('map', (descriptor, ref) {
  ///   final config = descriptor.config;
  ///   return MapWidget(
  ///     center: config['center'],
  ///     markers: config['markers'],
  ///   );
  /// });
  /// ```
  void registerComponent(String componentType, ComponentPluginBuilder builder) {
    if (_componentPlugins.containsKey(componentType)) {
      _logger.warning(
        'Replacing existing component plugin for: $componentType (last one wins)',
      );
    }
    _componentPlugins[componentType] = builder;
    _logger.info('Registered component plugin: $componentType');
  }

  /// Check if a component plugin is registered
  bool hasComponentPlugin(String componentType) =>
      _componentPlugins.containsKey(componentType);

  /// Get a component plugin builder
  ComponentPluginBuilder? getComponentPlugin(String componentType) =>
      _componentPlugins[componentType];

  /// Unregister a component plugin
  void unregisterComponent(String componentType) {
    if (_componentPlugins.remove(componentType) != null) {
      _logger.info('Unregistered component plugin: $componentType');
    }
  }

  // ========== Schema Renderer Plugins ==========

  /// Register a custom schema renderer (custom form)
  ///
  /// The builder receives the Schema definition and an onSaved callback.
  /// Plugins are responsible for:
  /// - Validating inputs
  /// - Calling onSaved with the form data map
  /// - Checking field permissions
  ///
  /// Example:
  /// ```dart
  /// registry.registerSchemaRenderer('invoice_schema', (schema, ref, onSaved) {
  ///   return CustomInvoiceForm(
  ///     schema: schema,
  ///     onSubmit: (data) => onSaved(data),
  ///   );
  /// });
  /// ```
  void registerSchemaRenderer(
    String schemaId,
    SchemaRendererPluginBuilder builder,
  ) {
    if (_schemaRendererPlugins.containsKey(schemaId)) {
      _logger.warning(
        'Replacing existing schema renderer plugin for: $schemaId (last one wins)',
      );
    }
    _schemaRendererPlugins[schemaId] = builder;
    _logger.info('Registered schema renderer plugin: $schemaId');
  }

  /// Check if a schema renderer plugin is registered
  bool hasSchemaRenderer(String schemaId) =>
      _schemaRendererPlugins.containsKey(schemaId);

  /// Get a schema renderer plugin builder
  SchemaRendererPluginBuilder? getSchemaRenderer(String schemaId) =>
      _schemaRendererPlugins[schemaId];

  /// Unregister a schema renderer plugin
  void unregisterSchemaRenderer(String schemaId) {
    if (_schemaRendererPlugins.remove(schemaId) != null) {
      _logger.info('Unregistered schema renderer plugin: $schemaId');
    }
  }

  // ========== Bulk Operations ==========

  /// Clear all registered plugins
  void clearAll() {
    _pagePlugins.clear();
    _componentPlugins.clear();
    _schemaRendererPlugins.clear();
    _logger.info('Cleared all plugins');
  }

  /// Get registration counts
  Map<String, int> getStats() => {
        'pages': _pagePlugins.length,
        'components': _componentPlugins.length,
        'schemaRenderers': _schemaRendererPlugins.length,
      };
}
