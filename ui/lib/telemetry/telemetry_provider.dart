/// Riverpod providers for telemetry service.
library;

import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'exporters/otel_exporter.dart';
import 'performance_monitor.dart';
import 'telemetry_service.dart';

part 'telemetry_provider.g.dart';

/// Telemetry configuration provider
@Riverpod(keepAlive: true)
TelemetryConfig telemetryConfig(TelemetryConfigRef ref) {
  // Load from environment or config
  const endpoint = String.fromEnvironment(
    'OTEL_ENDPOINT',
    defaultValue: 'http://localhost:4318/v1/traces',
  );

  const enabled = bool.fromEnvironment(
    'TELEMETRY_ENABLED',
    defaultValue: true,
  );

  return TelemetryConfig(
    flushIntervalSeconds: 30,
    maxBufferSize: 100,
    maxBufferSizeBeforeDrop: 10000,
    enabled: enabled,
    exportEndpoint: enabled ? endpoint : null,
  );
}

/// Telemetry service provider
@Riverpod(keepAlive: true)
TelemetryService telemetryService(TelemetryServiceRef ref) {
  final config = ref.watch(telemetryConfigProvider);

  final exporter = OtelExporter(
    endpoint: config.exportEndpoint ?? '',
    serviceName: 'thesa-ui',
  );

  final service = TelemetryService(
    config: config,
    exporter: exporter,
  );

  ref.onDispose(() {
    service.dispose();
  });

  return service;
}

/// Performance monitor provider
@Riverpod(keepAlive: true)
PerformanceMonitor performanceMonitor(PerformanceMonitorRef ref) {
  final telemetryService = ref.watch(telemetryServiceProvider);

  final monitor = PerformanceMonitor(
    telemetryService: telemetryService,
    jankThresholdMs: 32.0, // 2 frames at 60fps
    recordAllFrames: false, // Only record janks
  );

  ref.onDispose(() {
    monitor.dispose();
  });

  return monitor;
}
