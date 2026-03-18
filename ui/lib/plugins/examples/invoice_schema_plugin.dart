/// Example schema renderer plugin demonstrating custom form rendering.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/core.dart';
import '../../design/design.dart';
import '../../widgets/shared/shared.dart';

/// Example invoice schema renderer plugin
///
/// Demonstrates custom form rendering for a specific schema
Widget buildInvoiceForm(
  Schema schema,
  WidgetRef ref,
  ValueChanged<Map<String, dynamic>> onSaved,
) {
  return InvoiceFormPlugin(
    schema: schema,
    onSaved: onSaved,
  );
}

/// Custom invoice form implementation
///
/// This is a specialized form for invoice data entry that provides
/// a better UX than the generic form renderer
class InvoiceFormPlugin extends StatefulWidget {
  const InvoiceFormPlugin({
    required this.schema,
    required this.onSaved,
    super.key,
  });

  final Schema schema;
  final ValueChanged<Map<String, dynamic>> onSaved;

  @override
  State<InvoiceFormPlugin> createState() => _InvoiceFormPluginState();
}

class _InvoiceFormPluginState extends State<InvoiceFormPlugin> {
  final _formKey = GlobalKey<FormState>();
  final _invoiceNumberController = TextEditingController();
  final _customerController = TextEditingController();
  final _amountController = TextEditingController();
  final _notesController = TextEditingController();

  DateTime _dueDate = DateTime.now().add(const Duration(days: 30));
  String _status = 'draft';

  final List<_LineItem> _lineItems = [
    _LineItem(description: '', quantity: 1, unitPrice: 0.0),
  ];

  @override
  void dispose() {
    _invoiceNumberController.dispose();
    _customerController.dispose();
    _amountController.dispose();
    _notesController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Form(
      key: _formKey,
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(AppSpacing.space16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Header
            Text(
              'Create Invoice',
              style: theme.textTheme.headlineMedium,
            ),
            const SizedBox(height: AppSpacing.space8),
            Text(
              'Custom invoice form with line items',
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurface.withOpacity(0.7),
              ),
            ),
            const SizedBox(height: AppSpacing.space24),

            // Invoice details section
            _buildSectionHeader(theme, 'Invoice Details'),
            const SizedBox(height: AppSpacing.space16),

            Row(
              children: [
                Expanded(
                  child: TextFormField(
                    controller: _invoiceNumberController,
                    decoration: const InputDecoration(
                      labelText: 'Invoice Number',
                      hintText: 'INV-001',
                      border: OutlineInputBorder(),
                    ),
                    validator: (value) =>
                        value?.isEmpty ?? true ? 'Required' : null,
                  ),
                ),
                const SizedBox(width: AppSpacing.space16),
                Expanded(
                  child: TextFormField(
                    controller: _customerController,
                    decoration: const InputDecoration(
                      labelText: 'Customer',
                      hintText: 'Customer name',
                      border: OutlineInputBorder(),
                    ),
                    validator: (value) =>
                        value?.isEmpty ?? true ? 'Required' : null,
                  ),
                ),
              ],
            ),

            const SizedBox(height: AppSpacing.space16),

            Row(
              children: [
                Expanded(
                  child: InkWell(
                    onTap: () => _selectDueDate(context),
                    child: InputDecorator(
                      decoration: const InputDecoration(
                        labelText: 'Due Date',
                        border: OutlineInputBorder(),
                      ),
                      child: Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          Text(
                            '${_dueDate.year}-${_dueDate.month.toString().padLeft(2, '0')}-${_dueDate.day.toString().padLeft(2, '0')}',
                          ),
                          const Icon(Icons.calendar_today),
                        ],
                      ),
                    ),
                  ),
                ),
                const SizedBox(width: AppSpacing.space16),
                Expanded(
                  child: DropdownButtonFormField<String>(
                    value: _status,
                    decoration: const InputDecoration(
                      labelText: 'Status',
                      border: OutlineInputBorder(),
                    ),
                    items: const [
                      DropdownMenuItem(value: 'draft', child: Text('Draft')),
                      DropdownMenuItem(value: 'sent', child: Text('Sent')),
                      DropdownMenuItem(value: 'paid', child: Text('Paid')),
                    ],
                    onChanged: (value) => setState(() => _status = value!),
                  ),
                ),
              ],
            ),

            const SizedBox(height: AppSpacing.space32),

            // Line items section
            _buildSectionHeader(theme, 'Line Items'),
            const SizedBox(height: AppSpacing.space16),

            ..._lineItems.asMap().entries.map((entry) {
              return _buildLineItem(entry.key, entry.value);
            }),

            const SizedBox(height: AppSpacing.space16),

            AppButton(
              label: 'Add Line Item',
              icon: Icons.add,
              variant: AppButtonVariant.secondary,
              onPressed: () {
                setState(() {
                  _lineItems.add(_LineItem(
                    description: '',
                    quantity: 1,
                    unitPrice: 0.0,
                  ));
                });
              },
            ),

            const SizedBox(height: AppSpacing.space32),

            // Total
            Container(
              padding: const EdgeInsets.all(AppSpacing.space16),
              decoration: BoxDecoration(
                color: theme.colorScheme.primaryContainer,
                borderRadius: BorderRadius.circular(AppBorderRadius.medium),
              ),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(
                    'Total Amount',
                    style: theme.textTheme.titleLarge,
                  ),
                  Text(
                    '\$${_calculateTotal().toStringAsFixed(2)}',
                    style: theme.textTheme.titleLarge?.copyWith(
                      color: theme.colorScheme.primary,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ],
              ),
            ),

            const SizedBox(height: AppSpacing.space24),

            // Notes
            TextFormField(
              controller: _notesController,
              decoration: const InputDecoration(
                labelText: 'Notes',
                hintText: 'Additional notes...',
                border: OutlineInputBorder(),
              ),
              maxLines: 3,
            ),

            const SizedBox(height: AppSpacing.space32),

            // Actions
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                AppButton(
                  label: 'Cancel',
                  variant: AppButtonVariant.tertiary,
                  onPressed: () => Navigator.of(context).pop(),
                ),
                const SizedBox(width: AppSpacing.space12),
                AppButton(
                  label: 'Save Invoice',
                  icon: Icons.save,
                  onPressed: _handleSave,
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildSectionHeader(ThemeData theme, String title) {
    return Row(
      children: [
        Container(
          width: 4,
          height: 24,
          decoration: BoxDecoration(
            color: theme.colorScheme.primary,
            borderRadius: BorderRadius.circular(2),
          ),
        ),
        const SizedBox(width: AppSpacing.space12),
        Text(
          title,
          style: theme.textTheme.titleLarge,
        ),
      ],
    );
  }

  Widget _buildLineItem(int index, _LineItem item) {
    return Card(
      margin: const EdgeInsets.only(bottom: AppSpacing.space12),
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.space12),
        child: Row(
          children: [
            Expanded(
              flex: 3,
              child: TextFormField(
                initialValue: item.description,
                decoration: const InputDecoration(
                  labelText: 'Description',
                  border: OutlineInputBorder(),
                  isDense: true,
                ),
                onChanged: (value) => item.description = value,
              ),
            ),
            const SizedBox(width: AppSpacing.space12),
            Expanded(
              child: TextFormField(
                initialValue: item.quantity.toString(),
                decoration: const InputDecoration(
                  labelText: 'Qty',
                  border: OutlineInputBorder(),
                  isDense: true,
                ),
                keyboardType: TextInputType.number,
                onChanged: (value) => item.quantity = int.tryParse(value) ?? 1,
              ),
            ),
            const SizedBox(width: AppSpacing.space12),
            Expanded(
              child: TextFormField(
                initialValue: item.unitPrice.toString(),
                decoration: const InputDecoration(
                  labelText: 'Price',
                  border: OutlineInputBorder(),
                  isDense: true,
                ),
                keyboardType: TextInputType.number,
                onChanged: (value) =>
                    item.unitPrice = double.tryParse(value) ?? 0.0,
              ),
            ),
            const SizedBox(width: AppSpacing.space12),
            IconButton(
              icon: const Icon(Icons.delete_outline),
              onPressed: _lineItems.length > 1
                  ? () => setState(() => _lineItems.removeAt(index))
                  : null,
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _selectDueDate(BuildContext context) async {
    final picked = await showDatePicker(
      context: context,
      initialDate: _dueDate,
      firstDate: DateTime.now(),
      lastDate: DateTime.now().add(const Duration(days: 365)),
    );
    if (picked != null) {
      setState(() => _dueDate = picked);
    }
  }

  double _calculateTotal() {
    return _lineItems.fold(
      0.0,
      (sum, item) => sum + (item.quantity * item.unitPrice),
    );
  }

  void _handleSave() {
    if (_formKey.currentState?.validate() ?? false) {
      final data = {
        'invoiceNumber': _invoiceNumberController.text,
        'customer': _customerController.text,
        'dueDate': _dueDate.toIso8601String(),
        'status': _status,
        'lineItems': _lineItems
            .map((item) => {
                  'description': item.description,
                  'quantity': item.quantity,
                  'unitPrice': item.unitPrice,
                  'total': item.quantity * item.unitPrice,
                })
            .toList(),
        'total': _calculateTotal(),
        'notes': _notesController.text,
      };

      widget.onSaved(data);
    }
  }
}

class _LineItem {
  _LineItem({
    required this.description,
    required this.quantity,
    required this.unitPrice,
  });

  String description;
  int quantity;
  double unitPrice;
}
