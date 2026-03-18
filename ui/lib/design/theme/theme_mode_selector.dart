/// Theme mode selector widget for user settings.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'theme_provider.dart';

/// Theme mode selector widget
///
/// Allows users to choose between light, dark, and system theme modes.
class ThemeModeSelector extends ConsumerWidget {
  const ThemeModeSelector({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final themeModeAsync = ref.watch(themeModeNotifierProvider);

    return themeModeAsync.when(
      data: (themeMode) => Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'Theme',
            style: Theme.of(context).textTheme.titleMedium,
          ),
          const SizedBox(height: 16),
          RadioListTile<ThemeMode>(
            title: const Text('Light'),
            subtitle: const Text('Always use light theme'),
            value: ThemeMode.light,
            groupValue: themeMode,
            onChanged: (mode) {
              if (mode != null) {
                ref.read(themeModeNotifierProvider.notifier).setThemeMode(mode);
              }
            },
          ),
          RadioListTile<ThemeMode>(
            title: const Text('Dark'),
            subtitle: const Text('Always use dark theme'),
            value: ThemeMode.dark,
            groupValue: themeMode,
            onChanged: (mode) {
              if (mode != null) {
                ref.read(themeModeNotifierProvider.notifier).setThemeMode(mode);
              }
            },
          ),
          RadioListTile<ThemeMode>(
            title: const Text('System'),
            subtitle: const Text('Follow system preference'),
            value: ThemeMode.system,
            groupValue: themeMode,
            onChanged: (mode) {
              if (mode != null) {
                ref.read(themeModeNotifierProvider.notifier).setThemeMode(mode);
              }
            },
          ),
        ],
      ),
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (error, stack) => Text('Error loading theme: $error'),
    );
  }
}

/// Theme toggle button
///
/// Simple button to toggle between light and dark modes.
class ThemeToggleButton extends ConsumerWidget {
  const ThemeToggleButton({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final themeModeAsync = ref.watch(themeModeNotifierProvider);

    return themeModeAsync.when(
      data: (themeMode) {
        final isDark = Theme.of(context).brightness == Brightness.dark;

        return IconButton(
          icon: Icon(isDark ? Icons.light_mode : Icons.dark_mode),
          tooltip: isDark ? 'Switch to light mode' : 'Switch to dark mode',
          onPressed: () {
            ref.read(themeModeNotifierProvider.notifier).toggleTheme();
          },
        );
      },
      loading: () => const IconButton(
        icon: Icon(Icons.brightness_auto),
        onPressed: null,
      ),
      error: (_, __) => const SizedBox.shrink(),
    );
  }
}
