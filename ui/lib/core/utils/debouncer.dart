/// Debouncer utility for delaying function execution.
library;

import 'dart:async';

/// Debounces function calls to avoid excessive invocations
///
/// Useful for search inputs, filter fields, and other high-frequency user input.
/// When called multiple times rapidly, only executes the function once after
/// the specified delay since the last call.
///
/// Example:
/// ```dart
/// final debouncer = Debouncer(delay: Duration(milliseconds: 300));
///
/// // In a text field's onChanged:
/// debouncer.run(() {
///   performSearch(query);
/// });
/// ```
class Debouncer {
  Debouncer({
    this.delay = const Duration(milliseconds: 300),
  });

  /// Delay duration before executing the function
  final Duration delay;

  Timer? _timer;

  /// Run a function after the debounce delay
  ///
  /// Cancels any pending execution and schedules a new one.
  /// If called repeatedly within the delay period, only the last call executes.
  void run(VoidCallback action) {
    _timer?.cancel();
    _timer = Timer(delay, action);
  }

  /// Cancel any pending execution
  void cancel() {
    _timer?.cancel();
    _timer = null;
  }

  /// Dispose resources
  void dispose() {
    cancel();
  }
}

/// Typedef for void callback
typedef VoidCallback = void Function();
