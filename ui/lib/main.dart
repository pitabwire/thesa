/// Main entry point for Thesa UI
///
/// This file initializes the application and starts the Flutter framework.
/// It sets up:
/// - Environment configuration from .env files
/// - Drift database for offline-first caching
/// - Flutter Secure Storage for auth token persistence
/// - Riverpod ProviderScope for state management
/// - Telemetry and error reporting
library;

import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:logging/logging.dart';

import 'package:stawi_theme/stawi_theme.dart';

import 'app/routing/router_provider.dart';

/// Logger for main.dart
final _logger = Logger('main');

/// Main entry point
Future<void> main() async {
  // Ensure Flutter bindings are initialized before any async operations
  WidgetsFlutterBinding.ensureInitialized();

  // Set up logging
  _setupLogging();

  _logger.info('Thesa UI starting...');

  // Load environment configuration
  await _loadEnvironment();

  // Run the app in a guarded zone to catch all errors
  await runZonedGuarded<Future<void>>(
    () async {
      _logger.info('Initialization complete. Launching app...');

      // Launch the app with Riverpod
      runApp(const ProviderScope(child: ThesaApp()));
    },
    (error, stack) {
      // Global error handler
      _logger.severe('Uncaught error in app', error, stack);
    },
  );
}

/// Load environment configuration from .env files.
///
/// In release builds, `.env.production` is automatically loaded as an
/// override so the production BFF URL is used without requiring a
/// `--dart-define=ENV=production` flag at build time.
Future<void> _loadEnvironment() async {
  try {
    // Load base .env file
    await dotenv.load();

    // Determine environment: explicit dart-define takes precedence,
    // otherwise release builds default to "production".
    const envName = String.fromEnvironment('ENV');
    final effectiveEnv = envName.isNotEmpty
        ? envName
        : (kReleaseMode ? 'production' : '');

    if (effectiveEnv.isNotEmpty) {
      try {
        await dotenv.load(
          fileName: '.env.$effectiveEnv',
          mergeWith: dotenv.env,
        );
        _logger.info('Loaded .env.$effectiveEnv overrides');
      } catch (_) {
        _logger.info('No .env.$effectiveEnv file found, using defaults');
      }
    }

    _logger.info('Environment loaded (BFF: ${dotenv.get('BFF_BASE_URL', fallback: 'not set')})');
  } catch (e) {
    _logger.warning('Failed to load .env file, using dart-define values', e);
  }
}

/// Sets up logging for the application
void _setupLogging() {
  // Set log level based on build mode
  Logger.root.level = kDebugMode ? Level.ALL : Level.INFO;

  // Configure log output
  Logger.root.onRecord.listen((record) {
    final message =
        '${record.level.name}: ${record.time}: '
        '${record.loggerName}: ${record.message}';

    if (kDebugMode) {
      // ignore: avoid_print
      print(message);

      if (record.error != null) {
        // ignore: avoid_print
        print('Error: ${record.error}');
      }
      if (record.stackTrace != null) {
        // ignore: avoid_print
        print('Stack trace: ${record.stackTrace}');
      }
    }
  });
}

/// Main Thesa UI application widget
class ThesaApp extends ConsumerWidget {
  const ThesaApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);

    return MaterialApp.router(
      title: 'Thesa UI',
      debugShowCheckedModeBanner: false,
      theme: StawiTheme.light(),
      darkTheme: StawiTheme.dark(),
      themeMode: ThemeMode.dark,
      routerConfig: router,
    );
  }
}
