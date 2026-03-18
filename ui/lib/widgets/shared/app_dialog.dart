/// Modal dialog with consistent structure.
library;

import 'package:flutter/material.dart';

import '../../design/design.dart';
import 'app_button.dart';

/// Dialog component with title, body, and actions
class AppDialog extends StatelessWidget {
  const AppDialog({
    required this.title,
    required this.body,
    this.confirmLabel = 'Confirm',
    this.cancelLabel = 'Cancel',
    this.onConfirm,
    this.onCancel,
    this.isDestructive = false,
    this.showCloseButton = true,
    this.barrierDismissible = true,
    super.key,
  });

  /// Dialog title
  final String title;

  /// Dialog body content
  final Widget body;

  /// Confirm button label
  final String confirmLabel;

  /// Cancel button label
  final String cancelLabel;

  /// Confirm callback
  final VoidCallback? onConfirm;

  /// Cancel callback (null = no cancel button)
  final VoidCallback? onCancel;

  /// Use destructive styling (red confirm button)
  final bool isDestructive;

  /// Show close button in title bar
  final bool showCloseButton;

  /// Allow closing by tapping outside
  final bool barrierDismissible;

  @override
  Widget build(BuildContext context) {
    return Dialog(
      child: ConstrainedBox(
        constraints: const BoxConstraints(
          maxWidth: 500,
          minWidth: 280,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            // Title bar
            Padding(
              padding: const EdgeInsets.all(AppSpacing.space16),
              child: Row(
                children: [
                  Expanded(
                    child: Text(
                      title,
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                  ),
                  if (showCloseButton)
                    IconButton(
                      icon: const Icon(Icons.close),
                      onPressed: () => Navigator.of(context).pop(),
                      iconSize: AppSizing.iconMedium,
                    ),
                ],
              ),
            ),
            const Divider(height: 1),

            // Body
            Flexible(
              child: SingleChildScrollView(
                padding: const EdgeInsets.all(AppSpacing.space16),
                child: body,
              ),
            ),

            // Actions
            if (onConfirm != null || onCancel != null) ...[
              const Divider(height: 1),
              Padding(
                padding: const EdgeInsets.all(AppSpacing.space16),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: [
                    // Cancel button
                    if (onCancel != null) ...[
                      AppButton(
                        label: cancelLabel,
                        variant: AppButtonVariant.tertiary,
                        onPressed: () {
                          Navigator.of(context).pop();
                          onCancel!();
                        },
                      ),
                      const SizedBox(width: AppSpacing.space8),
                    ],
                    // Confirm button
                    if (onConfirm != null)
                      AppButton(
                        label: confirmLabel,
                        variant: isDestructive
                            ? AppButtonVariant.destructive
                            : AppButtonVariant.primary,
                        onPressed: () {
                          Navigator.of(context).pop();
                          onConfirm!();
                        },
                      ),
                  ],
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  /// Show dialog and return result
  static Future<T?> show<T>({
    required BuildContext context,
    required String title,
    required Widget body,
    String confirmLabel = 'Confirm',
    String cancelLabel = 'Cancel',
    VoidCallback? onConfirm,
    VoidCallback? onCancel,
    bool isDestructive = false,
    bool showCloseButton = true,
    bool barrierDismissible = true,
  }) {
    return showDialog<T>(
      context: context,
      barrierDismissible: barrierDismissible,
      builder: (context) => AppDialog(
        title: title,
        body: body,
        confirmLabel: confirmLabel,
        cancelLabel: cancelLabel,
        onConfirm: onConfirm,
        onCancel: onCancel,
        isDestructive: isDestructive,
        showCloseButton: showCloseButton,
        barrierDismissible: barrierDismissible,
      ),
    );
  }
}

/// Confirmation dialog helper
class AppConfirmDialog {
  const AppConfirmDialog._();

  /// Show confirmation dialog
  static Future<bool> show({
    required BuildContext context,
    required String title,
    required String message,
    String confirmLabel = 'Confirm',
    String cancelLabel = 'Cancel',
    bool isDestructive = false,
  }) async {
    var confirmed = false;

    await AppDialog.show(
      context: context,
      title: title,
      body: Text(message),
      confirmLabel: confirmLabel,
      cancelLabel: cancelLabel,
      isDestructive: isDestructive,
      onConfirm: () => confirmed = true,
      onCancel: () => confirmed = false,
    );

    return confirmed;
  }

  /// Show delete confirmation dialog
  static Future<bool> showDelete({
    required BuildContext context,
    required String itemName,
  }) {
    return show(
      context: context,
      title: 'Delete $itemName?',
      message: 'This action cannot be undone.',
      confirmLabel: 'Delete',
      cancelLabel: 'Cancel',
      isDestructive: true,
    );
  }
}
