/// Filter panel for tables and lists.
library;

import 'package:flutter/material.dart';

import '../../core/core.dart';
import '../../design/design.dart';
import 'app_button.dart';

/// Filter panel widget
class AppFilterPanel extends StatefulWidget {
  const AppFilterPanel({
    required this.filters,
    required this.onApply,
    this.onClear,
    this.initialValues = const {},
    super.key,
  });

  final List<FilterDescriptor> filters;
  final void Function(Map<String, dynamic> values) onApply;
  final VoidCallback? onClear;
  final Map<String, dynamic> initialValues;

  @override
  State<AppFilterPanel> createState() => _AppFilterPanelState();
}

class _AppFilterPanelState extends State<AppFilterPanel> {
  late Map<String, dynamic> _values;

  @override
  void initState() {
    super.initState();
    _values = Map.from(widget.initialValues);
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.space16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          mainAxisSize: MainAxisSize.min,
          children: [
            // Header
            Row(
              children: [
                const Icon(Icons.filter_list, size: 20),
                const SizedBox(width: AppSpacing.space8),
                Text(
                  'Filters',
                  style: AppTypography.titleMedium,
                ),
                const Spacer(),
                if (widget.onClear != null)
                  TextButton(
                    onPressed: _handleClear,
                    child: const Text('Clear all'),
                  ),
              ],
            ),

            const SizedBox(height: AppSpacing.space16),
            const Divider(),
            const SizedBox(height: AppSpacing.space16),

            // Filter fields
            ...widget.filters.map((filter) {
              return Padding(
                padding: const EdgeInsets.only(bottom: AppSpacing.space16),
                child: _buildFilterField(filter),
              );
            }),

            const SizedBox(height: AppSpacing.space8),

            // Actions
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                AppButton(
                  label: 'Cancel',
                  onPressed: () => Navigator.of(context).pop(),
                  variant: AppButtonVariant.tertiary,
                  size: AppButtonSize.small,
                ),
                const SizedBox(width: AppSpacing.space8),
                AppButton(
                  label: 'Apply',
                  onPressed: _handleApply,
                  variant: AppButtonVariant.primary,
                  size: AppButtonSize.small,
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildFilterField(FilterDescriptor filter) {
    switch (filter.type) {
      case FilterType.text:
        return TextField(
          decoration: InputDecoration(
            labelText: filter.label,
            border: const OutlineInputBorder(),
          ),
          onChanged: (value) {
            setState(() {
              _values[filter.field] = value;
            });
          },
          controller: TextEditingController(
            text: _values[filter.field]?.toString() ?? '',
          ),
        );

      case FilterType.enumType:
        return DropdownButtonFormField<String>(
          decoration: InputDecoration(
            labelText: filter.label,
            border: const OutlineInputBorder(),
          ),
          value: _values[filter.field] as String?,
          items: filter.options?.map((option) {
            return DropdownMenuItem(
              value: option.value,
              child: Text(option.label),
            );
          }).toList(),
          onChanged: (value) {
            setState(() {
              _values[filter.field] = value;
            });
          },
        );

      case FilterType.boolean:
        return CheckboxListTile(
          title: Text(filter.label),
          value: _values[filter.field] as bool? ?? false,
          onChanged: (value) {
            setState(() {
              _values[filter.field] = value;
            });
          },
        );

      case FilterType.numberRange:
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(filter.label, style: AppTypography.labelMedium),
            const SizedBox(height: AppSpacing.space8),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    decoration: const InputDecoration(
                      labelText: 'Min',
                      border: OutlineInputBorder(),
                    ),
                    keyboardType: TextInputType.number,
                    onChanged: (value) {
                      setState(() {
                        _values['${filter.field}_min'] = num.tryParse(value);
                      });
                    },
                  ),
                ),
                const SizedBox(width: AppSpacing.space8),
                Expanded(
                  child: TextField(
                    decoration: const InputDecoration(
                      labelText: 'Max',
                      border: OutlineInputBorder(),
                    ),
                    keyboardType: TextInputType.number,
                    onChanged: (value) {
                      setState(() {
                        _values['${filter.field}_max'] = num.tryParse(value);
                      });
                    },
                  ),
                ),
              ],
            ),
          ],
        );

      case FilterType.dateRange:
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(filter.label, style: AppTypography.labelMedium),
            const SizedBox(height: AppSpacing.space8),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton.icon(
                    onPressed: () async {
                      final date = await showDatePicker(
                        context: context,
                        initialDate: DateTime.now(),
                        firstDate: DateTime(2000),
                        lastDate: DateTime(2100),
                      );
                      if (date != null) {
                        setState(() {
                          _values['${filter.field}_from'] = date;
                        });
                      }
                    },
                    icon: const Icon(Icons.calendar_today, size: 16),
                    label: const Text('From'),
                  ),
                ),
                const SizedBox(width: AppSpacing.space8),
                Expanded(
                  child: OutlinedButton.icon(
                    onPressed: () async {
                      final date = await showDatePicker(
                        context: context,
                        initialDate: DateTime.now(),
                        firstDate: DateTime(2000),
                        lastDate: DateTime(2100),
                      );
                      if (date != null) {
                        setState(() {
                          _values['${filter.field}_to'] = date;
                        });
                      }
                    },
                    icon: const Icon(Icons.calendar_today, size: 16),
                    label: const Text('To'),
                  ),
                ),
              ],
            ),
          ],
        );
    }
  }

  void _handleApply() {
    widget.onApply(_values);
    Navigator.of(context).pop();
  }

  void _handleClear() {
    setState(() {
      _values.clear();
    });
    widget.onClear?.call();
  }
}
