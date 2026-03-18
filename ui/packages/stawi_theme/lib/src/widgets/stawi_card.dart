import 'package:flutter/material.dart';
import '../theme.dart';

/// A styled card with optional hover effect, matching BoundaryML's card pattern.
///
/// ```dart
/// StawiCard(
///   child: Text('Content'),
///   onTap: () {},
/// )
/// ```
class StawiCard extends StatefulWidget {
  const StawiCard({
    super.key,
    required this.child,
    this.onTap,
    this.padding,
    this.hoverEffect = true,
  });

  final Widget child;
  final VoidCallback? onTap;
  final EdgeInsetsGeometry? padding;
  final bool hoverEffect;

  @override
  State<StawiCard> createState() => _StawiCardState();
}

class _StawiCardState extends State<StawiCard> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final tokens = context.stawiColors;
    final spacing = context.stawiSpacing;

    return MouseRegion(
      onEnter: widget.hoverEffect ? (_) => setState(() => _hovered = true) : null,
      onExit: widget.hoverEffect ? (_) => setState(() => _hovered = false) : null,
      child: GestureDetector(
        onTap: widget.onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 200),
          padding: widget.padding ?? EdgeInsets.all(spacing.lg),
          decoration: BoxDecoration(
            color: _hovered
                ? tokens.muted.withValues(alpha: 0.5)
                : tokens.card,
            borderRadius: spacing.borderRadiusLg,
            border: Border.all(
              color: _hovered ? tokens.mutedForeground.withValues(alpha: 0.3) : tokens.border,
            ),
          ),
          child: widget.child,
        ),
      ),
    );
  }
}

/// A bento-grid card that uses pseudo-border lines (like BoundaryML's bento).
///
/// Best used inside a [Wrap] or [GridView] to create a 2-column bento layout.
class StawiBentoCard extends StatefulWidget {
  const StawiBentoCard({
    super.key,
    required this.title,
    required this.description,
    required this.visual,
    this.onTap,
    this.minHeight = 400,
  });

  final String title;
  final String description;
  final Widget visual;
  final VoidCallback? onTap;
  final double minHeight;

  @override
  State<StawiBentoCard> createState() => _StawiBentoCardState();
}

class _StawiBentoCardState extends State<StawiBentoCard> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final tokens = context.stawiColors;

    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 200),
          constraints: BoxConstraints(minHeight: widget.minHeight),
          decoration: BoxDecoration(
            color: _hovered ? tokens.muted.withValues(alpha: 0.3) : Colors.transparent,
          ),
          child: Column(
            children: [
              Expanded(
                child: Center(child: Padding(
                  padding: const EdgeInsets.all(24),
                  child: widget.visual,
                )),
              ),
              Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      widget.title,
                      style: TextStyle(
                        fontSize: 18,
                        fontWeight: FontWeight.w600,
                        letterSpacing: -0.2,
                        color: tokens.foreground,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      widget.description,
                      style: TextStyle(
                        fontSize: 15,
                        height: 1.6,
                        color: tokens.mutedForeground,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
