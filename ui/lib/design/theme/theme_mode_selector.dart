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
    final themeModeAsync = ref.watch(themeModeProvider);

    return themeModeAsync.when<Widget>(
      data: (themeMode) => Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'Theme',
            style: Theme.of(context).textTheme.titleMedium,
          ),
          const SizedBox(height: 16),
          RadioGroup<ThemeMode>(
            groupValue: themeMode,
            onChanged: (mode) {
              if (mode != null) {
                ref.read(themeModeProvider.notifier).setThemeMode(mode);
              }
            },
            child: const Column(
              children: [
                RadioListTile<ThemeMode>(
                  title: Text('Light'),
                  subtitle: Text('Always use light theme'),
                  value: ThemeMode.light,
                ),
                RadioListTile<ThemeMode>(
                  title: Text('Dark'),
                  subtitle: Text('Always use dark theme'),
                  value: ThemeMode.dark,
                ),
                RadioListTile<ThemeMode>(
                  title: Text('System'),
                  subtitle: Text('Follow system preference'),
                  value: ThemeMode.system,
                ),
              ],
            ),
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
    final themeModeAsync = ref.watch(themeModeProvider);

    return themeModeAsync.when<Widget>(
      data: (themeMode) {
        final isDark = Theme.of(context).brightness == Brightness.dark;

        return IconButton(
          icon: Icon(isDark ? Icons.light_mode : Icons.dark_mode),
          tooltip: isDark ? 'Switch to light mode' : 'Switch to dark mode',
          onPressed: () {
            ref.read(themeModeProvider.notifier).toggleTheme();
          },
        );
      },
      loading: () => const IconButton(
        icon: Icon(Icons.brightness_auto),
        onPressed: null,
      ),
      error: (_, _) => const SizedBox.shrink(),
    );
  }
}
