/// Alert/notification renderer.
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../../design/design.dart';

/// Renders alert/notification component
class AlertRenderer extends StatelessWidget {
  const AlertRenderer({
    required this.component,
    super.key,
  });

  final ComponentDescriptor component;

  @override
  Widget build(BuildContext context) {
    final title = component.config['title'] as String? ?? component.ui?.label;
    final message = component.config['message'] as String? ??
        component.config['text'] as String? ??
        '';
    final severity = component.config['severity'] as String? ??
        component.config['type'] as String? ??
        'info';
    final dismissible = component.config['dismissible'] as bool? ?? false;

    final colorScheme = _getColorScheme(context, severity);

    return Container(
      decoration: BoxDecoration(
        color: colorScheme.background,
        border: Border.all(color: colorScheme.border),
        borderRadius: BorderRadius.circular(8),
      ),
      padding: const EdgeInsets.all(AppSpacing.space12),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            _getIcon(severity),
            color: colorScheme.icon,
            size: 20,
          ),
          const SizedBox(width: AppSpacing.space12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                if (title != null) ...[
                  Text(
                    title,
                    style: AppTypography.labelLarge.copyWith(
                      color: colorScheme.text,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 4),
                ],
                if (message.isNotEmpty)
                  Text(
                    message,
                    style: AppTypography.bodySmall.copyWith(
                      color: colorScheme.text,
                    ),
                  ),
              ],
            ),
          ),
          if (dismissible) ...[
            const SizedBox(width: AppSpacing.space8),
            IconButton(
              icon: const Icon(Icons.close, size: 16),
              onPressed: () {
                // TODO: Handle dismiss action
              },
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints(),
            ),
          ],
        ],
      ),
    );
  }

  IconData _getIcon(String severity) {
    switch (severity.toLowerCase()) {
      case 'error':
      case 'danger':
        return Icons.error;
      case 'warning':
        return Icons.warning;
      case 'success':
        return Icons.check_circle;
      case 'info':
      default:
        return Icons.info;
    }
  }

  _AlertColorScheme _getColorScheme(BuildContext context, String severity) {
    final theme = Theme.of(context);

    switch (severity.toLowerCase()) {
      case 'error':
      case 'danger':
        return _AlertColorScheme(
          background: theme.colorScheme.errorContainer,
          border: theme.colorScheme.error,
          icon: theme.colorScheme.error,
          text: theme.colorScheme.onErrorContainer,
        );

      case 'warning':
        return _AlertColorScheme(
          background: Colors.orange.shade50,
          border: Colors.orange.shade300,
          icon: Colors.orange.shade700,
          text: Colors.orange.shade900,
        );

      case 'success':
        return _AlertColorScheme(
          background: Colors.green.shade50,
          border: Colors.green.shade300,
          icon: Colors.green.shade700,
          text: Colors.green.shade900,
        );

      case 'info':
      default:
        return _AlertColorScheme(
          background: theme.colorScheme.primaryContainer,
          border: theme.colorScheme.primary,
          icon: theme.colorScheme.primary,
          text: theme.colorScheme.onPrimaryContainer,
        );
    }
  }
}

class _AlertColorScheme {
  const _AlertColorScheme({
    required this.background,
    required this.border,
    required this.icon,
    required this.text,
  });

  final Color background;
  final Color border;
  final Color icon;
  final Color text;
}
