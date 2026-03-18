/// Focus management utilities for keyboard navigation and dialog focus trapping.
library;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// Focus trap for dialogs and modals
///
/// Keeps keyboard focus within the dialog, preventing navigation to elements behind it.
/// Supports Escape key to close.
class FocusTrap extends StatefulWidget {
  const FocusTrap({
    required this.child,
    this.onEscape,
    this.autofocus = true,
    super.key,
  });

  final Widget child;
  final VoidCallback? onEscape;
  final bool autofocus;

  @override
  State<FocusTrap> createState() => _FocusTrapState();
}

class _FocusTrapState extends State<FocusTrap> {
  final FocusNode _focusNode = FocusNode();

  @override
  void initState() {
    super.initState();
    if (widget.autofocus) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _focusNode.requestFocus();
      });
    }
  }

  @override
  void dispose() {
    _focusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Focus(
      focusNode: _focusNode,
      onKeyEvent: (node, event) {
        // Handle Escape key
        if (event is KeyDownEvent &&
            event.logicalKey == LogicalKeyboardKey.escape) {
          widget.onEscape?.call();
          return KeyEventResult.handled;
        }
        return KeyEventResult.ignored;
      },
      child: FocusScope(
        child: widget.child,
      ),
    );
  }
}

/// Accessible dialog wrapper
///
/// Provides focus trap, Escape handling, and proper semantics.
class AccessibleDialog extends StatelessWidget {
  const AccessibleDialog({
    required this.title,
    required this.content,
    this.actions,
    this.onClose,
    super.key,
  });

  final String title;
  final Widget content;
  final List<Widget>? actions;
  final VoidCallback? onClose;

  @override
  Widget build(BuildContext context) {
    return FocusTrap(
      onEscape: onClose ?? () => Navigator.of(context).pop(),
      child: Semantics(
        namesRoute: true,
        scopesRoute: true,
        explicitChildNodes: true,
        label: 'Dialog: $title',
        child: AlertDialog(
          title: Text(title),
          content: content,
          actions: actions,
        ),
      ),
    );
  }
}

/// Keyboard navigation helper
///
/// Provides utilities for managing keyboard focus order.
class KeyboardNavigation {
  KeyboardNavigation._();

  /// Request focus on a node
  static void requestFocus(BuildContext context, FocusNode node) {
    FocusScope.of(context).requestFocus(node);
  }

  /// Move focus to next field
  static void nextFocus(BuildContext context) {
    FocusScope.of(context).nextFocus();
  }

  /// Move focus to previous field
  static void previousFocus(BuildContext context) {
    FocusScope.of(context).previousFocus();
  }

  /// Unfocus current field
  static void unfocus(BuildContext context) {
    FocusScope.of(context).unfocus();
  }

  /// Check if any field has focus
  static bool hasFocus(BuildContext context) {
    return FocusScope.of(context).hasFocus;
  }

  /// Submit form (typically on Enter key)
  static void submitForm(BuildContext context, VoidCallback onSubmit) {
    unfocus(context);
    onSubmit();
  }
}

/// Focus order widget
///
/// Ensures logical tab order for dynamic components.
class FocusOrder extends StatelessWidget {
  const FocusOrder({
    required this.order,
    required this.child,
    super.key,
  });

  final double order;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return FocusTraversalOrder(
      order: NumericFocusOrder(order),
      child: child,
    );
  }
}

/// Skip link for keyboard users
///
/// Allows keyboard users to skip repetitive navigation.
class SkipLink extends StatelessWidget {
  const SkipLink({
    required this.targetKey,
    required this.label,
    super.key,
  });

  final GlobalKey targetKey;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      label: label,
      child: InkWell(
        onTap: () {
          final targetContext = targetKey.currentContext;
          if (targetContext != null) {
            Scrollable.ensureVisible(
              targetContext,
              duration: const Duration(milliseconds: 300),
              curve: Curves.easeInOut,
            );
            FocusScope.of(targetContext).requestFocus();
          }
        },
        child: Padding(
          padding: const EdgeInsets.all(16.0),
          child: Text(label),
        ),
      ),
    );
  }
}
