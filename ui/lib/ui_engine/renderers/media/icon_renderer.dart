/// Icon renderer.
library;

import 'package:flutter/material.dart';

import '../../../core/core.dart';

/// Renders icon component
class IconRenderer extends StatelessWidget {
  const IconRenderer({
    required this.component,
    super.key,
  });

  final ComponentDescriptor component;

  @override
  Widget build(BuildContext context) {
    final iconName = component.config['icon'] as String? ?? 'circle';
    final size = (component.config['size'] as num?)?.toDouble() ?? 24.0;
    final colorString = component.config['color'] as String?;

    Color? color;
    if (colorString != null) {
      color = _parseColor(colorString, context);
    }

    return Icon(
      _parseIcon(iconName),
      size: size,
      color: color,
    );
  }

  IconData _parseIcon(String icon) {
    // Extended icon mapping
    switch (icon.toLowerCase()) {
      // Navigation
      case 'home':
        return Icons.home;
      case 'menu':
        return Icons.menu;
      case 'back':
        return Icons.arrow_back;
      case 'forward':
        return Icons.arrow_forward;
      case 'close':
        return Icons.close;

      // Actions
      case 'add':
      case 'plus':
        return Icons.add;
      case 'edit':
        return Icons.edit;
      case 'delete':
        return Icons.delete;
      case 'save':
        return Icons.save;
      case 'cancel':
        return Icons.cancel;
      case 'search':
        return Icons.search;
      case 'filter':
        return Icons.filter_list;
      case 'refresh':
        return Icons.refresh;
      case 'download':
        return Icons.download;
      case 'upload':
        return Icons.upload;

      // Content
      case 'file':
      case 'document':
        return Icons.description;
      case 'folder':
        return Icons.folder;
      case 'image':
        return Icons.image;
      case 'video':
        return Icons.videocam;
      case 'audio':
        return Icons.audiotrack;

      // UI Elements
      case 'settings':
        return Icons.settings;
      case 'user':
      case 'person':
        return Icons.person;
      case 'users':
      case 'people':
        return Icons.people;
      case 'calendar':
        return Icons.calendar_today;
      case 'clock':
      case 'time':
        return Icons.access_time;
      case 'notification':
      case 'bell':
        return Icons.notifications;
      case 'mail':
      case 'email':
        return Icons.mail;
      case 'phone':
        return Icons.phone;
      case 'location':
      case 'map':
        return Icons.location_on;

      // Status
      case 'check':
      case 'checkmark':
        return Icons.check;
      case 'error':
      case 'alert':
        return Icons.error;
      case 'warning':
        return Icons.warning;
      case 'info':
        return Icons.info;
      case 'help':
      case 'question':
        return Icons.help;

      // Data
      case 'table':
        return Icons.table_chart;
      case 'list':
        return Icons.list;
      case 'grid':
        return Icons.grid_view;
      case 'chart':
      case 'graph':
        return Icons.show_chart;

      // Arrows
      case 'up':
        return Icons.arrow_upward;
      case 'down':
        return Icons.arrow_downward;
      case 'left':
        return Icons.arrow_back;
      case 'right':
        return Icons.arrow_forward;

      default:
        return Icons.circle;
    }
  }

  Color? _parseColor(String colorString, BuildContext context) {
    final theme = Theme.of(context);

    switch (colorString.toLowerCase()) {
      case 'primary':
        return theme.colorScheme.primary;
      case 'secondary':
        return theme.colorScheme.secondary;
      case 'error':
        return theme.colorScheme.error;
      case 'success':
        return Colors.green;
      case 'warning':
        return Colors.orange;
      case 'info':
        return Colors.blue;
      default:
        // Try to parse as hex color
        if (colorString.startsWith('#')) {
          try {
            return Color(
              int.parse(colorString.substring(1), radix: 16) + 0xFF000000,
            );
          } catch (e) {
            return null;
          }
        }
        return null;
    }
  }
}
