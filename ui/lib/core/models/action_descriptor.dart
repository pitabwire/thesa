/// Action descriptors for user-triggerable operations.
///
/// Actions can be:
/// - Page-level buttons (e.g., "Create New Order")
/// - Row-level operations (e.g., "Edit", "Delete")
/// - Bulk operations (e.g., "Export CSV", "Mark as Shipped")
library;

import 'package:freezed_annotation/freezed_annotation.dart';

import 'permission.dart';
import 'ui_metadata.dart';

part 'action_descriptor.freezed.dart';
part 'action_descriptor.g.dart';

/// Describes an action that the user can trigger
@freezed
class ActionDescriptor with _$ActionDescriptor {
  const factory ActionDescriptor({
    /// Unique identifier for this action
    required String actionId,

    /// Display label shown to the user
    required String label,

    /// Position where this action should appear
    @Default(ActionPosition.context) ActionPosition position,

    /// Type of action
    @Default(ActionType.button) ActionType type,

    /// UI metadata (icon, color, etc.)
    UiMetadata? ui,

    /// HTTP method for API call
    @Default('POST') String method,

    /// API endpoint to call (may contain placeholders like {id})
    String? endpoint,

    /// Schema ID for input form (if action requires user input)
    String? inputSchema,

    /// Navigation path if action navigates (e.g., "/orders/{id}")
    String? navigation,

    /// Workflow ID if action starts a workflow
    String? workflowId,

    /// Confirmation message before executing
    String? confirmationMessage,

    /// Success message after execution
    String? successMessage,

    /// Permission check
    @Default(Permission(allowed: true)) Permission permission,

    /// Whether this action appears in bulk operations
    @Default(false) bool supportsBulk,

    /// Whether to open action in modal dialog vs inline
    @Default(false) bool modal,

    /// Custom action handler name (for plugin override)
    String? customHandler,
  }) = _ActionDescriptor;

  factory ActionDescriptor.fromJson(Map<String, dynamic> json) =>
      _$ActionDescriptorFromJson(json);
}

/// Where the action button should appear
enum ActionPosition {
  /// Page header (top-right)
  header,

  /// Context menu (right-click or row menu)
  context,

  /// Inline within a table row
  inline,

  /// Bottom action bar
  footer,

  /// Floating action button
  fab,
}

/// Type of action execution
enum ActionType {
  /// Standard button action
  button,

  /// Link/navigation action
  link,

  /// Menu item
  menuItem,

  /// Icon-only button
  iconButton,

  /// Custom action type (requires plugin)
  custom,
}
