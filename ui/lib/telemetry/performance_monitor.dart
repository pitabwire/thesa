/// Performance monitoring for frame timing and jank detection.
library;

import 'dart:ui';

import 'package:flutter/scheduler.dart';
import 'package:logger/logger.dart';

import 'models/telemetry_event.dart';
import 'telemetry_service.dart';

/// Performance monitor for tracking frame timing and jank
///
/// Monitors frame rendering performance and records telemetry events
/// when frames take too long (janks) or when rebuild counts are high.
class PerformanceMonitor {
  PerformanceMonitor({
    required this.telemetryService,
    this.jankThresholdMs = 32.0, // 2 frames at 60fps
    this.recordAllFrames = false,
  }) {
    _startMonitoring();
  }

  final TelemetryService telemetryService;

  /// Threshold for considering a frame as jank (milliseconds)
  final double jankThresholdMs;

  /// Whether to record all frames or only janks
  final bool recordAllFrames;

  final Logger _logger = Logger();

  String? _currentPageId;
  int _widgetBuildCount = 0;
  FrameTiming? _lastFrameTiming;

  /// Set the current page ID for frame attribution
  void setCurrentPage(String pageId) {
    _currentPageId = pageId;
    _widgetBuildCount = 0;
  }

  /// Increment widget build count
  void recordWidgetBuild() {
    _widgetBuildCount++;
  }

  /// Start monitoring frame timing
  void _startMonitoring() {
    // Register frame timing callback
    SchedulerBinding.instance.addTimingsCallback(_onFrameTiming);
  }

  /// Handle frame timing data
  void _onFrameTiming(List<FrameTiming> timings) {
    if (_currentPageId == null) {
      return;
    }

    for (final timing in timings) {
      _processFrameTiming(timing);
    }
  }

  /// Process a single frame timing
  void _processFrameTiming(FrameTiming timing) {
    final buildDuration = timing.buildDuration.inMicroseconds / 1000.0;
    final rasterDuration = timing.rasterDuration.inMicroseconds / 1000.0;
    final totalFrameTime = buildDuration + rasterDuration;

    final isJank = totalFrameTime > jankThresholdMs;

    // Record event if it's a jank or if we're recording all frames
    if (isJank || recordAllFrames) {
      telemetryService.record(
        TelemetryEvent.frameTiming(
          pageId: _currentPageId!,
          frameTimeMs: totalFrameTime,
          isJank: isJank,
          widgetBuildCount: _widgetBuildCount,
          timestamp: DateTime.now(),
        ),
      );

      if (isJank) {
        _logger.warning(
          'Frame jank detected on $_currentPageId: ${totalFrameTime.toStringAsFixed(2)}ms '
          '(build: ${buildDuration.toStringAsFixed(2)}ms, '
          'raster: ${rasterDuration.toStringAsFixed(2)}ms)',
        );
      }
    }

    _lastFrameTiming = timing;
  }

  /// Get current jank rate (percentage of frames that are janky)
  ///
  /// This would require maintaining a sliding window of frame timings.
  /// For now, we just track individual janks via telemetry.
  double getJankRate() {
    // TODO: Implement sliding window tracking if needed
    return 0.0;
  }

  /// Dispose resources
  void dispose() {
    SchedulerBinding.instance.removeTimingsCallback(_onFrameTiming);
  }
}
