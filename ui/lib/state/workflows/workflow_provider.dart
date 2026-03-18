/// Workflow provider for workflow state management.
///
/// Tracks workflow progress and manages step transitions.
library;

import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../core/core.dart';
import '../core/dependencies_provider.dart';

part 'workflow_provider.g.dart';

final _logger = Logger('WorkflowProvider');

/// Workflow provider (family - one instance per workflow instance ID)
@riverpod
class Workflow extends _$Workflow {
  @override
  Future<WorkflowDescriptor> build(String instanceId) async {
    _logger.info('Loading workflow: $instanceId');

    final database = await ref.read(databaseProvider.future);
    final bffClient = ref.read(bffClientProvider);

    try {
      // Check if we have persisted workflow state
      final persistedState = await database.workflowDao.getWorkflow(instanceId);

      if (persistedState != null) {
        // Load workflow from persisted state
        _logger.info(
          'Loaded workflow from cache: $instanceId '
          '(step ${persistedState.currentStep})',
        );

        // Parse the workflow descriptor from data field
        // Note: data is stored as JSON string in the database
        final descriptor = WorkflowDescriptor.fromJson(
          Map<String, dynamic>.from(
            Map<String, dynamic>.from(persistedState.data as Object),
          ),
        );

        return descriptor.copyWith(currentStep: persistedState.currentStep);
      }

      // No persisted state - load from network
      // Extract workflow ID from instance ID (format: workflowId_instanceId)
      final workflowId = instanceId.split('_').first;
      final descriptor = await bffClient.getWorkflow(workflowId);

      _logger.info('Loaded workflow from network: $instanceId');

      // Save initial state
      await _saveWorkflowState(instanceId, descriptor, 0);

      return descriptor;
    } catch (e, stack) {
      _logger.severe('Failed to load workflow: $instanceId', e, stack);
      rethrow;
    }
  }

  /// Advance to next step
  Future<void> nextStep(Map<String, dynamic> stepData) async {
    final current = state.valueOrNull;
    if (current == null) return;

    _logger.info('Advancing workflow to next step: $instanceId');

    final bffClient = ref.read(bffClientProvider);

    try {
      // Validate that we're not past the last step
      if (current.currentStep >= current.steps.length - 1) {
        _logger.warning('Cannot advance: already at last step');
        return;
      }

      // Submit current step data to BFF
      final currentStepDescriptor = current.steps[current.currentStep];
      await bffClient.submitWorkflowStep(
        current.workflowId,
        currentStepDescriptor.stepId,
        data: stepData,
      );

      // Advance to next step
      final newStep = current.currentStep + 1;
      final updated = current.copyWith(currentStep: newStep);

      // Update state and persist
      state = AsyncValue.data(updated);
      await _saveWorkflowState(instanceId, updated, newStep);

      _logger.info('Advanced to step $newStep');
    } catch (e, stack) {
      _logger.severe('Failed to advance workflow step', e, stack);
      rethrow;
    }
  }

  /// Go back to previous step
  Future<void> previousStep() async {
    final current = state.valueOrNull;
    if (current == null || current.currentStep == 0) return;

    _logger.info('Going back to previous step: $instanceId');

    try {
      final newStep = current.currentStep - 1;
      final updated = current.copyWith(currentStep: newStep);

      // Update state and persist
      state = AsyncValue.data(updated);
      await _saveWorkflowState(instanceId, updated, newStep);

      _logger.info('Went back to step $newStep');
    } catch (e, stack) {
      _logger.severe('Failed to go back to previous step', e, stack);
      rethrow;
    }
  }

  /// Complete workflow
  Future<void> complete() async {
    final current = state.valueOrNull;
    if (current == null) return;

    _logger.info('Completing workflow: $instanceId');

    final bffClient = ref.read(bffClientProvider);
    final database = await ref.read(databaseProvider.future);

    try {
      // Submit final step
      final finalStep = current.steps.last;
      await bffClient.submitWorkflowStep(
        current.workflowId,
        finalStep.stepId,
        data: {}, // Final submission
      );

      // Mark as completed in database
      await database.workflowDao.markCompleted(instanceId);

      _logger.info('Workflow completed: $instanceId');
    } catch (e, stack) {
      _logger.severe('Failed to complete workflow', e, stack);
      rethrow;
    }
  }

  /// Cancel workflow
  Future<void> cancel() async {
    _logger.info('Cancelling workflow: $instanceId');

    final database = await ref.read(databaseProvider.future);

    try {
      await database.workflowDao.markCancelled(instanceId);
      _logger.info('Workflow cancelled: $instanceId');
    } catch (e, stack) {
      _logger.severe('Failed to cancel workflow', e, stack);
      rethrow;
    }
  }

  /// Save workflow state to database
  Future<void> _saveWorkflowState(
    String instanceId,
    WorkflowDescriptor descriptor,
    int currentStep,
  ) async {
    final database = await ref.read(databaseProvider.future);
    await database.workflowDao.saveWorkflowState(
      instanceId: instanceId,
      workflowId: descriptor.workflowId,
      payload: descriptor.toJson(),
      currentStep: currentStep,
    );
  }
}
