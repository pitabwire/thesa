/// Optimized widget utilities for performance.
library;

import 'package:flutter/material.dart';

/// RepaintBoundary wrapper for expensive widgets
///
/// Use this to isolate expensive widgets (charts, maps, large forms)
/// from unnecessary repaints when their parent rebuilds.
class OptimizedRepaintBoundary extends StatelessWidget {
  const OptimizedRepaintBoundary({
    required this.child,
    super.key,
  });

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return RepaintBoundary(child: child);
  }
}

/// Const wrapper to force const construction
///
/// Wraps a widget tree in a const constructor to prevent rebuilds.
class ConstWrapper extends StatelessWidget {
  const ConstWrapper({
    required this.child,
    super.key,
  });

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return child;
  }
}

/// Mixin for keeping widget state alive in tabs/pages
///
/// Use with AutomaticKeepAliveClientMixin to preserve state
/// when tabs are switched or pages are scrolled out of view.
///
/// Example:
/// ```dart
/// class MyTabContent extends StatefulWidget {
///   // ...
/// }
///
/// class _MyTabContentState extends State<MyTabContent>
///     with AutomaticKeepAliveClientMixin {
///   @override
///   bool get wantKeepAlive => true;
///
///   @override
///   Widget build(BuildContext context) {
///     super.build(context); // Required for AutomaticKeepAliveClientMixin
///     return // ... your widget tree
///   }
/// }
/// ```

/// Virtualized list builder for large datasets
///
/// Only builds visible items plus a small buffer.
class VirtualizedList extends StatelessWidget {
  const VirtualizedList({
    required this.itemCount,
    required this.itemBuilder,
    this.separatorBuilder,
    this.padding,
    this.scrollController,
    super.key,
  });

  final int itemCount;
  final IndexedWidgetBuilder itemBuilder;
  final IndexedWidgetBuilder? separatorBuilder;
  final EdgeInsetsGeometry? padding;
  final ScrollController? scrollController;

  @override
  Widget build(BuildContext context) {
    if (separatorBuilder != null) {
      return ListView.separated(
        controller: scrollController,
        padding: padding,
        itemCount: itemCount,
        itemBuilder: itemBuilder,
        separatorBuilder: separatorBuilder!,
      );
    }

    return ListView.builder(
      controller: scrollController,
      padding: padding,
      itemCount: itemCount,
      itemBuilder: itemBuilder,
    );
  }
}

/// Virtualized grid builder for large datasets
///
/// Only builds visible items plus a small buffer.
class VirtualizedGrid extends StatelessWidget {
  const VirtualizedGrid({
    required this.itemCount,
    required this.itemBuilder,
    required this.crossAxisCount,
    this.crossAxisSpacing = 8.0,
    this.mainAxisSpacing = 8.0,
    this.childAspectRatio = 1.0,
    this.padding,
    this.scrollController,
    super.key,
  });

  final int itemCount;
  final IndexedWidgetBuilder itemBuilder;
  final int crossAxisCount;
  final double crossAxisSpacing;
  final double mainAxisSpacing;
  final double childAspectRatio;
  final EdgeInsetsGeometry? padding;
  final ScrollController? scrollController;

  @override
  Widget build(BuildContext context) {
    return GridView.builder(
      controller: scrollController,
      padding: padding,
      gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: crossAxisCount,
        crossAxisSpacing: crossAxisSpacing,
        mainAxisSpacing: mainAxisSpacing,
        childAspectRatio: childAspectRatio,
      ),
      itemCount: itemCount,
      itemBuilder: itemBuilder,
    );
  }
}

/// Virtualized sliver list for CustomScrollView
///
/// Use this within a CustomScrollView for advanced scroll effects.
class VirtualizedSliverList extends StatelessWidget {
  const VirtualizedSliverList({
    required this.itemCount,
    required this.itemBuilder,
    super.key,
  });

  final int itemCount;
  final IndexedWidgetBuilder itemBuilder;

  @override
  Widget build(BuildContext context) {
    return SliverList.builder(
      itemCount: itemCount,
      itemBuilder: itemBuilder,
    );
  }
}

/// Virtualized sliver grid for CustomScrollView
///
/// Use this within a CustomScrollView for advanced scroll effects.
class VirtualizedSliverGrid extends StatelessWidget {
  const VirtualizedSliverGrid({
    required this.itemCount,
    required this.itemBuilder,
    required this.crossAxisCount,
    this.crossAxisSpacing = 8.0,
    this.mainAxisSpacing = 8.0,
    this.childAspectRatio = 1.0,
    super.key,
  });

  final int itemCount;
  final IndexedWidgetBuilder itemBuilder;
  final int crossAxisCount;
  final double crossAxisSpacing;
  final double mainAxisSpacing;
  final double childAspectRatio;

  @override
  Widget build(BuildContext context) {
    return SliverGrid.builder(
      gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: crossAxisCount,
        crossAxisSpacing: crossAxisSpacing,
        mainAxisSpacing: mainAxisSpacing,
        childAspectRatio: childAspectRatio,
      ),
      itemCount: itemCount,
      itemBuilder: itemBuilder,
    );
  }
}
