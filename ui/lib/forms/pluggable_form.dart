/// Pluggable form wrapper that checks plugin registry for custom schema renderers.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../core/core.dart';
import '../plugins/plugin_provider.dart';
import 'dynamic_form.dart';

/// Form wrapper that supports plugin-based schema renderers
///
/// Checks plugin registry for custom schema renderers before
/// falling back to the generic DynamicForm.
class PluggableForm extends ConsumerWidget {
  const PluggableForm({
    required this.schema,
    required this.onSubmit,
    this.initialData,
    this.mode = FormMode.create,
    this.submitButtonText,
    this.showCancelButton = true,
    this.onCancel,
    super.key,
  });

  final Schema schema;
  final Future<void> Function(Map<String, dynamic> data) onSubmit;
  final Map<String, dynamic>? initialData;
  final FormMode mode;
  final String? submitButtonText;
  final bool showCancelButton;
  final VoidCallback? onCancel;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final pluginRegistry = ref.watch(pluginRegistryProvider);

    // Check plugin registry for custom schema renderer
    if (pluginRegistry.hasSchemaRenderer(schema.id)) {
      final builder = pluginRegistry.getSchemaRenderer(schema.id)!;
      return builder(
        schema,
        ref,
        (data) async => await onSubmit(data),
      );
    }

    // Fall back to generic form renderer
    return DynamicForm(
      schema: schema,
      onSubmit: onSubmit,
      initialData: initialData,
      mode: mode,
      submitButtonText: submitButtonText,
      showCancelButton: showCancelButton,
      onCancel: onCancel,
    );
  }
}
