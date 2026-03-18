import 'package:flutter/material.dart';
import '../colors.dart';

/// A small colored status indicator dot with optional label.
///
/// ```dart
/// StawiStatusDot.running()
/// StawiStatusDot.pending()
/// StawiStatusDot.error()
/// StawiStatusDot(color: Colors.blue, label: 'Syncing')
/// ```
class StawiStatusDot extends StatelessWidget {
  const StawiStatusDot({
    super.key,
    required this.color,
    this.label,
    this.size = 6,
    this.animate = false,
  });

  /// Running / healthy status (green).
  const StawiStatusDot.running({super.key, this.label})
      : color = StawiColors.emerald,
        size = 6,
        animate = false;

  /// Pending / warning status (amber).
  const StawiStatusDot.pending({super.key, this.label})
      : color = StawiColors.amber,
        size = 6,
        animate = true;

  /// Error / failing status (red).
  const StawiStatusDot.error({super.key, this.label})
      : color = StawiColors.rose,
        size = 6,
        animate = true;

  final Color color;
  final String? label;
  final double size;
  final bool animate;

  @override
  Widget build(BuildContext context) {
    final dot = Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        color: color,
        shape: BoxShape.circle,
      ),
    );

    final dotWidget = animate ? _PulsingDot(color: color, size: size) : dot;

    if (label == null) return dotWidget;

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        dotWidget,
        const SizedBox(width: 6),
        Text(
          label!,
          style: TextStyle(
            fontSize: 12,
            fontWeight: FontWeight.w500,
            color: color,
          ),
        ),
      ],
    );
  }
}

class _PulsingDot extends StatefulWidget {
  const _PulsingDot({required this.color, required this.size});
  final Color color;
  final double size;

  @override
  State<_PulsingDot> createState() => _PulsingDotState();
}

class _PulsingDotState extends State<_PulsingDot>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 2),
    )..repeat(reverse: true);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _controller,
      builder: (context, _) => Container(
        width: widget.size,
        height: widget.size,
        decoration: BoxDecoration(
          color: widget.color.withValues(alpha: 0.5 + _controller.value * 0.5),
          shape: BoxShape.circle,
          boxShadow: [
            BoxShadow(
              color: widget.color.withValues(alpha: _controller.value * 0.4),
              blurRadius: 6,
              spreadRadius: 1,
            ),
          ],
        ),
      ),
    );
  }
}
