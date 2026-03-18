/// Action confirmation dialog widget.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../design/design.dart';
import '../../state/actions/action_provider.dart';
import '../../state/actions/action_state.dart';
import 'app_button.dart';

/// Shows a confirmation dialog for an action
Future<bool?> showActionConfirmation(
  BuildContext context, {
  required String actionId,
  required String message,
  required bool isDestructive,
  required Map<String, dynamic> payload,
}) {
  return showDialog<bool>(
    context: context,
    builder: (context) => ActionConfirmationDialog(
      actionId: actionId,
      message: message,
      isDestructive: isDestructive,
      payload: payload,
    ),
  );
}

/// Action confirmation dialog
class ActionConfirmationDialog extends ConsumerWidget {
  const ActionConfirmationDialog({
    required this.actionId,
    required this.message,
    required this.isDestructive,
    required this.payload,
    super.key,
  });

  final String actionId;
  final String message;
  final bool isDestructive;
  final Map<String, dynamic> payload;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final actionState = ref.watch(actionProvider(actionId));

    return AlertDialog(
      title: Text(
        isDestructive ? 'Confirm Action' : 'Confirm',
        style: theme.textTheme.titleLarge,
      ),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (isDestructive)
            Container(
              padding: const EdgeInsets.all(AppSpacing.space12),
              decoration: BoxDecoration(
                color: theme.colorScheme.error.withOpacity(0.1),
                borderRadius: BorderRadius.circular(AppBorderRadius.medium),
                border: Border.all(
                  color: theme.colorScheme.error.withOpacity(0.3),
                ),
              ),
              child: Row(
                children: [
                  Icon(
                    Icons.warning_rounded,
                    color: theme.colorScheme.error,
                    size: AppSizing.iconMedium,
                  ),
                  const SizedBox(width: AppSpacing.space8),
                  Expanded(
                    child: Text(
                      'This action cannot be undone',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.error,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          if (isDestructive) const SizedBox(height: AppSpacing.space16),
          Text(
            message,
            style: theme.textTheme.bodyMedium,
          ),
          if (actionState is _Error) ...[
            const SizedBox(height: AppSpacing.space16),
            Container(
              padding: const EdgeInsets.all(AppSpacing.space12),
              decoration: BoxDecoration(
                color: theme.colorScheme.error.withOpacity(0.1),
                borderRadius: BorderRadius.circular(AppBorderRadius.medium),
              ),
              child: Row(
                children: [
                  Icon(
                    Icons.error_outline,
                    color: theme.colorScheme.error,
                    size: AppSizing.iconSmall,
                  ),
                  const SizedBox(width: AppSpacing.space8),
                  Expanded(
                    child: Text(
                      actionState.message,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.error,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ],
      ),
      actions: [
        TextButton(
          onPressed: actionState is _Submitting
              ? null
              : () {
                  ref.read(actionProvider(actionId).notifier).cancelConfirmation();
                  Navigator.of(context).pop(false);
                },
          child: const Text('Cancel'),
        ),
        AppButton(
          label: isDestructive ? 'Delete' : 'Confirm',
          icon: isDestructive ? Icons.delete : Icons.check,
          variant: isDestructive ? AppButtonVariant.destructive : AppButtonVariant.primary,
          onPressed: actionState is _Submitting
              ? null
              : () async {
                  await ref
                      .read(actionProvider(actionId).notifier)
                      .execute(payload: payload);

                  final finalState = ref.read(actionProvider(actionId));
                  if (finalState is _Success && context.mounted) {
                    Navigator.of(context).pop(true);
                  }
                },
          isLoading: actionState is _Submitting,
        ),
      ],
    );
  }
}
