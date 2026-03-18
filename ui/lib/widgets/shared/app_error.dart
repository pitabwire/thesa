/// Error display widgets.
library;

import 'package:flutter/material.dart';

import '../../core/core.dart';
import '../../design/design.dart';
import 'app_button.dart';

/// Error widget with message and optional retry action
class AppErrorWidget extends StatelessWidget {
  const AppErrorWidget({
    required this.error,
    this.onRetry,
    this.compact = false,
    super.key,
  });

  /// Error object or message
  final Object error;

  /// Optional retry callback
  final VoidCallback? onRetry;

  /// Use compact layout
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    // Extract error message
    final message = _getErrorMessage(error);
    final icon = _getErrorIcon(error);

    if (compact) {
      return Padding(
        padding: const EdgeInsets.all(AppSpacing.space16),
        child: Row(
          children: [
            Icon(
              icon,
              color: theme.colorScheme.error,
              size: AppSizing.iconMedium,
            ),
            const SizedBox(width: AppSpacing.space12),
            Expanded(
              child: Text(
                message,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.error,
                ),
              ),
            ),
            if (onRetry != null) ...[
              const SizedBox(width: AppSpacing.space8),
              IconButton(
                icon: const Icon(Icons.refresh),
                onPressed: onRetry,
                tooltip: 'Retry',
              ),
            ],
          ],
        ),
      );
    }

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.space24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              icon,
              size: 64,
              color: theme.colorScheme.error,
            ),
            const SizedBox(height: AppSpacing.space16),
            Text(
              'Something went wrong',
              style: theme.textTheme.titleLarge,
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: AppSpacing.space8),
            Text(
              message,
              style: theme.textTheme.bodyMedium,
              textAlign: TextAlign.center,
            ),
            if (onRetry != null) ...[
              const SizedBox(height: AppSpacing.space24),
              AppButton(
                label: 'Try Again',
                icon: Icons.refresh,
                onPressed: onRetry,
              ),
            ],
          ],
        ),
      ),
    );
  }

  /// Extract user-friendly error message
  String _getErrorMessage(Object error) {
    if (error is AppError) {
      return error.map(
        network: (e) => e.message,
        permission: (e) => e.message,
        validation: (e) => e.message,
        cache: (e) => e.message,
        parse: (e) => 'Failed to process server response',
        notFound: (e) => e.message,
        config: (e) => e.message,
        rendering: (e) => e.message,
        workflow: (e) => e.message,
        unknown: (e) => e.message,
      );
    }

    return error.toString();
  }

  /// Get appropriate icon for error type
  IconData _getErrorIcon(Object error) {
    if (error is AppError) {
      return error.map(
        network: (_) => Icons.cloud_off,
        permission: (_) => Icons.lock,
        validation: (_) => Icons.error_outline,
        cache: (_) => Icons.storage,
        parse: (_) => Icons.warning,
        notFound: (_) => Icons.search_off,
        config: (_) => Icons.settings,
        rendering: (_) => Icons.error,
        workflow: (_) => Icons.list_alt,
        unknown: (_) => Icons.error,
      );
    }

    return Icons.error;
  }
}

/// Empty state widget
class AppEmptyState extends StatelessWidget {
  const AppEmptyState({
    required this.message,
    this.icon = Icons.inbox,
    this.action,
    this.actionLabel,
    super.key,
  });

  /// Empty state message
  final String message;

  /// Icon to display
  final IconData icon;

  /// Optional action button callback
  final VoidCallback? action;

  /// Optional action button label
  final String? actionLabel;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.space24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              icon,
              size: 64,
              color: theme.colorScheme.onSurface.withOpacity(0.4),
            ),
            const SizedBox(height: AppSpacing.space16),
            Text(
              message,
              style: theme.textTheme.bodyLarge,
              textAlign: TextAlign.center,
            ),
            if (action != null && actionLabel != null) ...[
              const SizedBox(height: AppSpacing.space24),
              AppButton(
                label: actionLabel!,
                onPressed: action,
                variant: AppButtonVariant.secondary,
              ),
            ],
          ],
        ),
      ),
    );
  }
}
