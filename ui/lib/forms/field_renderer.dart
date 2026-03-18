/// Field renderer for dynamic forms.
///
/// Renders form fields based on schema field types.
library;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:reactive_forms/reactive_forms.dart';

import '../core/core.dart';
import '../design/design.dart';

/// Renders a form field based on its type
class FieldRenderer extends StatelessWidget {
  const FieldRenderer({
    required this.field,
    required this.formControlName,
    super.key,
  });

  final SchemaField field;
  final String formControlName;

  @override
  Widget build(BuildContext context) {
    // Don't render if readonly in create mode
    if (field.readonly) {
      return _buildReadonlyField(context);
    }

    switch (field.type) {
      case FieldType.string:
        return _buildTextField(context);

      case FieldType.number:
      case FieldType.decimal:
      case FieldType.money:
        return _buildNumberField(context);

      case FieldType.boolean:
        return _buildBooleanField(context);

      case FieldType.date:
        return _buildDateField(context);

      case FieldType.datetime:
        return _buildDateTimeField(context);

      case FieldType.time:
        return _buildTimeField(context);

      case FieldType.enumType:
        return _buildEnumField(context);

      case FieldType.email:
        return _buildEmailField(context);

      case FieldType.phone:
        return _buildPhoneField(context);

      case FieldType.url:
        return _buildUrlField(context);

      case FieldType.richText:
        return _buildRichTextField(context);

      case FieldType.file:
      case FieldType.image:
        return _buildFileField(context);

      case FieldType.reference:
        return _buildReferenceField(context);

      case FieldType.array:
        return _buildArrayField(context);

      case FieldType.object:
        return _buildObjectField(context);

      case FieldType.custom:
        return _buildCustomField(context);
    }
  }

  Widget _buildTextField(BuildContext context) {
    return ReactiveTextField<String>(
      formControlName: formControlName,
      decoration: _inputDecoration(context),
      maxLines: field.multiline ? null : 1,
      minLines: field.multiline ? 3 : 1,
      validationMessages: _validationMessages,
    );
  }

  Widget _buildNumberField(BuildContext context) {
    return ReactiveTextField<num>(
      formControlName: formControlName,
      decoration: _inputDecoration(context),
      keyboardType: TextInputType.numberWithOptions(
        decimal: field.type == FieldType.decimal || field.type == FieldType.money,
      ),
      inputFormatters: [
        if (field.type == FieldType.number)
          FilteringTextInputFormatter.digitsOnly,
      ],
      validationMessages: _validationMessages,
    );
  }

  Widget _buildBooleanField(BuildContext context) {
    return ReactiveCheckbox(
      formControlName: formControlName,
      title: Text(field.label ?? field.name),
    );
  }

  Widget _buildDateField(BuildContext context) {
    return ReactiveDatePicker<DateTime>(
      formControlName: formControlName,
      builder: (context, picker, child) {
        return ReactiveTextField<DateTime>(
          formControlName: formControlName,
          decoration: _inputDecoration(context).copyWith(
            suffixIcon: const Icon(Icons.calendar_today),
          ),
          readOnly: true,
          onTap: (_) => picker.showPicker(),
          validationMessages: _validationMessages,
        );
      },
    );
  }

  Widget _buildDateTimeField(BuildContext context) {
    return ReactiveDatePicker<DateTime>(
      formControlName: formControlName,
      builder: (context, picker, child) {
        return ReactiveTextField<DateTime>(
          formControlName: formControlName,
          decoration: _inputDecoration(context).copyWith(
            suffixIcon: const Icon(Icons.calendar_today),
          ),
          readOnly: true,
          onTap: (_) => picker.showPicker(),
          validationMessages: _validationMessages,
        );
      },
    );
  }

  Widget _buildTimeField(BuildContext context) {
    return ReactiveTimePicker(
      formControlName: formControlName,
      builder: (context, picker, child) {
        return ReactiveTextField<TimeOfDay>(
          formControlName: formControlName,
          decoration: _inputDecoration(context).copyWith(
            suffixIcon: const Icon(Icons.access_time),
          ),
          readOnly: true,
          onTap: (_) => picker.showPicker(),
          validationMessages: _validationMessages,
        );
      },
    );
  }

  Widget _buildEnumField(BuildContext context) {
    if (field.options == null || field.options!.isEmpty) {
      return Text('No options provided for ${field.name}');
    }

    return ReactiveDropdownField<String>(
      formControlName: formControlName,
      decoration: _inputDecoration(context),
      items: field.options!.map((option) {
        return DropdownMenuItem<String>(
          value: option.value,
          child: Text(option.label),
        );
      }).toList(),
      validationMessages: _validationMessages,
    );
  }

  Widget _buildEmailField(BuildContext context) {
    return ReactiveTextField<String>(
      formControlName: formControlName,
      decoration: _inputDecoration(context),
      keyboardType: TextInputType.emailAddress,
      validationMessages: _validationMessages,
    );
  }

  Widget _buildPhoneField(BuildContext context) {
    return ReactiveTextField<String>(
      formControlName: formControlName,
      decoration: _inputDecoration(context),
      keyboardType: TextInputType.phone,
      validationMessages: _validationMessages,
    );
  }

  Widget _buildUrlField(BuildContext context) {
    return ReactiveTextField<String>(
      formControlName: formControlName,
      decoration: _inputDecoration(context),
      keyboardType: TextInputType.url,
      validationMessages: _validationMessages,
    );
  }

  Widget _buildRichTextField(BuildContext context) {
    // TODO: Implement rich text editor (markdown or HTML)
    return ReactiveTextField<String>(
      formControlName: formControlName,
      decoration: _inputDecoration(context).copyWith(
        helperText: 'Rich text editor coming soon',
      ),
      maxLines: 10,
      validationMessages: _validationMessages,
    );
  }

  Widget _buildFileField(BuildContext context) {
    // TODO: Implement file upload
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          field.label ?? field.name,
          style: AppTypography.labelMedium,
        ),
        const SizedBox(height: AppSpacing.space8),
        OutlinedButton.icon(
          onPressed: () {
            // TODO: Open file picker
          },
          icon: const Icon(Icons.upload_file),
          label: Text('Choose ${field.type == FieldType.image ? "Image" : "File"}'),
        ),
        if (field.helpText != null) ...[
          const SizedBox(height: 4),
          Text(
            field.helpText!,
            style: AppTypography.bodySmall.copyWith(
              color: Theme.of(context).colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ],
    );
  }

  Widget _buildReferenceField(BuildContext context) {
    // TODO: Implement reference field with search/autocomplete
    return ReactiveTextField<String>(
      formControlName: formControlName,
      decoration: _inputDecoration(context).copyWith(
        suffixIcon: const Icon(Icons.search),
        helperText: 'Reference: ${field.resource}',
      ),
      readOnly: true,
      onTap: (_) {
        // TODO: Open reference picker dialog
      },
      validationMessages: _validationMessages,
    );
  }

  Widget _buildArrayField(BuildContext context) {
    // TODO: Implement array field with add/remove items
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          field.label ?? field.name,
          style: AppTypography.labelMedium,
        ),
        const SizedBox(height: AppSpacing.space8),
        Container(
          padding: const EdgeInsets.all(AppSpacing.space12),
          decoration: BoxDecoration(
            border: Border.all(
              color: Theme.of(context).colorScheme.outline,
            ),
            borderRadius: BorderRadius.circular(8),
          ),
          child: const Text('Array field coming soon'),
        ),
      ],
    );
  }

  Widget _buildObjectField(BuildContext context) {
    // TODO: Implement nested object rendering
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          field.label ?? field.name,
          style: AppTypography.labelMedium,
        ),
        const SizedBox(height: AppSpacing.space8),
        Container(
          padding: const EdgeInsets.all(AppSpacing.space12),
          decoration: BoxDecoration(
            border: Border.all(
              color: Theme.of(context).colorScheme.outline,
            ),
            borderRadius: BorderRadius.circular(8),
          ),
          child: const Text('Nested object field coming soon'),
        ),
      ],
    );
  }

  Widget _buildCustomField(BuildContext context) {
    // Plugin hook for custom field types
    return Container(
      padding: const EdgeInsets.all(AppSpacing.space12),
      decoration: BoxDecoration(
        border: Border.all(
          color: Theme.of(context).colorScheme.outline,
        ),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text('Custom field type: ${field.name}'),
    );
  }

  Widget _buildReadonlyField(BuildContext context) {
    return ReactiveValueListenableBuilder<dynamic>(
      formControlName: formControlName,
      builder: (context, control, child) {
        final value = control.value?.toString() ?? '-';
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              field.label ?? field.name,
              style: AppTypography.labelSmall.copyWith(
                color: Theme.of(context).colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 4),
            Text(
              value,
              style: AppTypography.bodyMedium,
            ),
          ],
        );
      },
    );
  }

  InputDecoration _inputDecoration(BuildContext context) {
    return InputDecoration(
      labelText: field.label ?? field.name,
      hintText: field.placeholder,
      helperText: field.helpText,
      border: const OutlineInputBorder(),
    );
  }

  Map<String, String Function(Object)> get _validationMessages {
    return {
      ValidationMessage.required: (_) =>
          '${field.label ?? field.name} is required',
      ValidationMessage.minLength: (error) =>
          'Must be at least ${(error as Map)['requiredLength']} characters',
      ValidationMessage.maxLength: (error) =>
          'Must be at most ${(error as Map)['requiredLength']} characters',
      ValidationMessage.min: (error) =>
          'Must be at least ${(error as Map)['min']}',
      ValidationMessage.max: (error) =>
          'Must be at most ${(error as Map)['max']}',
      ValidationMessage.pattern: (_) =>
          field.validation?.patternDescription ??
          'Invalid format for ${field.label ?? field.name}',
      ValidationMessage.email: (_) => 'Must be a valid email address',
      if (field.validation?.errorMessage != null)
        'custom': (_) => field.validation!.errorMessage!,
    };
  }
}
