/// Action provider for executing BFF actions.
///
/// Handles confirmation dialogs, loading states, success/error responses,
/// and data invalidation.
library;

import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../core/core.dart';
import '../../telemetry/telemetry.dart';
import '../core/dependencies_provider.dart';
import 'action_state.dart';

part 'action_provider.g.dart';

final _logger = Logger('ActionProvider');

/// Action provider (family - one instance per action ID)
@riverpod
class Action extends _$Action {
  @override
  ActionState build(String actionId) {
    return const ActionState.idle();
  }

  /// Request action execution with optional confirmation
  Future<void> requestExecution({
    required ActionDescriptor descriptor,
    Map<String, dynamic> payload = const {},
  }) async {
    _logger.info('Requesting execution for action: $actionId');

    // If action requires confirmation, transition to confirming state
    if (descriptor.confirmation != null) {
      final confirmation = descriptor.confirmation!;

      // Interpolate variables in confirmation message
      var message = confirmation.message;
      payload.forEach((key, value) {
        message = message.replaceAll('{$key}', value.toString());
      });

      state = ActionState.confirming(
        actionId: actionId,
        message: message,
        isDestructive: confirmation.style == ConfirmationStyle.destructive,
        payload: payload,
      );
      return;
    }

    // No confirmation needed, execute immediately
    await execute(payload: payload);
  }

  /// Execute the action (called after confirmation or directly)
  Future<void> execute({Map<String, dynamic> payload = const {}}) async {
    state = const ActionState.submitting();
    final startTime = DateTime.now();
    final telemetryService = ref.read(telemetryServiceProvider);

    try {
      _logger.info('Executing action: $actionId');

      final bffClient = ref.read(bffClientProvider);

      // Execute the action
      final response = await bffClient.executeAction(actionId, data: payload);

      _logger.info('Action executed successfully: $actionId');

      // Record success telemetry
      final durationMs = DateTime.now().difference(startTime).inMilliseconds;
      telemetryService.record(
        TelemetryEvent.actionExecution(
          actionId: actionId,
          actionType: 'bff_action',
          pageId: 'unknown', // TODO: Track current page context
          success: true,
          durationMs: durationMs,
          timestamp: DateTime.now(),
        ),
      );

      // Extract response data
      final message = response['message'] as String?;
      final navigationUrl = response['navigationUrl'] as String?;
      final invalidate = response['invalidate'] as List?;

      // Invalidate specified providers
      if (invalidate != null) {
        for (final target in invalidate) {
          _invalidateProvider(target as String);
        }
      }

      state = ActionState.success(
        response: response,
        message: message,
        navigationUrl: navigationUrl,
      );
    } catch (e, stack) {
      _logger.severe('Action execution failed: $actionId', e, stack);

      // Record error telemetry
      final durationMs = DateTime.now().difference(startTime).inMilliseconds;
      telemetryService.record(
        TelemetryEvent.actionExecution(
          actionId: actionId,
          actionType: 'bff_action',
          pageId: 'unknown', // TODO: Track current page context
          success: false,
          durationMs: durationMs,
          errorMessage: e.toString(),
          timestamp: DateTime.now(),
        ),
      );

      // Extract error details
      String message = 'Action failed';
      Map<String, String>? fieldErrors;

      if (e is AppError) {
        message = e.message;
        if (e is ValidationError) {
          fieldErrors = e.fieldErrors;
        }
      } else {
        message = e.toString();
      }

      state = ActionState.error(
        message: message,
        fieldErrors: fieldErrors,
      );
    }
  }

  /// Cancel confirmation dialog
  void cancelConfirmation() {
    _logger.info('Action confirmation cancelled: $actionId');
    state = const ActionState.idle();
  }

  /// Reset to idle state
  void reset() {
    state = const ActionState.idle();
  }

  /// Invalidate a provider by name
  void _invalidateProvider(String target) {
    _logger.fine('Invalidating provider: $target');

    // Parse target format: "type:id" or just "type"
    final parts = target.split(':');
    final providerType = parts[0];
    final providerId = parts.length > 1 ? parts[1] : null;

    // Invalidate based on provider type
    switch (providerType) {
      case 'page':
        if (providerId != null) {
          // Invalidate specific page
          // ref.invalidate(pageProvider(providerId));
        }
        break;
      case 'navigation':
        // ref.invalidate(navigationProvider);
        break;
      case 'schema':
        if (providerId != null) {
          // ref.invalidate(schemaProvider(providerId));
        }
        break;
      case 'table':
        // Refresh table data - implementation depends on table provider setup
        break;
      default:
        _logger.warning('Unknown provider type for invalidation: $providerType');
    }
  }
}
