/// Loading indicator widgets.
library;

import 'package:flutter/material.dart';

import '../../design/design.dart';

/// Loading indicator sizes
enum AppLoadingSize {
  small,
  medium,
  large,
}

/// Centered loading indicator
class AppLoadingIndicator extends StatelessWidget {
  const AppLoadingIndicator({
    this.size = AppLoadingSize.medium,
    this.message,
    super.key,
  });

  /// Indicator size
  final AppLoadingSize size;

  /// Optional loading message
  final String? message;

  @override
  Widget build(BuildContext context) {
    final indicatorSize = switch (size) {
      AppLoadingSize.small => 24.0,
      AppLoadingSize.medium => 40.0,
      AppLoadingSize.large => 64.0,
    };

    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          SizedBox(
            width: indicatorSize,
            height: indicatorSize,
            child: CircularProgressIndicator(
              strokeWidth: size == AppLoadingSize.small ? 2 : 3,
            ),
          ),
          if (message != null) ...[
            const SizedBox(height: AppSpacing.space16),
            Text(
              message!,
              style: Theme.of(context).textTheme.bodyMedium,
              textAlign: TextAlign.center,
            ),
          ],
        ],
      ),
    );
  }
}

/// Full-screen loading overlay
class AppLoadingOverlay extends StatelessWidget {
  const AppLoadingOverlay({
    this.message,
    super.key,
  });

  /// Optional loading message
  final String? message;

  @override
  Widget build(BuildContext context) {
    return Container(
      color: Colors.black54,
      child: AppLoadingIndicator(
        size: AppLoadingSize.large,
        message: message,
      ),
    );
  }

  /// Show loading overlay
  static void show(
    BuildContext context, {
    String? message,
  }) {
    showDialog(
      context: context,
      barrierDismissible: false,
      builder: (context) => AppLoadingOverlay(message: message),
    );
  }

  /// Hide loading overlay
  static void hide(BuildContext context) {
    Navigator.of(context).pop();
  }
}

/// Inline loading widget for lists
class AppLoadingListItem extends StatelessWidget {
  const AppLoadingListItem({super.key});

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.all(AppSpacing.space16),
      child: Center(
        child: AppLoadingIndicator(size: AppLoadingSize.small),
      ),
    );
  }
}
