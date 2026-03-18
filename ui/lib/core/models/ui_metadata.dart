/// UI metadata for display hints, colors, icons, and formatting.
///
/// Provides visual customization without changing underlying data structure.
library;

import 'package:freezed_annotation/freezed_annotation.dart';

part 'ui_metadata.freezed.dart';
part 'ui_metadata.g.dart';

/// Display hints for UI elements
@freezed
class UiMetadata with _$UiMetadata {
  const factory UiMetadata({
    /// Icon identifier (Material Icons name or custom icon key)
    String? icon,

    /// Color hex code (e.g., "#FF5722")
    String? color,

    /// Background color hex code
    String? backgroundColor,

    /// Display format hint (e.g., "currency", "relative_date", "percentage")
    String? format,

    /// Tooltip text shown on hover
    String? tooltip,

    /// CSS-like class names for styling
    List<String>? cssClasses,

    /// Additional arbitrary metadata
    Map<String, dynamic>? custom,
  }) = _UiMetadata;

  factory UiMetadata.fromJson(Map<String, dynamic> json) =>
      _$UiMetadataFromJson(json);
}
