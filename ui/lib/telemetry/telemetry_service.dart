/// Core telemetry service for event collection and buffering.
library;

import 'dart:async';

import 'package:logger/logger.dart';

import 'exporters/telemetry_exporter.dart';
import 'models/telemetry_event.dart';

/// Configuration for telemetry service
class TelemetryConfig {
  const TelemetryConfig({
    this.flushIntervalSeconds = 30,
    this.maxBufferSize = 100,
    this.maxBufferSizeBeforeDrop = 10000,
    this.enabled = true,
    this.exportEndpoint,
  });

  /// How often to flush the buffer (seconds)
  final int flushIntervalSeconds;

  /// Max events before auto-flush
  final int maxBufferSize;

  /// Max events to keep in buffer before dropping old ones
  final int maxBufferSizeBeforeDrop;

  /// Whether telemetry is enabled
  final bool enabled;

  /// OpenTelemetry export endpoint (if null, events are buffered but not exported)
  final String? exportEndpoint;
}

/// Telemetry service for collecting and exporting events
///
/// Buffers events in memory and flushes them periodically or when buffer is full.
/// Integrates with OpenTelemetry for standardized observability.
class TelemetryService {
  TelemetryService({
    required this.config,
    required this.exporter,
  }) {
    if (config.enabled) {
      _startFlushTimer();
    }
  }

  final TelemetryConfig config;
  final TelemetryExporter exporter;
  final Logger _logger = Logger();

  final List<TelemetryEvent> _buffer = [];
  Timer? _flushTimer;
  bool _isFlushing = false;

  /// Record a telemetry event
  void record(TelemetryEvent event) {
    if (!config.enabled) {
      return;
    }

    _buffer.add(event);

    // Drop oldest events if buffer is too large
    if (_buffer.length > config.maxBufferSizeBeforeDrop) {
      final dropCount = _buffer.length - config.maxBufferSizeBeforeDrop;
      _buffer.removeRange(0, dropCount);
      _logger.warning(
        'Telemetry buffer overflow: dropped $dropCount events',
      );
    }

    // Auto-flush if buffer reaches max size
    if (_buffer.length >= config.maxBufferSize) {
      _flush();
    }
  }

  /// Manually flush the buffer
  Future<void> flush() async {
    await _flush();
  }

  /// Start the periodic flush timer
  void _startFlushTimer() {
    _flushTimer?.cancel();
    _flushTimer = Timer.periodic(
      Duration(seconds: config.flushIntervalSeconds),
      (_) => _flush(),
    );
  }

  /// Flush buffered events to exporter
  Future<void> _flush() async {
    if (_isFlushing || _buffer.isEmpty) {
      return;
    }

    _isFlushing = true;

    try {
      // Take a snapshot of current buffer
      final events = List<TelemetryEvent>.from(_buffer);
      _buffer.clear();

      // Export events
      await exporter.export(events);

      _logger.info('Flushed ${events.length} telemetry events');
    } catch (error, stack) {
      _logger.error(
        'Failed to flush telemetry events',
        error: error,
        stackTrace: stack,
      );
      // Don't re-add events to buffer to avoid infinite growth
    } finally {
      _isFlushing = false;
    }
  }

  /// Dispose resources
  void dispose() {
    _flushTimer?.cancel();
    _flushTimer = null;
  }
}
