/// Search bar widget.
library;

import 'package:flutter/material.dart';

import '../../design/design.dart';

/// Search bar widget
class AppSearchBar extends StatefulWidget {
  const AppSearchBar({
    required this.onSearch,
    this.onClear,
    this.hintText = 'Search...',
    this.initialValue,
    this.debounceMs = 300,
    super.key,
  });

  final void Function(String query) onSearch;
  final VoidCallback? onClear;
  final String hintText;
  final String? initialValue;
  final int debounceMs;

  @override
  State<AppSearchBar> createState() => _AppSearchBarState();
}

class _AppSearchBarState extends State<AppSearchBar> {
  late TextEditingController _controller;
  DateTime? _lastInputTime;

  @override
  void initState() {
    super.initState();
    _controller = TextEditingController(text: widget.initialValue);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: _controller,
      decoration: InputDecoration(
        hintText: widget.hintText,
        prefixIcon: const Icon(Icons.search),
        suffixIcon: _controller.text.isNotEmpty
            ? IconButton(
                icon: const Icon(Icons.clear),
                onPressed: _handleClear,
              )
            : null,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
        ),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: AppSpacing.space16,
          vertical: AppSpacing.space12,
        ),
      ),
      onChanged: _handleChange,
      onSubmitted: widget.onSearch,
    );
  }

  void _handleChange(String value) {
    setState(() {});

    _lastInputTime = DateTime.now();

    // Debounce search
    Future.delayed(Duration(milliseconds: widget.debounceMs), () {
      final now = DateTime.now();
      if (_lastInputTime != null &&
          now.difference(_lastInputTime!).inMilliseconds >=
              widget.debounceMs) {
        widget.onSearch(value);
      }
    });
  }

  void _handleClear() {
    _controller.clear();
    setState(() {});
    widget.onClear?.call();
    widget.onSearch('');
  }
}
