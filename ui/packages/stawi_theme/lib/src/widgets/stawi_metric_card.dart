import 'package:flutter/material.dart';
import '../theme.dart';

/// A metric display card with label, value, and optional progress bar.
///
/// ```dart
/// StawiMetricCard(
///   label: 'Uptime',
///   value: '99.98%',
///   valueColor: StawiColors.emerald,
///   progress: 0.9998,
///   progressColor: StawiColors.emerald,
/// )
/// ```
class StawiMetricCard extends StatelessWidget {
  const StawiMetricCard({
    super.key,
    required this.label,
    required this.value,
    this.valueColor,
    this.progress,
    this.progressColor,
  });

  final String label;
  final String value;
  final Color? valueColor;

  /// If non-null, shows a progress bar (0.0 – 1.0).
  final double? progress;
  final Color? progressColor;

  @override
  Widget build(BuildContext context) {
    final tokens = context.stawiColors;

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: tokens.card,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: tokens.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label.toUpperCase(),
            style: TextStyle(
              fontSize: 11,
              fontWeight: FontWeight.w500,
              color: tokens.mutedForeground,
              letterSpacing: 0.5,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            value,
            style: TextStyle(
              fontSize: 24,
              fontWeight: FontWeight.w700,
              letterSpacing: -0.5,
              color: valueColor ?? tokens.foreground,
            ),
          ),
          if (progress != null) ...[
            const SizedBox(height: 8),
            ClipRRect(
              borderRadius: BorderRadius.circular(2),
              child: LinearProgressIndicator(
                value: progress,
                backgroundColor: tokens.muted,
                valueColor: AlwaysStoppedAnimation(
                  progressColor ?? tokens.secondary,
                ),
                minHeight: 4,
              ),
            ),
          ],
        ],
      ),
    );
  }
}
