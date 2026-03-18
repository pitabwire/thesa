/// Performance budget constants and monitoring.
library;

import 'package:logger/logger.dart';

import '../../telemetry/models/telemetry_event.dart';
import '../../telemetry/telemetry_service.dart';

/// Performance budget thresholds
///
/// These define the maximum acceptable values for key performance metrics.
/// Exceeding these budgets triggers warnings in telemetry.
class PerformanceBudgets {
  PerformanceBudgets._();

  /// Page render should complete in <300ms (P95)
  static const int pageRenderMs = 300;

  /// Cache-first render should complete in <100ms
  static const int cacheFirstRenderMs = 100;

  /// API requests should complete in <500ms (P95)
  static const int apiResponseMs = 500;

  /// Frame render should complete in <16ms for 60fps
  static const double frameRenderMs = 16.0;

  /// Jank rate should be <1% of frames
  static const double jankRatePercent = 1.0;

  /// Form submission should complete in <1000ms
  static const int formSubmissionMs = 1000;

  /// Action execution should complete in <2000ms
  static const int actionExecutionMs = 2000;
}

/// Performance monitor for budget enforcement
///
/// Tracks performance metrics and logs warnings when budgets are exceeded.
class PerformanceBudgetMonitor {
  PerformanceBudgetMonitor({
    required this.telemetryService,
  });

  final TelemetryService telemetryService;
  final Logger _logger = Logger();

  /// Check and log if page render exceeded budget
  void checkPageRenderBudget({
    required String pageId,
    required int renderTimeMs,
    required bool fromCache,
  }) {
    final budget =
        fromCache ? PerformanceBudgets.cacheFirstRenderMs : PerformanceBudgets.pageRenderMs;

    if (renderTimeMs > budget) {
      final exceededBy = renderTimeMs - budget;
      _logger.warning(
        'Page render budget exceeded for $pageId: '
        '${renderTimeMs}ms (budget: ${budget}ms, exceeded by: ${exceededBy}ms)',
      );

      // This is already recorded by PageRenderer, just log the warning
    }
  }

  /// Check and log if API request exceeded budget
  void checkApiRequestBudget({
    required String endpoint,
    required int durationMs,
  }) {
    if (durationMs > PerformanceBudgets.apiResponseMs) {
      final exceededBy = durationMs - PerformanceBudgets.apiResponseMs;
      _logger.warning(
        'API request budget exceeded for $endpoint: '
        '${durationMs}ms (budget: ${PerformanceBudgets.apiResponseMs}ms, '
        'exceeded by: ${exceededBy}ms)',
      );
    }
  }

  /// Check and log if frame render exceeded budget (jank)
  void checkFrameRenderBudget({
    required String pageId,
    required double frameTimeMs,
  }) {
    if (frameTimeMs > PerformanceBudgets.frameRenderMs) {
      final exceededBy = frameTimeMs - PerformanceBudgets.frameRenderMs;
      _logger.warning(
        'Frame render budget exceeded on $pageId: '
        '${frameTimeMs.toStringAsFixed(2)}ms '
        '(budget: ${PerformanceBudgets.frameRenderMs}ms, '
        'exceeded by: ${exceededBy.toStringAsFixed(2)}ms)',
      );

      // Frame timing events are recorded by PerformanceMonitor
    }
  }

  /// Check and log if form submission exceeded budget
  void checkFormSubmissionBudget({
    required String schemaId,
    required int durationMs,
  }) {
    if (durationMs > PerformanceBudgets.formSubmissionMs) {
      final exceededBy = durationMs - PerformanceBudgets.formSubmissionMs;
      _logger.warning(
        'Form submission budget exceeded for $schemaId: '
        '${durationMs}ms (budget: ${PerformanceBudgets.formSubmissionMs}ms, '
        'exceeded by: ${exceededBy}ms)',
      );
    }
  }

  /// Check and log if action execution exceeded budget
  void checkActionExecutionBudget({
    required String actionId,
    required int durationMs,
  }) {
    if (durationMs > PerformanceBudgets.actionExecutionMs) {
      final exceededBy = durationMs - PerformanceBudgets.actionExecutionMs;
      _logger.warning(
        'Action execution budget exceeded for $actionId: '
        '${durationMs}ms (budget: ${PerformanceBudgets.actionExecutionMs}ms, '
        'exceeded by: ${exceededBy}ms)',
      );
    }
  }
}
