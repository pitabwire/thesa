/// Accessible widget wrappers for common interactive elements.
library;

import 'package:flutter/material.dart';

import 'accessibility_utils.dart';

/// Accessible button wrapper
///
/// Ensures minimum touch target size and proper semantics.
class AccessibleButton extends StatelessWidget {
  const AccessibleButton({
    required this.onPressed,
    required this.child,
    this.tooltip,
    this.semanticLabel,
    this.enabled = true,
    super.key,
  });

  final VoidCallback? onPressed;
  final Widget child;
  final String? tooltip;
  final String? semanticLabel;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    Widget button = MaterialButton(
      onPressed: enabled ? onPressed : null,
      materialTapTargetSize: MaterialTapTargetSize.padded,
      child: child,
    );

    // Add tooltip for icon-only buttons
    if (tooltip != null) {
      button = Tooltip(
        message: tooltip!,
        child: button,
      );
    }

    // Add semantic label if provided
    if (semanticLabel != null) {
      button = Semantics(
        label: semanticLabel,
        button: true,
        enabled: enabled,
        child: button,
      );
    }

    return button;
  }
}

/// Accessible icon button wrapper
///
/// Always includes tooltip for screen readers.
class AccessibleIconButton extends StatelessWidget {
  const AccessibleIconButton({
    required this.icon,
    required this.onPressed,
    required this.tooltip,
    this.semanticLabel,
    super.key,
  });

  final IconData icon;
  final VoidCallback? onPressed;
  final String tooltip;
  final String? semanticLabel;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      label: semanticLabel ?? tooltip,
      button: true,
      enabled: onPressed != null,
      child: Tooltip(
        message: tooltip,
        child: IconButton(
          icon: Icon(icon),
          onPressed: onPressed,
          iconSize: 24,
          constraints: BoxConstraints(
            minWidth: kMinimumTouchTarget,
            minHeight: kMinimumTouchTarget,
          ),
        ),
      ),
    );
  }
}

/// Accessible checkbox wrapper
///
/// Ensures proper semantics and touch target.
class AccessibleCheckbox extends StatelessWidget {
  const AccessibleCheckbox({
    required this.value,
    required this.onChanged,
    this.label,
    this.semanticLabel,
    super.key,
  });

  final bool value;
  final ValueChanged<bool?>? onChanged;
  final String? label;
  final String? semanticLabel;

  @override
  Widget build(BuildContext context) {
    final checkbox = Checkbox(
      value: value,
      onChanged: onChanged,
      materialTapTargetSize: MaterialTapTargetSize.padded,
    );

    if (label != null) {
      return Semantics(
        label: semanticLabel ?? label,
        checked: value,
        child: InkWell(
          onTap: onChanged != null ? () => onChanged!(!value) : null,
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 8.0),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                checkbox,
                const SizedBox(width: 8),
                Expanded(child: Text(label!)),
              ],
            ),
          ),
        ),
      );
    }

    return Semantics(
      label: semanticLabel,
      checked: value,
      child: checkbox,
    );
  }
}

/// Accessible text field wrapper
///
/// Ensures proper labels and error announcement.
class AccessibleTextField extends StatelessWidget {
  const AccessibleTextField({
    required this.controller,
    this.label,
    this.hint,
    this.helperText,
    this.errorText,
    this.onChanged,
    this.enabled = true,
    this.obscureText = false,
    this.keyboardType,
    this.maxLines = 1,
    super.key,
  });

  final TextEditingController controller;
  final String? label;
  final String? hint;
  final String? helperText;
  final String? errorText;
  final ValueChanged<String>? onChanged;
  final bool enabled;
  final bool obscureText;
  final TextInputType? keyboardType;
  final int? maxLines;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      label: label,
      textField: true,
      enabled: enabled,
      // Announce errors as live region
      liveRegion: errorText != null,
      child: TextField(
        controller: controller,
        decoration: InputDecoration(
          labelText: label,
          hintText: hint,
          helperText: helperText,
          errorText: errorText,
          border: const OutlineInputBorder(),
        ),
        enabled: enabled,
        obscureText: obscureText,
        keyboardType: keyboardType,
        maxLines: maxLines,
        onChanged: onChanged,
      ),
    );
  }
}

/// Accessible badge/status indicator
///
/// Announces status to screen readers.
class AccessibleBadge extends StatelessWidget {
  const AccessibleBadge({
    required this.label,
    required this.semanticLabel,
    this.color,
    super.key,
  });

  final String label;
  final String semanticLabel;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      label: semanticLabel,
      readOnly: true,
      child: Chip(
        label: Text(label),
        backgroundColor: color,
        labelPadding: const EdgeInsets.symmetric(horizontal: 8),
      ),
    );
  }
}

/// Decorative widget wrapper
///
/// Excludes decorative elements from screen reader navigation.
class Decorative extends StatelessWidget {
  const Decorative({
    required this.child,
    super.key,
  });

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return ExcludeSemantics(child: child);
  }
}

/// Live region for announcing dynamic content changes
///
/// Use this to announce status changes, errors, or other important updates.
class LiveRegion extends StatelessWidget {
  const LiveRegion({
    required this.message,
    required this.child,
    this.politeness = Assertiveness.polite,
    super.key,
  });

  final String message;
  final Widget child;
  final Assertiveness politeness;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      liveRegion: true,
      label: message,
      child: child,
    );
  }
}

/// Assertiveness level for live regions
enum Assertiveness {
  /// Polite: announce when current speech finishes
  polite,

  /// Assertive: interrupt current speech
  assertive,
}
