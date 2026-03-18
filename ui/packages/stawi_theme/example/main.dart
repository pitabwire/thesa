import 'package:flutter/material.dart';
import 'package:stawi_theme/stawi_theme.dart';

void main() => runApp(const StawiThemeDemo());

class StawiThemeDemo extends StatefulWidget {
  const StawiThemeDemo({super.key});

  @override
  State<StawiThemeDemo> createState() => _StawiThemeDemoState();
}

class _StawiThemeDemoState extends State<StawiThemeDemo> {
  ThemeMode _mode = ThemeMode.dark;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Stawi Theme Demo',
      debugShowCheckedModeBanner: false,
      theme: StawiTheme.light(),
      darkTheme: StawiTheme.dark(),
      themeMode: _mode,
      home: DemoHome(
        themeMode: _mode,
        onToggle: () => setState(() {
          _mode =
              _mode == ThemeMode.dark ? ThemeMode.light : ThemeMode.dark;
        }),
      ),
    );
  }
}

class DemoHome extends StatelessWidget {
  const DemoHome({
    super.key,
    required this.themeMode,
    required this.onToggle,
  });

  final ThemeMode themeMode;
  final VoidCallback onToggle;

  @override
  Widget build(BuildContext context) {
    final tokens = context.stawiColors;
    final spacing = context.stawiSpacing;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Stawi Theme'),
        actions: [
          IconButton(
            icon: Icon(
              themeMode == ThemeMode.dark
                  ? Icons.light_mode
                  : Icons.dark_mode,
            ),
            onPressed: onToggle,
          ),
        ],
      ),
      body: SingleChildScrollView(
        padding: EdgeInsets.all(spacing.lg),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 800),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // ── Badge ──────────────────────────
                const Center(
                  child: StawiBadge(
                    label: 'Now in Public Beta',
                    dotColor: StawiColors.emerald,
                  ),
                ),
                SizedBox(height: spacing.xxl),

                // ── Section Header ─────────────────
                const StawiSectionHeader(
                  title: 'Component Showcase',
                  subtitle: 'All themed components in one place',
                ),
                SizedBox(height: spacing.xl),

                // ── Buttons ────────────────────────
                _label(context, 'Buttons'),
                SizedBox(height: spacing.sm),
                Wrap(
                  spacing: spacing.sm,
                  runSpacing: spacing.sm,
                  children: [
                    ElevatedButton(
                      onPressed: () {},
                      child: const Text('Primary'),
                    ),
                    ElevatedButton.icon(
                      onPressed: () {},
                      icon: const Icon(Icons.rocket_launch, size: 16),
                      label: const Text('Get Started'),
                    ),
                    OutlinedButton(
                      onPressed: () {},
                      child: const Text('Outline'),
                    ),
                    TextButton(
                      onPressed: () {},
                      child: const Text('Ghost'),
                    ),
                    ElevatedButton(
                      onPressed: () {},
                      style: StawiButtonStyles.destructive(
                        Theme.of(context).brightness,
                      ),
                      child: const Text('Destructive'),
                    ),
                    ElevatedButton(
                      onPressed: () {},
                      style: StawiButtonStyles.secondary(
                        Theme.of(context).brightness,
                      ),
                      child: const Text('Secondary'),
                    ),
                  ],
                ),
                SizedBox(height: spacing.xl),

                // ── Inputs ─────────────────────────
                _label(context, 'Inputs'),
                SizedBox(height: spacing.sm),
                const TextField(
                  decoration: InputDecoration(
                    labelText: 'Cluster name',
                    hintText: 'prod-us-east-1',
                  ),
                ),
                SizedBox(height: spacing.md),
                TextField(
                  decoration: StawiInputStyles.searchDecoration(
                    brightness: Theme.of(context).brightness,
                    hintText: 'Search services...',
                  ),
                ),
                SizedBox(height: spacing.xl),

                // ── Cards ──────────────────────────
                _label(context, 'Cards'),
                SizedBox(height: spacing.sm),
                StawiCard(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Service Health',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      const SizedBox(height: 8),
                      Text(
                        'All 47 services running normally.',
                        style: TextStyle(color: tokens.mutedForeground),
                      ),
                    ],
                  ),
                ),
                SizedBox(height: spacing.xl),

                // ── Terminal ───────────────────────
                _label(context, 'Terminal'),
                SizedBox(height: spacing.sm),
                const StawiTerminal(
                  lines: [
                    TerminalLine.command('stawi init --cluster production'),
                    TerminalLine.comment(
                        '# Connecting to Kubernetes cluster...'),
                    TerminalLine.output(
                        '✓ Cluster connected: prod-us-east-1'),
                    TerminalLine.output('✓ 47 services discovered'),
                    TerminalLine.output('✓ Monitoring agents deployed'),
                    TerminalLine.command('stawi watch --auto-build'),
                    TerminalLine.output(
                        '◈ Watching for feature requests...'),
                  ],
                ),
                SizedBox(height: spacing.xl),

                // ── Metrics ────────────────────────
                _label(context, 'Metrics'),
                SizedBox(height: spacing.sm),
                GridView.count(
                  crossAxisCount: 2,
                  shrinkWrap: true,
                  physics: const NeverScrollableScrollPhysics(),
                  crossAxisSpacing: spacing.sm,
                  mainAxisSpacing: spacing.sm,
                  childAspectRatio: 1.6,
                  children: const [
                    StawiMetricCard(
                      label: 'Uptime',
                      value: '99.98%',
                      valueColor: StawiColors.emerald,
                      progress: 0.9998,
                      progressColor: StawiColors.emerald,
                    ),
                    StawiMetricCard(
                      label: 'P99 Latency',
                      value: '42ms',
                      valueColor: StawiColors.sky,
                      progress: 0.15,
                      progressColor: StawiColors.sky,
                    ),
                    StawiMetricCard(
                      label: 'Error Rate',
                      value: '0.02%',
                      valueColor: StawiColors.amber,
                      progress: 0.02,
                      progressColor: StawiColors.amber,
                    ),
                    StawiMetricCard(
                      label: 'Throughput',
                      value: '14.2K/s',
                      valueColor: StawiColors.indigo,
                      progress: 0.71,
                      progressColor: StawiColors.indigo,
                    ),
                  ],
                ),
                SizedBox(height: spacing.xl),

                // ── Status Dots ────────────────────
                _label(context, 'Status Indicators'),
                SizedBox(height: spacing.sm),
                Wrap(
                  spacing: spacing.lg,
                  runSpacing: spacing.sm,
                  children: const [
                    StawiStatusDot.running(label: 'Running'),
                    StawiStatusDot.pending(label: 'Pending'),
                    StawiStatusDot.error(label: 'Error'),
                    StawiStatusDot(
                      color: StawiColors.sky,
                      label: 'Syncing',
                    ),
                  ],
                ),
                SizedBox(height: spacing.xl),

                // ── Chips ──────────────────────────
                _label(context, 'Chips'),
                SizedBox(height: spacing.sm),
                Wrap(
                  spacing: spacing.sm,
                  children: [
                    const Chip(label: Text('kubernetes')),
                    const Chip(label: Text('monitoring')),
                    ChoiceChip(
                      label: const Text('selected'),
                      selected: true,
                      onSelected: (_) {},
                    ),
                    const Chip(
                      avatar: Icon(Icons.close, size: 14),
                      label: Text('removable'),
                    ),
                  ],
                ),
                SizedBox(height: spacing.xl),

                // ── Toggles ───────────────────────
                _label(context, 'Toggles'),
                SizedBox(height: spacing.sm),
                Row(
                  children: [
                    Switch(value: true, onChanged: (_) {}),
                    SizedBox(width: spacing.md),
                    Switch(value: false, onChanged: (_) {}),
                    SizedBox(width: spacing.md),
                    Checkbox(value: true, onChanged: (_) {}),
                    SizedBox(width: spacing.md),
                    Checkbox(value: false, onChanged: (_) {}),
                    SizedBox(width: spacing.md),
                    RadioGroup<bool>(
                      groupValue: true,
                      onChanged: (_) {},
                      child: const Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Radio<bool>(value: true),
                          Radio<bool>(value: false),
                        ],
                      ),
                    ),
                  ],
                ),
                SizedBox(height: spacing.xl),

                // ── Progress ──────────────────────
                _label(context, 'Progress'),
                SizedBox(height: spacing.sm),
                const LinearProgressIndicator(value: 0.65),
                SizedBox(height: spacing.md),
                const Row(
                  children: [
                    SizedBox(
                      width: 24,
                      height: 24,
                      child: CircularProgressIndicator(strokeWidth: 3),
                    ),
                    SizedBox(width: 16),
                    Text('Loading...'),
                  ],
                ),
                SizedBox(height: spacing.xl),

                // ── Colors Palette ─────────────────
                _label(context, 'Color Palette'),
                SizedBox(height: spacing.sm),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    _swatch('Background', tokens.background),
                    _swatch('Foreground', tokens.foreground),
                    _swatch('Card', tokens.card),
                    _swatch('Muted', tokens.muted),
                    _swatch('Muted FG', tokens.mutedForeground),
                    _swatch('Border', tokens.border),
                    _swatch('Secondary', tokens.secondary),
                    _swatch('Destructive', tokens.destructive),
                  ],
                ),
                SizedBox(height: spacing.sm),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: StawiColors.chartColors
                      .asMap()
                      .entries
                      .map(
                        (e) => _swatch('Chart ${e.key + 1}', e.value),
                      )
                      .toList(),
                ),
                SizedBox(height: spacing.xxxl),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _label(BuildContext context, String text) {
    return Text(
      text,
      style: StawiTypography.overline(
        color: context.stawiColors.mutedForeground,
      ),
    );
  }

  Widget _swatch(String name, Color color) {
    return Column(
      children: [
        Container(
          width: 56,
          height: 56,
          decoration: BoxDecoration(
            color: color,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
              color: Colors.white.withValues(alpha: 0.1),
            ),
          ),
        ),
        const SizedBox(height: 4),
        Text(
          name,
          style: const TextStyle(fontSize: 10),
          textAlign: TextAlign.center,
        ),
      ],
    );
  }
}
