/// Riverpod providers for performance monitoring and optimization.
library;

import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../networking/background_refresh_coordinator.dart';
import '../../telemetry/telemetry_provider.dart';
import 'performance_budgets.dart';

part 'performance_providers.g.dart';

/// Performance budget monitor provider
@Riverpod(keepAlive: true)
PerformanceBudgetMonitor performanceBudgetMonitor(
  Ref ref,
) {
  final telemetryService = ref.watch(telemetryServiceProvider);

  return PerformanceBudgetMonitor(
    telemetryService: telemetryService,
  );
}

/// Background refresh coordinator provider
@Riverpod(keepAlive: true)
BackgroundRefreshCoordinator backgroundRefreshCoordinator(
  Ref ref,
) {
  final coordinator = BackgroundRefreshCoordinator(ref: ref);

  ref.onDispose(() {
    coordinator.dispose();
  });

  return coordinator;
}
