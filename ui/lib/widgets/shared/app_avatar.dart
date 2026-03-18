/// Avatar widget for user profiles.
library;

import 'package:flutter/material.dart';

import '../../design/design.dart';

/// Avatar size options
enum AppAvatarSize {
  small,
  medium,
  large,
}

/// Avatar component for user profiles
class AppAvatar extends StatelessWidget {
  const AppAvatar({
    this.imageUrl,
    this.initials,
    this.size = AppAvatarSize.medium,
    this.backgroundColor,
    super.key,
  });

  /// Optional image URL
  final String? imageUrl;

  /// Optional initials (shown if no image)
  final String? initials;

  /// Avatar size
  final AppAvatarSize size;

  /// Background color (for initials avatar)
  final Color? backgroundColor;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    final diameter = switch (size) {
      AppAvatarSize.small => AppSizing.avatarSmall,
      AppAvatarSize.medium => AppSizing.avatarMedium,
      AppAvatarSize.large => AppSizing.avatarLarge,
    };

    final fontSize = switch (size) {
      AppAvatarSize.small => 10.0,
      AppAvatarSize.medium => 16.0,
      AppAvatarSize.large => 24.0,
    };

    // If image URL provided, show image avatar
    if (imageUrl != null && imageUrl!.isNotEmpty) {
      return CircleAvatar(
        radius: diameter / 2,
        backgroundImage: NetworkImage(imageUrl!),
        backgroundColor: backgroundColor ?? theme.colorScheme.primary,
      );
    }

    // Otherwise show initials avatar
    return CircleAvatar(
      radius: diameter / 2,
      backgroundColor: backgroundColor ?? theme.colorScheme.primary,
      child: Text(
        initials ?? '?',
        style: TextStyle(
          fontSize: fontSize,
          fontWeight: FontWeight.w600,
          color: theme.colorScheme.onPrimary,
        ),
      ),
    );
  }

  /// Generate initials from name
  static String getInitials(String name) {
    if (name.isEmpty) return '?';

    final parts = name.trim().split(RegExp(r'\s+'));
    if (parts.length == 1) {
      return parts[0][0].toUpperCase();
    }

    // Take first letter of first and last name
    return '${parts.first[0]}${parts.last[0]}'.toUpperCase();
  }
}

/// Avatar group (overlapping avatars)
class AppAvatarGroup extends StatelessWidget {
  const AppAvatarGroup({
    required this.avatars,
    this.maxVisible = 3,
    this.size = AppAvatarSize.medium,
    super.key,
  });

  /// List of avatar data
  final List<_AvatarData> avatars;

  /// Maximum visible avatars
  final int maxVisible;

  /// Avatar size
  final AppAvatarSize size;

  @override
  Widget build(BuildContext context) {
    final visibleCount = avatars.length > maxVisible ? maxVisible : avatars.length;
    final remainingCount = avatars.length - visibleCount;

    final diameter = switch (size) {
      AppAvatarSize.small => AppSizing.avatarSmall,
      AppAvatarSize.medium => AppSizing.avatarMedium,
      AppAvatarSize.large => AppSizing.avatarLarge,
    };

    return SizedBox(
      height: diameter,
      child: Stack(
        children: [
          // Visible avatars
          for (var i = 0; i < visibleCount; i++)
            Positioned(
              left: i * (diameter * 0.7),
              child: Container(
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  border: Border.all(
                    color: Theme.of(context).scaffoldBackgroundColor,
                    width: 2,
                  ),
                ),
                child: AppAvatar(
                  imageUrl: avatars[i].imageUrl,
                  initials: avatars[i].initials,
                  size: size,
                ),
              ),
            ),
          // "+N" indicator if more avatars exist
          if (remainingCount > 0)
            Positioned(
              left: visibleCount * (diameter * 0.7),
              child: Container(
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  border: Border.all(
                    color: Theme.of(context).scaffoldBackgroundColor,
                    width: 2,
                  ),
                ),
                child: CircleAvatar(
                  radius: diameter / 2,
                  backgroundColor:
                      Theme.of(context).colorScheme.primary.withOpacity(0.2),
                  child: Text(
                    '+$remainingCount',
                    style: TextStyle(
                      fontSize: diameter * 0.3,
                      fontWeight: FontWeight.w600,
                      color: Theme.of(context).colorScheme.primary,
                    ),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

/// Avatar data model
class _AvatarData {
  const _AvatarData({
    this.imageUrl,
    this.initials,
  });

  final String? imageUrl;
  final String? initials;
}
