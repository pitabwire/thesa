/// DAO for workflow state persistence.
library;

import 'package:drift/drift.dart';

import '../app_database.dart';
import '../tables/workflow_state.dart';

part 'workflow_dao.g.dart';

/// Workflow DAO for state persistence
@DriftAccessor(tables: [WorkflowState])
class WorkflowDao extends DatabaseAccessor<AppDatabase>
    with _$WorkflowDaoMixin {
  WorkflowDao(super.db);

  /// Get workflow state by instance ID
  Future<WorkflowStateEntry?> getWorkflow(String instanceId) {
    return (select(workflowState)
          ..where((tbl) => tbl.instanceId.equals(instanceId)))
        .getSingleOrNull();
  }

  /// Save workflow state
  Future<void> saveWorkflowState({
    required String instanceId,
    required String workflowId,
    required Map<String, dynamic> payload,
    required int currentStep,
  }) async {
    final now = DateTime.now();

    // Check if this is a new workflow or an update
    final existing = await getWorkflow(instanceId);

    await into(workflowState).insertOnConflictUpdate(
      WorkflowStateCompanion.insert(
        instanceId: instanceId,
        workflowId: workflowId,
        data: payload.toString(), // Store as JSON string
        currentStep: currentStep,
        status: 'in_progress',
        startedAt: existing?.startedAt ?? now,
        updatedAt: now,
      ),
    );
  }

  /// Mark workflow as completed
  Future<void> markCompleted(String instanceId) async {
    final now = DateTime.now();
    await (update(workflowState)
          ..where((tbl) => tbl.instanceId.equals(instanceId)))
        .write(
      WorkflowStateCompanion(
        status: const Value('completed'),
        completedAt: Value(now),
        updatedAt: Value(now),
      ),
    );
  }

  /// Mark workflow as cancelled
  Future<void> markCancelled(String instanceId) async {
    final now = DateTime.now();
    await (update(workflowState)
          ..where((tbl) => tbl.instanceId.equals(instanceId)))
        .write(
      WorkflowStateCompanion(
        status: const Value('cancelled'),
        completedAt: Value(now),
        updatedAt: Value(now),
      ),
    );
  }

  /// Delete workflow state
  Future<void> deleteWorkflow(String instanceId) async {
    await (delete(workflowState)
          ..where((tbl) => tbl.instanceId.equals(instanceId)))
        .go();
  }

  /// Get all active workflows (in progress)
  Future<List<WorkflowStateEntry>> getActiveWorkflows() {
    return (select(workflowState)
          ..where((tbl) => tbl.status.equals('in_progress'))
          ..orderBy([
            (tbl) => OrderingTerm(
                  expression: tbl.updatedAt,
                  mode: OrderingMode.desc,
                ),
          ]))
        .get();
  }

  /// Get all completed workflows
  Future<List<WorkflowStateEntry>> getCompletedWorkflows() {
    return (select(workflowState)
          ..where((tbl) => tbl.status.equals('completed'))
          ..orderBy([
            (tbl) => OrderingTerm(
                  expression: tbl.updatedAt,
                  mode: OrderingMode.desc,
                ),
          ]))
        .get();
  }

  /// Clear old completed/cancelled workflows (cleanup)
  Future<void> clearOldWorkflows({Duration age = const Duration(days: 30)}) async {
    final cutoff = DateTime.now().subtract(age);
    await (delete(workflowState)
          ..where((tbl) =>
              (tbl.status.equals('completed') | tbl.status.equals('cancelled')) &
              tbl.updatedAt.isSmallerThanValue(cutoff)))
        .go();
  }
}
