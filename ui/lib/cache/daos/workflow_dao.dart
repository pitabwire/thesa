/// Data Access Object for workflow state operations.
library;

import 'package:drift/drift.dart';

import '../database/app_database.dart';
import '../database/tables/workflow_state.dart';

part 'workflow_dao.g.dart';

/// DAO for workflow state operations
@DriftAccessor(tables: [WorkflowState])
class WorkflowDao extends DatabaseAccessor<AppDatabase> with _$WorkflowDaoMixin {
  /// Creates a new WorkflowDao
  WorkflowDao(super.db);

  /// Watch a specific workflow (reactive stream)
  Stream<WorkflowStateEntry?> watchWorkflow(String instanceId) {
    return (select(workflowState)..where((t) => t.instanceId.equals(instanceId)))
        .watchSingleOrNull();
  }

  /// Get a specific workflow (one-time read)
  Future<WorkflowStateEntry?> getWorkflow(String instanceId) {
    return (select(workflowState)..where((t) => t.instanceId.equals(instanceId)))
        .getSingleOrNull();
  }

  /// Save workflow state
  Future<void> saveWorkflow(WorkflowStateCompanion entry) {
    return into(workflowState).insertOnConflictUpdate(entry);
  }

  /// Update workflow step
  Future<void> updateWorkflowStep(String instanceId, int stepIndex) {
    return (update(workflowState)..where((t) => t.instanceId.equals(instanceId)))
        .write(
      WorkflowStateCompanion(
        currentStep: Value(stepIndex),
        updatedAt: Value(DateTime.now()),
      ),
    );
  }

  /// Update workflow data
  Future<void> updateWorkflowData(String instanceId, String data) {
    return (update(workflowState)..where((t) => t.instanceId.equals(instanceId)))
        .write(
      WorkflowStateCompanion(
        data: Value(data),
        updatedAt: Value(DateTime.now()),
      ),
    );
  }

  /// Complete workflow
  Future<void> completeWorkflow(String instanceId) {
    return (update(workflowState)..where((t) => t.instanceId.equals(instanceId)))
        .write(
      WorkflowStateCompanion(
        status: const Value('completed'),
        completedAt: Value(DateTime.now()),
        updatedAt: Value(DateTime.now()),
      ),
    );
  }

  /// Cancel workflow
  Future<void> cancelWorkflow(String instanceId) {
    return (update(workflowState)..where((t) => t.instanceId.equals(instanceId)))
        .write(
      WorkflowStateCompanion(
        status: const Value('cancelled'),
        updatedAt: Value(DateTime.now()),
      ),
    );
  }

  /// Mark workflow as errored
  Future<void> errorWorkflow(String instanceId, String errorMessage) {
    return (update(workflowState)..where((t) => t.instanceId.equals(instanceId)))
        .write(
      WorkflowStateCompanion(
        status: const Value('error'),
        errorMessage: Value(errorMessage),
        updatedAt: Value(DateTime.now()),
      ),
    );
  }

  /// Delete workflow
  Future<void> deleteWorkflow(String instanceId) {
    return (delete(workflowState)..where((t) => t.instanceId.equals(instanceId)))
        .go();
  }

  /// Get all in-progress workflows
  Future<List<WorkflowStateEntry>> getInProgressWorkflows() {
    return (select(workflowState)..where((t) => t.status.equals('in_progress')))
        .get();
  }

  /// Get all workflows for a specific workflow type
  Future<List<WorkflowStateEntry>> getWorkflowsByType(String workflowId) {
    return (select(workflowState)..where((t) => t.workflowId.equals(workflowId)))
        .get();
  }

  /// Clear completed workflows older than a certain date
  Future<void> clearOldCompletedWorkflows(DateTime before) {
    return (delete(workflowState)
          ..where((t) =>
              t.status.equals('completed') &
              t.completedAt.isSmallerThanValue(before)))
        .go();
  }
}
