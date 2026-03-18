/// Workflow state table for storing in-progress workflows.
///
/// Unlike other caches, workflow state never expires - it persists
/// until the workflow is completed or explicitly cancelled.
library;

import 'package:drift/drift.dart';

/// Stores workflow progress and state
@DataClassName('WorkflowStateEntry')
class WorkflowState extends Table {
  @override
  String get tableName => 'workflow_state';

  @override
  Set<Column> get primaryKey => {instanceId};

  /// Unique instance ID for this workflow execution
  TextColumn get instanceId => text()();

  /// Workflow descriptor ID
  TextColumn get workflowId => text()();

  /// Current step index
  IntColumn get currentStep => integer()();

  /// Accumulated workflow data (JSON)
  TextColumn get data => text()();

  /// Workflow status (in_progress, completed, cancelled, error)
  TextColumn get status => text()();

  /// When this workflow was started
  DateTimeColumn get startedAt => dateTime()();

  /// When this workflow was completed (if applicable)
  DateTimeColumn get completedAt => dateTime().nullable()();

  /// Last update timestamp
  DateTimeColumn get updatedAt => dateTime()();

  /// Error message (if status is error)
  TextColumn get errorMessage => text().nullable()();
}
