/// Schema definitions for dynamic forms and data structures.
///
/// Schemas describe the structure, types, and validation rules for data.
/// Used by:
/// - Form engine to generate input forms
/// - Table engine to understand column types
/// - Validation engine to validate user input
library;

import 'package:freezed_annotation/freezed_annotation.dart';

import 'ui_metadata.dart';

part 'schema.freezed.dart';
part 'schema.g.dart';

/// Schema definition for a data structure
@freezed
class Schema with _$Schema {
  const factory Schema({
    /// Unique identifier for this schema
    required String schemaId,

    /// Human-readable name
    String? title,

    /// Description of what this schema represents
    String? description,

    /// List of fields in this schema
    @Default([]) List<SchemaField> fields,

    /// Schema composition (allOf, oneOf, anyOf)
    SchemaComposition? composition,

    /// Version of this schema (for cache invalidation)
    String? version,

    /// Whether this schema extends another schema
    @JsonKey(name: r'$ref') String? ref,
  }) = _Schema;

  factory Schema.fromJson(Map<String, dynamic> json) => _$SchemaFromJson(json);
}

/// A single field definition within a schema
@freezed
class SchemaField with _$SchemaField {
  const factory SchemaField({
    /// Field name (key in the data object)
    required String name,

    /// Field type
    required FieldType type,

    /// Human-readable label
    String? label,

    /// Help text shown below the field
    String? helpText,

    /// Placeholder text for empty fields
    String? placeholder,

    /// Default value when creating new entries
    dynamic defaultValue,

    /// Whether this field is required
    @Default(false) bool required,

    /// Whether this field is read-only
    @Default(false) bool readonly,

    /// Whether this field is for multiline text
    @Default(false) bool multiline,

    /// Validation rules
    ValidationRules? validation,

    /// Visibility rules (conditional display)
    VisibilityRule? visibleWhen,

    /// For enum/reference types: available options
    List<FieldOption>? options,

    /// For reference type: which resource to reference
    String? resource,

    /// For reference type: which field to display
    String? displayField,

    /// For array type: schema of array items
    Schema? items,

    /// For object type: nested schema
    Schema? properties,

    /// UI metadata (icon, color, format hints)
    UiMetadata? ui,

    /// Priority level for responsive tables (1=always show, 5=hide first)
    @Default(3) int priority,

    /// Whether this field can be sorted (for tables)
    @Default(false) bool sortable,

    /// Whether this field can be filtered (for tables)
    @Default(false) bool filterable,

    /// Whether this field is searchable (for full-text search)
    @Default(false) bool searchable,
  }) = _SchemaField;

  factory SchemaField.fromJson(Map<String, dynamic> json) =>
      _$SchemaFieldFromJson(json);
}

/// Field data types
enum FieldType {
  /// String/text
  string,

  /// Integer number
  number,

  /// Decimal number
  decimal,

  /// Money/currency
  money,

  /// Boolean (true/false)
  boolean,

  /// Date (no time)
  date,

  /// Date with time
  datetime,

  /// Time only
  time,

  /// Enum/select from predefined list
  @JsonValue('enum')
  enumType,

  /// Reference to another resource
  reference,

  /// Array/list of items
  array,

  /// Nested object
  object,

  /// File upload
  file,

  /// Image upload
  image,

  /// Rich text / HTML editor
  richText,

  /// URL
  url,

  /// Email address
  email,

  /// Phone number
  phone,

  /// Custom type (requires plugin)
  custom,
}

/// Validation rules for a field
@freezed
class ValidationRules with _$ValidationRules {
  const factory ValidationRules({
    /// Minimum length (for strings)
    int? minLength,

    /// Maximum length (for strings)
    int? maxLength,

    /// Minimum value (for numbers)
    num? min,

    /// Maximum value (for numbers)
    num? max,

    /// Regular expression pattern
    String? pattern,

    /// Pattern description (shown in error message)
    String? patternDescription,

    /// Must be a valid email
    @Default(false) bool email,

    /// Must be a valid URL
    @Default(false) bool url,

    /// Must be a valid phone number
    @Default(false) bool phone,

    /// Custom validation rule name (evaluated by BFF)
    String? custom,

    /// Custom error message
    String? errorMessage,
  }) = _ValidationRules;

  factory ValidationRules.fromJson(Map<String, dynamic> json) =>
      _$ValidationRulesFromJson(json);
}

/// Conditional visibility rule
@freezed
class VisibilityRule with _$VisibilityRule {
  const factory VisibilityRule({
    /// Field name to watch
    required String field,

    /// Show when field equals this value
    dynamic equals,

    /// Show when field is one of these values
    List<dynamic>? oneOf,

    /// Show when field is not this value
    dynamic notEquals,

    /// Show when field is not empty/null
    @Default(false) bool notEmpty,

    /// Show when field is empty/null
    @Default(false) bool empty,

    /// Combine multiple rules with AND logic
    List<VisibilityRule>? allOf,

    /// Combine multiple rules with OR logic
    List<VisibilityRule>? anyOf,
  }) = _VisibilityRule;

  factory VisibilityRule.fromJson(Map<String, dynamic> json) =>
      _$VisibilityRuleFromJson(json);
}

/// Option for enum fields
@freezed
class FieldOption with _$FieldOption {
  const factory FieldOption({
    /// Value stored in the data
    required String value,

    /// Label shown to the user
    required String label,

    /// Optional icon
    String? icon,

    /// Optional color
    String? color,

    /// Optional group (for grouped dropdowns)
    String? group,

    /// Whether this option is disabled
    @Default(false) bool disabled,
  }) = _FieldOption;

  factory FieldOption.fromJson(Map<String, dynamic> json) =>
      _$FieldOptionFromJson(json);
}

/// Schema composition (for combining schemas)
@freezed
class SchemaComposition with _$SchemaComposition {
  const factory SchemaComposition({
    /// Merge all of these schemas
    List<Schema>? allOf,

    /// Match exactly one of these schemas
    List<Schema>? oneOf,

    /// Match any of these schemas
    List<Schema>? anyOf,
  }) = _SchemaComposition;

  factory SchemaComposition.fromJson(Map<String, dynamic> json) =>
      _$SchemaCompositionFromJson(json);
}
