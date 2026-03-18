/// Plugin registry provider.
library;

import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'plugin_registry.dart';

part 'plugin_provider.g.dart';

/// Global plugin registry provider
@Riverpod(keepAlive: true)
PluginRegistry pluginRegistry(PluginRegistryRef ref) {
  return PluginRegistry();
}
