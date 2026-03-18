/// Skeleton loader widgets for loading states.
library;

import 'package:flutter/material.dart';
import 'package:shimmer/shimmer.dart';

import '../../design/design.dart';

/// Base skeleton loader
///
/// Shows a shimmering placeholder while content loads.
/// Prevents layout shift by matching the size of the final content.
class AppSkeletonLoader extends StatelessWidget {
  const AppSkeletonLoader({
    required this.width,
    required this.height,
    this.borderRadius = AppBorderRadius.medium,
    super.key,
  });

  final double width;
  final double height;
  final double borderRadius;

  @override
  Widget build(BuildContext context) {
    return Shimmer.fromColors(
      baseColor: Theme.of(context).colorScheme.surfaceVariant,
      highlightColor: Theme.of(context).colorScheme.surface,
      child: Container(
        width: width,
        height: height,
        decoration: BoxDecoration(
          color: Theme.of(context).colorScheme.surfaceVariant,
          borderRadius: BorderRadius.circular(borderRadius),
        ),
      ),
    );
  }
}

/// Skeleton loader for text lines
class TextLineSkeleton extends StatelessWidget {
  const TextLineSkeleton({
    this.width = double.infinity,
    this.height = 16.0,
    super.key,
  });

  final double width;
  final double height;

  @override
  Widget build(BuildContext context) {
    return AppSkeletonLoader(
      width: width,
      height: height,
      borderRadius: AppBorderRadius.small,
    );
  }
}

/// Skeleton loader for cards
class CardSkeleton extends StatelessWidget {
  const CardSkeleton({
    this.width = double.infinity,
    this.height = 200.0,
    super.key,
  });

  final double width;
  final double height;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.space16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const TextLineSkeleton(width: 120, height: 20),
            const SizedBox(height: AppSpacing.space12),
            Expanded(
              child: AppSkeletonLoader(
                width: width,
                height: height - 60,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Skeleton loader for list items
class ListItemSkeleton extends StatelessWidget {
  const ListItemSkeleton({super.key});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(
        horizontal: AppSpacing.space16,
        vertical: AppSpacing.space8,
      ),
      child: Row(
        children: [
          const AppSkeletonLoader(
            width: 40,
            height: 40,
            borderRadius: AppBorderRadius.full,
          ),
          const SizedBox(width: AppSpacing.space12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const TextLineSkeleton(height: 16),
                const SizedBox(height: AppSpacing.space4),
                TextLineSkeleton(
                  width: MediaQuery.of(context).size.width * 0.6,
                  height: 14,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

/// Skeleton loader for table rows
class TableRowSkeleton extends StatelessWidget {
  const TableRowSkeleton({
    this.columnCount = 4,
    super.key,
  });

  final int columnCount;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(
        horizontal: AppSpacing.space16,
        vertical: AppSpacing.space12,
      ),
      child: Row(
        children: List.generate(
          columnCount,
          (index) => Expanded(
            child: Padding(
              padding: const EdgeInsets.symmetric(
                horizontal: AppSpacing.space8,
              ),
              child: const TextLineSkeleton(height: 16),
            ),
          ),
        ),
      ),
    );
  }
}

/// Skeleton loader for navigation sidebar
class SidebarSkeleton extends StatelessWidget {
  const SidebarSkeleton({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        const SizedBox(height: AppSpacing.space16),
        ...List.generate(
          6,
          (index) => const Padding(
            padding: EdgeInsets.symmetric(
              horizontal: AppSpacing.space16,
              vertical: AppSpacing.space8,
            ),
            child: TextLineSkeleton(height: 40),
          ),
        ),
      ],
    );
  }
}

/// Skeleton loader for page content
class PageContentSkeleton extends StatelessWidget {
  const PageContentSkeleton({super.key});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(AppSpacing.space16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const TextLineSkeleton(width: 200, height: 28),
          const SizedBox(height: AppSpacing.space24),
          ...List.generate(
            3,
            (index) => Padding(
              padding: const EdgeInsets.only(bottom: AppSpacing.space16),
              child: const CardSkeleton(height: 150),
            ),
          ),
        ],
      ),
    );
  }
}
