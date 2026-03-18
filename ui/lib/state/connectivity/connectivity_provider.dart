/// Connectivity provider for monitoring online/offline state.
///
/// Always alive - never disposed.
library;

import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'connectivity_provider.g.dart';

/// Stream of connectivity status
@Riverpod(keepAlive: true)
Stream<bool> connectivity(ConnectivityRef ref) async* {
  final connectivity = Connectivity();

  // Emit initial connectivity state
  final initial = await connectivity.checkConnectivity();
  yield initial.any((result) => result != ConnectivityResult.none);

  // Listen for connectivity changes
  await for (final results in connectivity.onConnectivityChanged) {
    yield results.any((result) => result != ConnectivityResult.none);
  }
}

/// Current connectivity status (sync)
@Riverpod(keepAlive: true)
Future<bool> isOnline(IsOnlineRef ref) async {
  final connectivityStream = ref.watch(connectivityProvider);
  return connectivityStream.when(
    data: (isOnline) => isOnline,
    loading: () => true, // Assume online while checking
    error: (_, __) => false,
  );
}
