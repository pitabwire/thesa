/// Dynamic form widget.
///
/// Renders a complete form from a schema with validation and submission.
library;

import 'package:flutter/material.dart';
import 'package:reactive_forms/reactive_forms.dart';

import '../core/core.dart';
import '../design/design.dart';
import '../widgets/shared/shared.dart';
import 'field_renderer.dart';
import 'form_engine.dart';

/// Dynamic form that renders from schema
class DynamicForm extends StatefulWidget {
  const DynamicForm({
    required this.schema,
    required this.onSubmit,
    this.initialData,
    this.mode = FormMode.create,
    this.submitButtonText,
    this.showCancelButton = true,
    this.onCancel,
    super.key,
  });

  final Schema schema;
  final Future<void> Function(Map<String, dynamic> data) onSubmit;
  final Map<String, dynamic>? initialData;
  final FormMode mode;
  final String? submitButtonText;
  final bool showCancelButton;
  final VoidCallback? onCancel;

  @override
  State<DynamicForm> createState() => _DynamicFormState();
}

class _DynamicFormState extends State<DynamicForm> {
  late FormEngine _engine;
  bool _isSubmitting = false;

  @override
  void initState() {
    super.initState();
    _engine = FormEngine(
      schema: widget.schema,
      onSubmit: widget.onSubmit,
      initialData: widget.initialData,
      mode: widget.mode,
    );
  }

  @override
  void dispose() {
    _engine.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return ReactiveForm(
      formGroup: _engine.form,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Form title and description
          if (widget.schema.title != null) ...[
            Text(
              widget.schema.title!,
              style: AppTypography.headlineMedium,
            ),
            const SizedBox(height: AppSpacing.space8),
          ],
          if (widget.schema.description != null) ...[
            Text(
              widget.schema.description!,
              style: AppTypography.bodyMedium.copyWith(
                color: Theme.of(context).colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: AppSpacing.space24),
          ],

          // Form fields
          ...widget.schema.fields.map((field) {
            return ReactiveValueListenableBuilder<dynamic>(
              formControl: _engine.form,
              builder: (context, form, child) {
                // Check visibility
                final isVisible = _engine.isFieldVisible(
                  field,
                  form.value as Map<String, dynamic>,
                );

                if (!isVisible) {
                  return const SizedBox.shrink();
                }

                return Padding(
                  padding: const EdgeInsets.only(bottom: AppSpacing.space16),
                  child: FieldRenderer(
                    field: field,
                    formControlName: field.name,
                  ),
                );
              },
            );
          }),

          const SizedBox(height: AppSpacing.space24),

          // Form actions
          _buildActions(context),
        ],
      ),
    );
  }

  Widget _buildActions(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.end,
      children: [
        if (widget.showCancelButton) ...[
          AppButton(
            label: 'Cancel',
            onPressed: widget.onCancel ??
                () {
                  Navigator.of(context).pop();
                },
            variant: AppButtonVariant.tertiary,
          ),
          const SizedBox(width: AppSpacing.space8),
        ],
        ReactiveFormConsumer(
          builder: (context, form, child) {
            return AppButton(
              label: widget.submitButtonText ?? _getSubmitButtonText(),
              onPressed: form.valid && !_isSubmitting ? _handleSubmit : null,
              isLoading: _isSubmitting,
              variant: AppButtonVariant.primary,
            );
          },
        ),
      ],
    );
  }

  String _getSubmitButtonText() {
    switch (widget.mode) {
      case FormMode.create:
        return 'Create';
      case FormMode.edit:
        return 'Update';
      case FormMode.view:
        return 'Close';
    }
  }

  Future<void> _handleSubmit() async {
    if (!_engine.form.valid) {
      _engine.form.markAllAsTouched();
      return;
    }

    setState(() {
      _isSubmitting = true;
    });

    try {
      await _engine.submit();

      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(
              widget.mode == FormMode.create
                  ? 'Created successfully'
                  : 'Updated successfully',
            ),
            backgroundColor: Theme.of(context).colorScheme.primary,
          ),
        );

        // Close form or navigate back
        Navigator.of(context).pop(true);
      }
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Error: ${error.toString()}'),
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
        );
      }
    } finally {
      if (mounted) {
        setState(() {
          _isSubmitting = false;
        });
      }
    }
  }
}
