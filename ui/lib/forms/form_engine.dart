/// Dynamic form engine core.
///
/// Manages form state, validation, and submission using reactive_forms.
library;

import 'package:flutter/material.dart';
import 'package:reactive_forms/reactive_forms.dart';

import '../core/core.dart';

/// Form engine that manages form state and submission
class FormEngine {
  FormEngine({
    required this.schema,
    required this.onSubmit,
    this.initialData,
    this.mode = FormMode.create,
  }) {
    form = _buildForm();
  }

  /// Schema defining the form structure
  final Schema schema;

  /// Callback when form is submitted
  final Future<void> Function(Map<String, dynamic> data) onSubmit;

  /// Initial data (for edit mode)
  final Map<String, dynamic>? initialData;

  /// Form mode (create or edit)
  final FormMode mode;

  /// The reactive form group
  late final FormGroup form;

  /// Build reactive form from schema
  FormGroup _buildForm() {
    final controls = <String, AbstractControl<dynamic>>{};

    for (final field in schema.fields) {
      final control = _buildControl(field);
      controls[field.name] = control;
    }

    return FormGroup(controls);
  }

  /// Build control for a single field
  AbstractControl<dynamic> _buildControl(SchemaField field) {
    // Get initial value
    final initialValue = initialData?[field.name] ?? field.defaultValue;

    // Build validators
    final validators = <Validator<dynamic>>[];

    if (field.required) {
      validators.add(Validators.required);
    }

    if (field.validation != null) {
      validators.addAll(_buildValidators(field));
    }

    // Create control based on type
    switch (field.type) {
      case FieldType.array:
        return FormArray<dynamic>(
          [],
          validators: validators,
        );

      case FieldType.object:
        // Nested form group
        if (field.properties != null) {
          final nestedEngine = FormEngine(
            schema: field.properties!,
            onSubmit: (_) async {},
            initialData: initialValue as Map<String, dynamic>?,
          );
          return nestedEngine.form;
        }
        return FormControl<Map<String, dynamic>>(
          value: initialValue as Map<String, dynamic>?,
          validators: validators,
        );

      case FieldType.boolean:
        return FormControl<bool>(
          value: initialValue as bool? ?? false,
          validators: validators,
        );

      case FieldType.number:
      case FieldType.decimal:
      case FieldType.money:
        return FormControl<num>(
          value: initialValue as num?,
          validators: validators,
        );

      default:
        return FormControl<String>(
          value: initialValue?.toString(),
          validators: validators,
        );
    }
  }

  /// Build validators from validation rules
  List<Validator<dynamic>> _buildValidators(SchemaField field) {
    final validators = <Validator<dynamic>>[];
    final rules = field.validation!;

    if (rules.minLength != null) {
      validators.add(Validators.minLength(rules.minLength!));
    }

    if (rules.maxLength != null) {
      validators.add(Validators.maxLength(rules.maxLength!));
    }

    if (rules.min != null) {
      validators.add(Validators.min(rules.min!));
    }

    if (rules.max != null) {
      validators.add(Validators.max(rules.max!));
    }

    if (rules.pattern != null) {
      validators.add(Validators.pattern(rules.pattern!));
    }

    if (rules.email) {
      validators.add(Validators.email);
    }

    return validators;
  }

  /// Submit the form
  Future<void> submit() async {
    if (!form.valid) {
      form.markAllAsTouched();
      return;
    }

    await onSubmit(form.value);
  }

  /// Reset the form
  void reset() {
    form.reset();
  }

  /// Get field visibility based on conditional rules
  bool isFieldVisible(SchemaField field, Map<String, dynamic> formData) {
    if (field.visibleWhen == null) {
      return true;
    }

    return _evaluateVisibilityRule(field.visibleWhen!, formData);
  }

  /// Evaluate visibility rule
  bool _evaluateVisibilityRule(
    VisibilityRule rule,
    Map<String, dynamic> formData,
  ) {
    // Handle compound rules
    if (rule.allOf != null) {
      return rule.allOf!
          .every((r) => _evaluateVisibilityRule(r, formData));
    }

    if (rule.anyOf != null) {
      return rule.anyOf!
          .any((r) => _evaluateVisibilityRule(r, formData));
    }

    // Get field value
    final fieldValue = formData[rule.field];

    // Evaluate conditions
    if (rule.equals != null) {
      return fieldValue == rule.equals;
    }

    if (rule.notEquals != null) {
      return fieldValue != rule.notEquals;
    }

    if (rule.oneOf != null) {
      return rule.oneOf!.contains(fieldValue);
    }

    if (rule.notEmpty) {
      return fieldValue != null &&
          (fieldValue is String ? fieldValue.isNotEmpty : true);
    }

    if (rule.empty) {
      return fieldValue == null ||
          (fieldValue is String ? fieldValue.isEmpty : false);
    }

    return true;
  }

  /// Dispose form resources
  void dispose() {
    form.dispose();
  }
}

/// Form mode
enum FormMode {
  /// Creating a new entry
  create,

  /// Editing an existing entry
  edit,

  /// Read-only view
  view,
}
