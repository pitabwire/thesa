/// Theme provider with persistence and branding support.
library;

import 'package:flutter/material.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../branding/branding_config.dart';
import 'theme_builder.dart';

part 'theme_provider.g.dart';

/// Storage key for theme mode preference
const String _themeModeKey = 'theme_mode';

/// Branding configuration provider
///
/// In a real app, this would load from:
/// - Local config file (assets/branding.json)
/// - Remote branding endpoint
/// - Environment variables
@Riverpod(keepAlive: true)
Future<BrandingConfig> brandingConfig(BrandingConfigRef ref) async {
  // TODO: Load branding from config file or endpoint
  // For now, return default branding
  return BrandingConfig.defaultBranding;
}

/// Theme mode provider
///
/// Persists user's theme preference (light/dark/system).
@riverpod
class ThemeModeNotifier extends _$ThemeModeNotifier {
  @override
  Future<ThemeMode> build() async {
    // Load persisted theme mode
    final prefs = await SharedPreferences.getInstance();
    final themeModeIndex = prefs.getInt(_themeModeKey);

    if (themeModeIndex != null && themeModeIndex < ThemeMode.values.length) {
      return ThemeMode.values[themeModeIndex];
    }

    // Default to system theme
    return ThemeMode.system;
  }

  /// Set theme mode and persist preference
  Future<void> setThemeMode(ThemeMode mode) async {
    state = AsyncValue.data(mode);

    // Persist preference
    final prefs = await SharedPreferences.getInstance();
    await prefs.setInt(_themeModeKey, mode.index);
  }

  /// Toggle between light and dark (skip system)
  Future<void> toggleTheme() async {
    final current = await future;

    switch (current) {
      case ThemeMode.light:
        await setThemeMode(ThemeMode.dark);
        break;
      case ThemeMode.dark:
        await setThemeMode(ThemeMode.light);
        break;
      case ThemeMode.system:
        // If system, switch to opposite of current system brightness
        await setThemeMode(ThemeMode.light);
        break;
    }
  }
}

/// Light theme provider
@Riverpod(keepAlive: true)
Future<ThemeData> lightTheme(LightThemeRef ref) async {
  final branding = await ref.watch(brandingConfigProvider.future);
  return ThemeBuilder.buildLightTheme(branding: branding);
}

/// Dark theme provider
@Riverpod(keepAlive: true)
Future<ThemeData> darkTheme(DarkThemeRef ref) async {
  final branding = await ref.watch(brandingConfigProvider.future);
  return ThemeBuilder.buildDarkTheme(branding: branding);
}

/// Current brightness provider
///
/// Determines actual brightness based on theme mode and system preference.
@riverpod
Brightness currentBrightness(CurrentBrightnessRef ref, BuildContext context) {
  final themeModeAsync = ref.watch(themeModeNotifierProvider);

  return themeModeAsync.when(
    data: (themeMode) {
      switch (themeMode) {
        case ThemeMode.light:
          return Brightness.light;
        case ThemeMode.dark:
          return Brightness.dark;
        case ThemeMode.system:
          return MediaQuery.platformBrightnessOf(context);
      }
    },
    loading: () => MediaQuery.platformBrightnessOf(context),
    error: (_, __) => Brightness.light,
  );
}
