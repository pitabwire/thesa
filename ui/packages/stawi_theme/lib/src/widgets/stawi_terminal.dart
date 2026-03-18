import 'package:flutter/material.dart';
import '../colors.dart';
import '../theme.dart';
import '../typography.dart';

/// A macOS-style terminal window.
///
/// ```dart
/// StawiTerminal(
///   lines: [
///     TerminalLine.command('stawi init --cluster prod'),
///     TerminalLine.comment('Connecting to cluster...'),
///     TerminalLine.output('✓ Connected'),
///   ],
/// )
/// ```
class StawiTerminal extends StatelessWidget {
  const StawiTerminal({
    super.key,
    required this.lines,
    this.maxWidth = 480,
  });

  final List<TerminalLine> lines;
  final double maxWidth;

  @override
  Widget build(BuildContext context) {
    final tokens = context.stawiColors;

    return ConstrainedBox(
      constraints: BoxConstraints(maxWidth: maxWidth),
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: tokens.card,
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: tokens.border),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          mainAxisSize: MainAxisSize.min,
          children: [
            // Title bar
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
              decoration: BoxDecoration(
                color: tokens.muted,
                borderRadius:
                    const BorderRadius.vertical(top: Radius.circular(9)),
                border: Border(
                  bottom: BorderSide(color: tokens.border),
                ),
              ),
              child: Row(
                children: [
                  _dot(StawiColors.rose),
                  const SizedBox(width: 6),
                  _dot(StawiColors.amber),
                  const SizedBox(width: 6),
                  _dot(StawiColors.emerald),
                ],
              ),
            ),
            // Body
            Padding(
              padding: const EdgeInsets.symmetric(
                horizontal: 16,
                vertical: 12,
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: lines
                    .map((line) => _buildLine(context, line))
                    .toList(),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _dot(Color color) {
    return Container(
      width: 10,
      height: 10,
      decoration: BoxDecoration(color: color, shape: BoxShape.circle),
    );
  }

  Widget _buildLine(BuildContext context, TerminalLine line) {
    final tokens = context.stawiColors;

    switch (line.type) {
      case TerminalLineType.command:
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 1),
          child: Row(
            children: [
              Text(
                '\$ ',
                style: StawiTypography.mono(
                  color: StawiColors.emerald,
                  fontSize: 13,
                ),
              ),
              Flexible(
                child: Text(
                  line.text,
                  style: StawiTypography.mono(
                    color: tokens.foreground,
                    fontSize: 13,
                  ),
                ),
              ),
            ],
          ),
        );
      case TerminalLineType.comment:
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 1),
          child: Text(
            line.text,
            style: StawiTypography.mono(
              color: tokens.mutedForeground,
              fontSize: 13,
            ),
          ),
        );
      case TerminalLineType.output:
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 1),
          child: Text(
            line.text,
            style: StawiTypography.mono(
              color: StawiColors.emeraldLight,
              fontSize: 13,
            ),
          ),
        );
      case TerminalLineType.error:
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 1),
          child: Text(
            line.text,
            style: StawiTypography.mono(
              color: StawiColors.rose,
              fontSize: 13,
            ),
          ),
        );
    }
  }
}

enum TerminalLineType { command, comment, output, error }

/// A single line in a [StawiTerminal].
class TerminalLine {
  const TerminalLine._(this.text, this.type);

  /// A command line (`$ command`).
  const TerminalLine.command(String text) : this._(text, TerminalLineType.command);

  /// A comment line (dimmed).
  const TerminalLine.comment(String text) : this._(text, TerminalLineType.comment);

  /// An output/success line (green).
  const TerminalLine.output(String text) : this._(text, TerminalLineType.output);

  /// An error line (red).
  const TerminalLine.error(String text) : this._(text, TerminalLineType.error);

  final String text;
  final TerminalLineType type;
}
