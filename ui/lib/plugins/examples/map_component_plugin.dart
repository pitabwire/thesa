/// Example component plugin demonstrating custom component rendering.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/core.dart';
import '../../design/design.dart';

/// Example map component plugin
///
/// Demonstrates custom component rendering with config access
Widget buildMapComponent(
  ComponentDescriptor descriptor,
  WidgetRef ref,
) {
  return MapComponentPlugin(descriptor: descriptor);
}

/// Custom map component implementation
///
/// In production, this would integrate with a real map library
/// (e.g., flutter_map, google_maps_flutter)
class MapComponentPlugin extends StatelessWidget {
  const MapComponentPlugin({
    required this.descriptor,
    super.key,
  });

  final ComponentDescriptor descriptor;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final config = descriptor.config;

    // Extract configuration
    final center = config['center'] as Map<String, dynamic>?;
    final zoom = (config['zoom'] as num?)?.toDouble() ?? 12.0;
    final markers = config['markers'] as List? ?? [];

    final lat = (center?['lat'] as num?)?.toDouble() ?? 0.0;
    final lng = (center?['lng'] as num?)?.toDouble() ?? 0.0;

    return Card(
      child: Container(
        height: 400,
        padding: const EdgeInsets.all(AppSpacing.space16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Header
            Row(
              children: [
                Icon(
                  Icons.map,
                  color: theme.colorScheme.primary,
                ),
                const SizedBox(width: AppSpacing.space8),
                Text(
                  descriptor.id,
                  style: theme.textTheme.titleMedium,
                ),
              ],
            ),
            const SizedBox(height: AppSpacing.space16),

            // Map placeholder (would be actual map in production)
            Expanded(
              child: Container(
                decoration: BoxDecoration(
                  color: theme.colorScheme.surfaceVariant,
                  borderRadius: BorderRadius.circular(AppBorderRadius.medium),
                  border: Border.all(
                    color: theme.colorScheme.outline,
                  ),
                ),
                child: Stack(
                  children: [
                    // Map grid pattern
                    CustomPaint(
                      size: Size.infinite,
                      painter: _MapGridPainter(
                        color: theme.colorScheme.outline.withOpacity(0.3),
                      ),
                    ),

                    // Center marker
                    Center(
                      child: Icon(
                        Icons.location_on,
                        color: theme.colorScheme.error,
                        size: 48,
                      ),
                    ),

                    // Info overlay
                    Positioned(
                      top: AppSpacing.space12,
                      left: AppSpacing.space12,
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: AppSpacing.space12,
                          vertical: AppSpacing.space8,
                        ),
                        decoration: BoxDecoration(
                          color: theme.colorScheme.surface,
                          borderRadius: BorderRadius.circular(AppBorderRadius.small),
                          boxShadow: [
                            BoxShadow(
                              color: Colors.black.withOpacity(0.1),
                              blurRadius: 4,
                            ),
                          ],
                        ),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Text(
                              'Center: $lat, $lng',
                              style: theme.textTheme.bodySmall,
                            ),
                            Text(
                              'Zoom: $zoom',
                              style: theme.textTheme.bodySmall,
                            ),
                            Text(
                              'Markers: ${markers.length}',
                              style: theme.textTheme.bodySmall,
                            ),
                          ],
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),

            // Markers list
            if (markers.isNotEmpty) ...[
              const SizedBox(height: AppSpacing.space16),
              Text(
                'Locations (${markers.length})',
                style: theme.textTheme.titleSmall,
              ),
              const SizedBox(height: AppSpacing.space8),
              Wrap(
                spacing: AppSpacing.space8,
                children: markers.take(5).map((marker) {
                  final label = marker['label'] as String? ?? 'Unknown';
                  return Chip(
                    avatar: const Icon(Icons.place, size: 16),
                    label: Text(label),
                  );
                }).toList(),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// Custom painter for map grid pattern
class _MapGridPainter extends CustomPainter {
  _MapGridPainter({required this.color});

  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..strokeWidth = 1;

    const gridSize = 40.0;

    // Draw vertical lines
    for (var x = 0.0; x < size.width; x += gridSize) {
      canvas.drawLine(
        Offset(x, 0),
        Offset(x, size.height),
        paint,
      );
    }

    // Draw horizontal lines
    for (var y = 0.0; y < size.height; y += gridSize) {
      canvas.drawLine(
        Offset(0, y),
        Offset(size.width, y),
        paint,
      );
    }
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}
