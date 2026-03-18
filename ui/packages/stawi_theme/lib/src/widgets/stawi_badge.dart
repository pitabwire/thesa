import 'package:flutter/material.dart';
import '../theme.dart';

/// A pill-shaped badge, styled like BoundaryML's hero badge.
///
/// ```dart
/// StawiBadge(
///   label: 'Now in Public Beta',
///   dotColor: Colors.green,   // optional pulsing dot
/// )
/// ```
class StawiBadge extends StatefulWidget {
  const StawiBadge({
    super.key,
    required this.label,
    this.dotColor,
    this.icon,
    this.onTap,
  });

  /// Text to display.
  final String label;

  /// If non-null, shows a pulsing dot of this color before the label.
  final Color? dotColor;

  /// Optional leading icon (after the dot, before the label).
  final Widget? icon;

  /// Optional tap handler.
  final VoidCallback? onTap;

  @override
  State<StawiBadge> createState() => _StawiBadgeState();
}

class _StawiBadgeState extends State<StawiBadge>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<double> _opacity;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 2),
    )..repeat(reverse: true);
    _opacity = Tween<double>(begin: 1.0, end: 0.4).animate(
      CurvedAnimation(parent: _controller, curve: Curves.easeInOut),
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final tokens = context.stawiColors;

    final child = Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      decoration: BoxDecoration(
        color: tokens.muted,
        borderRadius: BorderRadius.circular(9999),
        border: Border.all(color: tokens.border),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (widget.dotColor != null) ...[
            AnimatedBuilder(
              animation: _opacity,
              builder: (context, _) => Container(
                width: 6,
                height: 6,
                decoration: BoxDecoration(
                  color: widget.dotColor!.withValues(alpha: _opacity.value),
                  shape: BoxShape.circle,
                ),
              ),
            ),
            const SizedBox(width: 8),
          ],
          if (widget.icon != null) ...[
            widget.icon!,
            const SizedBox(width: 6),
          ],
          Text(
            widget.label,
            style: TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.w500,
              color: tokens.mutedForeground,
            ),
          ),
        ],
      ),
    );

    if (widget.onTap != null) {
      return GestureDetector(onTap: widget.onTap, child: child);
    }
    return child;
  }
}
