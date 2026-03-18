/// Sidebar navigation menu.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/core.dart';
import '../../design/design.dart';
import '../../state/state.dart';
import '../../widgets/shared/shared.dart';

/// Sidebar navigation menu
class AppSidebar extends ConsumerWidget {
  const AppSidebar({super.key});

  static const double width = 256.0;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final navigationAsync = ref.watch(navigationProvider);
    final theme = Theme.of(context);

    return Container(
      width: width,
      color: theme.colorScheme.surface,
      child: Column(
        children: [
          // App branding/logo
          _buildHeader(context),

          const Divider(height: 1),

          // Navigation items
          Expanded(
            child: navigationAsync.when(
              data: (navigation) => _buildNavigationList(
                context,
                ref,
                navigation.items
                    .where((item) => item.permission.allowed)
                    .toList(),
              ),
              loading: () => const Center(
                child: AppLoadingIndicator(size: AppLoadingSize.small),
              ),
              error: (error, stack) => Center(
                child: AppErrorWidget(
                  error: error,
                  compact: true,
                  onRetry: () => ref.invalidate(navigationProvider),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  /// Build sidebar header with logo/title
  Widget _buildHeader(BuildContext context) {
    return Container(
      height: 64,
      padding: const EdgeInsets.all(AppSpacing.space16),
      child: Row(
        children: [
          // Logo placeholder
          Container(
            width: 32,
            height: 32,
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.primary,
              borderRadius: BorderRadius.circular(AppBorderRadius.small),
            ),
            child: Icon(
              Icons.dashboard,
              color: Theme.of(context).colorScheme.onPrimary,
              size: 20,
            ),
          ),
          const SizedBox(width: AppSpacing.space12),
          // App title
          Text(
            'Thesa UI',
            style: Theme.of(context).textTheme.titleLarge,
          ),
        ],
      ),
    );
  }

  /// Build navigation list
  Widget _buildNavigationList(
    BuildContext context,
    WidgetRef ref,
    List<NavigationItem> items,
  ) {
    // Separate top and bottom items
    final topItems =
        items.where((item) => !item.bottomPosition).toList();
    final bottomItems =
        items.where((item) => item.bottomPosition).toList();

    return Column(
      children: [
        // Top items (main navigation)
        Expanded(
          child: ListView(
            padding: const EdgeInsets.all(AppSpacing.space8),
            children: topItems
                .map((item) => _buildNavigationItem(context, ref, item))
                .toList(),
          ),
        ),

        // Bottom items (settings, logout, etc.)
        if (bottomItems.isNotEmpty) ...[
          const Divider(height: 1),
          Padding(
            padding: const EdgeInsets.all(AppSpacing.space8),
            child: Column(
              children: bottomItems
                  .map((item) => _buildNavigationItem(context, ref, item))
                  .toList(),
            ),
          ),
        ],
      ],
    );
  }

  /// Build single navigation item
  Widget _buildNavigationItem(
    BuildContext context,
    WidgetRef ref,
    NavigationItem item,
  ) {
    final currentPath = GoRouterState.of(context).uri.path;
    final isSelected = currentPath == item.path ||
        (item.children?.any((child) => currentPath == child.path) ?? false);

    // If item has children, show expandable tile
    if (item.children != null && item.children!.isNotEmpty) {
      return _buildExpandableItem(context, ref, item, isSelected);
    }

    // Otherwise show simple tile
    return _buildSimpleItem(context, item, isSelected);
  }

  /// Build simple navigation item (no children)
  Widget _buildSimpleItem(
    BuildContext context,
    NavigationItem item,
    bool isSelected,
  ) {
    final theme = Theme.of(context);

    return ListTile(
      leading: item.icon != null
          ? Icon(
              _getIconData(item.icon!),
              size: AppSizing.iconMedium,
            )
          : null,
      title: Row(
        children: [
          Expanded(
            child: Text(
              item.label,
              style: AppTypography.bodyMedium,
            ),
          ),
          if (item.badge != null) ...[
            const SizedBox(width: AppSpacing.space8),
            AppBadge(
              label: item.badge!,
              small: true,
              color: _getBadgeColor(item.badgeColor, theme),
            ),
          ],
        ],
      ),
      selected: isSelected,
      selectedTileColor: theme.colorScheme.primary.withOpacity(0.12),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(AppBorderRadius.medium),
      ),
      onTap: () {
        if (item.path != null) {
          context.go(item.path!);
        }
      },
    );
  }

  /// Build expandable navigation item (with children)
  Widget _buildExpandableItem(
    BuildContext context,
    WidgetRef ref,
    NavigationItem item,
    bool isSelected,
  ) {
    return ExpansionTile(
      leading: item.icon != null
          ? Icon(
              _getIconData(item.icon!),
              size: AppSizing.iconMedium,
            )
          : null,
      title: Text(
        item.label,
        style: AppTypography.bodyMedium,
      ),
      initiallyExpanded: !item.defaultCollapsed,
      children: item.children!
          .where((child) => child.permission.allowed)
          .map((child) => Padding(
                padding: const EdgeInsets.only(left: AppSpacing.space32),
                child: _buildNavigationItem(context, ref, child),
              ))
          .toList(),
    );
  }

  /// Get Material icon from string name
  IconData _getIconData(String iconName) {
    // Map common icon names to Material icons
    return switch (iconName.toLowerCase()) {
      'dashboard' => Icons.dashboard,
      'shopping_cart' => Icons.shopping_cart,
      'inventory' => Icons.inventory,
      'settings' => Icons.settings,
      'person' => Icons.person,
      'logout' => Icons.logout,
      'home' => Icons.home,
      'analytics' => Icons.analytics,
      _ => Icons.circle_outlined,
    };
  }

  /// Get badge color
  Color _getBadgeColor(String? colorName, ThemeData theme) {
    if (colorName == null) return theme.colorScheme.primary;

    final isDark = theme.brightness == Brightness.dark;
    return AppColors.getStatusColor(colorName, isDark: isDark);
  }
}
