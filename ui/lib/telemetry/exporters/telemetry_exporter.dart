/// Telemetry exporter interface.
library;

import '../models/telemetry_event.dart';

/// Base interface for telemetry exporters
abstract class TelemetryExporter {
  /// Export a batch of events
  Future<void> export(List<TelemetryEvent> events);
}
