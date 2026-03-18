/// Card renderer.
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../../widgets/shared/shared.dart';
import '../../component_renderer.dart';

/// Renders card component with optional header, body, and footer
class CardRenderer extends StatelessWidget {
  const CardRenderer({
    required this.component,
    this.params = const {},
    super.key,
  });

  final ComponentDescriptor component;
  final Map<String, String> params;

  @override
  Widget build(BuildContext context) {
    final title = component.config['title'] as String? ?? component.ui?.label;
    final subtitle = component.config['subtitle'] as String?;

    // If card has children, render them as body content
    Widget? body;
    if (component.children != null && component.children!.isNotEmpty) {
      body = Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: component.children!.map((child) {
          return ComponentRenderer(
            component: child,
            params: params,
          );
        }).toList(),
      );
    }

    return AppCard(
      title: title,
      subtitle: subtitle,
      body: body,
      actions: _buildActions(context),
    );
  }

  List<Widget>? _buildActions(BuildContext context) {
    if (component.actions == null || component.actions!.isEmpty) {
      return null;
    }

    return component.actions!
        .where((action) => action.permission.allowed)
        .map((action) {
      return AppButton(
        label: action.label,
        onPressed: () {
          // TODO: Handle action execution
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Action: ${action.label}')),
          );
        },
        variant: _parseButtonVariant(action.style),
        size: AppButtonSize.small,
      );
    }).toList();
  }

  AppButtonVariant _parseButtonVariant(String? style) {
    switch (style?.toLowerCase()) {
      case 'primary':
        return AppButtonVariant.primary;
      case 'secondary':
        return AppButtonVariant.secondary;
      case 'tertiary':
        return AppButtonVariant.tertiary;
      case 'destructive':
      case 'danger':
        return AppButtonVariant.destructive;
      default:
        return AppButtonVariant.secondary;
    }
  }
}
