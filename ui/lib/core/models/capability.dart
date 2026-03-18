/// Capability models for feature flags and tenant configuration.
///
/// Capabilities define what features are enabled for the current user/tenant.
/// Used for:
/// - Feature flagging
/// - Tenant-specific customization
/// - Progressive feature rollout
library;

import 'package:freezed_annotation/freezed_annotation.dart';

part 'capability.freezed.dart';
part 'capability.g.dart';

/// Complete set of capabilities for the current context
@freezed
class Capabilities with _$Capabilities {
  const factory Capabilities({
    /// Map of capability keys to their values
    @Default({}) Map<String, CapabilityValue> capabilities,

    /// User-specific capabilities
    UserCapabilities? user,

    /// Tenant/organization-specific capabilities
    TenantCapabilities? tenant,

    /// Application-level capabilities
    AppCapabilities? app,

    /// Version (for cache invalidation)
    String? version,

    /// ETag for cache validation
    String? etag,
  }) = _Capabilities;

  factory Capabilities.fromJson(Map<String, dynamic> json) =>
      _$CapabilitiesFromJson(json);
}

/// A single capability value
@freezed
class CapabilityValue with _$CapabilityValue {
  const factory CapabilityValue({
    /// Whether this capability is enabled
    required bool enabled,

    /// Optional value (for non-boolean capabilities)
    dynamic value,

    /// Optional metadata
    Map<String, dynamic>? metadata,
  }) = _CapabilityValue;

  factory CapabilityValue.fromJson(Map<String, dynamic> json) =>
      _$CapabilityValueFromJson(json);
}

/// User-specific capabilities
@freezed
class UserCapabilities with _$UserCapabilities {
  const factory UserCapabilities({
    /// User roles
    @Default([]) List<String> roles,

    /// User permissions
    @Default([]) List<String> permissions,

    /// Features enabled for this user
    @Default([]) List<String> features,

    /// User preferences
    Map<String, dynamic>? preferences,
  }) = _UserCapabilities;

  factory UserCapabilities.fromJson(Map<String, dynamic> json) =>
      _$UserCapabilitiesFromJson(json);
}

/// Tenant/organization capabilities
@freezed
class TenantCapabilities with _$TenantCapabilities {
  const factory TenantCapabilities({
    /// Tenant ID
    String? tenantId,

    /// Tenant name
    String? tenantName,

    /// Plan/tier (e.g., "free", "pro", "enterprise")
    String? plan,

    /// Features enabled for this tenant
    @Default([]) List<String> features,

    /// Feature limits (e.g., max users, max storage)
    Map<String, num>? limits,

    /// Tenant settings
    Map<String, dynamic>? settings,
  }) = _TenantCapabilities;

  factory TenantCapabilities.fromJson(Map<String, dynamic> json) =>
      _$TenantCapabilitiesFromJson(json);
}

/// Application-level capabilities
@freezed
class AppCapabilities with _$AppCapabilities {
  const factory AppCapabilities({
    /// App version
    String? version,

    /// Environment (e.g., "production", "staging")
    String? environment,

    /// Available plugins
    @Default([]) List<String> plugins,

    /// Feature flags
    Map<String, bool>? featureFlags,

    /// App configuration
    Map<String, dynamic>? config,
  }) = _AppCapabilities;

  factory AppCapabilities.fromJson(Map<String, dynamic> json) =>
      _$AppCapabilitiesFromJson(json);
}
