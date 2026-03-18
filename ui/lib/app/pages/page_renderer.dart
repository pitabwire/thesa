/// Dynamic page renderer.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/core.dart';
import '../../design/design.dart';
import '../../plugins/plugin_provider.dart';
import '../../state/state.dart';
import '../../telemetry/telemetry.dart';
import '../../ui_engine/ui_engine.dart';
import '../../widgets/shared/shared.dart';

/// Renders a dynamic page from BFF page descriptor
class PageRenderer extends ConsumerStatefulWidget {
  const PageRenderer({
    required this.pageId,
    this.params = const {},
    super.key,
  });

  /// Page ID to load
  final String pageId;

  /// Route parameters (e.g., {id: "123"})
  final Map<String, String> params;

  @override
  ConsumerState<PageRenderer> createState() => _PageRendererState();
}

class _PageRendererState extends ConsumerState<PageRenderer> {
  final DateTime _startTime = DateTime.now();

  @override
  void initState() {
    super.initState();
    // Set current page for performance monitoring
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(performanceMonitorProvider).setCurrentPage(widget.pageId);
    });
  }

  @override
  Widget build(BuildContext context) {
    final pageAsync = ref.watch(pageProvider(widget.pageId));
    final pluginRegistry = ref.watch(pluginRegistryProvider);
    final telemetryService = ref.watch(telemetryServiceProvider);

    return pageAsync.when(
      data: (page) {
        // Record page render telemetry
        WidgetsBinding.instance.addPostFrameCallback((_) {
          _recordPageRenderTelemetry(page, telemetryService);
        });

        // Check plugin registry first
        if (pluginRegistry.hasPagePlugin(widget.pageId)) {
          final builder = pluginRegistry.getPagePlugin(widget.pageId)!;
          return builder(context, page, ref);
        }

        // Fall back to generic renderer
        return _PageContent(
          page: page,
          params: widget.params,
        );
      },
      loading: () => const Scaffold(
        body: Center(
          child: AppLoadingIndicator(
            message: 'Loading page...',
          ),
        ),
      ),
      error: (error, stack) => Scaffold(
        body: Center(
          child: AppErrorWidget(
            error: error,
            onRetry: () => ref.invalidate(pageProvider(widget.pageId)),
          ),
        ),
      ),
    );
  }

  void _recordPageRenderTelemetry(
    PageDescriptor page,
    TelemetryService telemetryService,
  ) {
    final renderTime = DateTime.now().difference(_startTime).inMilliseconds;
    final componentCount =
        page.components.where((c) => c.permission.allowed).length;

    telemetryService.record(
      TelemetryEvent.pageRender(
        pageId: widget.pageId,
        renderTimeMs: renderTime,
        componentCount: componentCount,
        fromCache: true, // TODO: Track actual cache status from provider
        cacheAgeMs: null, // TODO: Get from cache metadata
        stale: false, // TODO: Get from cache metadata
        timestamp: DateTime.now(),
      ),
    );
  }
}

/// Page content renderer
class _PageContent extends StatelessWidget {
  const _PageContent({
    required this.page,
    required this.params,
  });

  final PageDescriptor page;
  final Map<String, String> params;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(page.title),
        actions: [
          // TODO: Render page actions from page.actions
          // For now, just show a placeholder
          if (page.actions.isNotEmpty)
            IconButton(
              icon: const Icon(Icons.more_vert),
              onPressed: () {
                // TODO: Show actions menu
              },
            ),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(AppSpacing.space16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            // Render breadcrumbs if available
            if (page.breadcrumbs != null && page.breadcrumbs!.isNotEmpty) ...[
              _Breadcrumbs(items: page.breadcrumbs!),
              const SizedBox(height: AppSpacing.space16),
            ],

            // Render components using ComponentRenderer
            ...page.components
                .where((component) => component.permission.allowed)
                .map(
                  (component) => Padding(
                    padding: const EdgeInsets.only(bottom: AppSpacing.space16),
                    child: ComponentRenderer(
                      component: component,
                      params: params,
                    ),
                  ),
                ),

            // Empty state if no visible components
            if (page.components
                .where((component) => component.permission.allowed)
                .isEmpty)
              const AppEmptyState(
                message: 'This page has no content',
                icon: Icons.description,
              ),
          ],
        ),
      ),
    );
  }
}

/// Breadcrumb navigation
class _Breadcrumbs extends StatelessWidget {
  const _Breadcrumbs({required this.items});

  final List<BreadcrumbItem> items;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        for (var i = 0; i < items.length; i++) ...[
          if (i > 0) ...[
            const SizedBox(width: AppSpacing.space8),
            const Icon(Icons.chevron_right, size: 16),
            const SizedBox(width: AppSpacing.space8),
          ],
          if (items[i].path != null)
            InkWell(
              onTap: () {
                // TODO: Navigate to breadcrumb path
              },
              child: Text(
                items[i].label,
                style: AppTypography.bodySmall.copyWith(
                  color: Theme.of(context).colorScheme.primary,
                ),
              ),
            )
          else
            Text(
              items[i].label,
              style: AppTypography.bodySmall,
            ),
        ],
      ],
    );
  }
}

