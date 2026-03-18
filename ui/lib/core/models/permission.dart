/// Permission models for capability-driven access control.
///
/// The BFF embeds permission flags in all descriptors. The UI simply reads
/// these flags and hides/shows elements accordingly. No local permission logic.
library;

import 'package:freezed_annotation/freezed_annotation.dart';

part 'permission.freezed.dart';
part 'permission.g.dart';

/// Permission check result embedded in BFF responses
@freezed
class Permission with _$Permission {
  const factory Permission({
    /// Whether the action/component/page is allowed for the current user
    required bool allowed,

    /// Optional reason why it's not allowed (for debugging, not shown to user)
    String? reason,

    /// Required capabilities for this permission
    List<String>? requiredCapabilities,
  }) = _Permission;

  factory Permission.fromJson(Map<String, dynamic> json) =>
      _$PermissionFromJson(json);

  /// Helper: create an "allowed" permission
  factory Permission.allowed() => const Permission(allowed: true);

  /// Helper: create a "denied" permission with optional reason
  factory Permission.denied([String? reason]) => Permission(
        allowed: false,
        reason: reason,
      );
}
