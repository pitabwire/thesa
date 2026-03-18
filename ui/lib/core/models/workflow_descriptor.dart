/// Workflow descriptors for multi-step processes.
///
/// Workflows guide users through complex, multi-step operations like:
/// - Order fulfillment
/// - Refund processing
/// - Onboarding
/// - Approval flows
library;

import 'package:freezed_annotation/freezed_annotation.dart';

import 'action_descriptor.dart';
import 'component_descriptor.dart';
import 'permission.dart';

part 'workflow_descriptor.freezed.dart';
part 'workflow_descriptor.g.dart';

/// Complete workflow definition
@freezed
class WorkflowDescriptor with _$WorkflowDescriptor {
  const factory WorkflowDescriptor({
    /// Unique workflow identifier
    required String workflowId,

    /// Workflow title
    required String title,

    /// Description
    String? description,

    /// Workflow steps
    required List<WorkflowStep> steps,

    /// Current step index (0-based)
    @Default(0) int currentStep,

    /// Workflow state data (accumulated as user progresses)
    @Default({}) Map<String, dynamic> state,

    /// Permission check
    @Default(Permission(allowed: true)) Permission permission,

    /// Whether user can go back to previous steps
    @Default(true) bool allowBack,

    /// Whether user can skip steps
    @Default(false) bool allowSkip,

    /// Whether workflow can be saved and resumed later
    @Default(true) bool resumable,

    /// Auto-save interval (seconds), 0 = disabled
    @Default(0) int autoSaveInterval,

    /// Version
    String? version,
  }) = _WorkflowDescriptor;

  factory WorkflowDescriptor.fromJson(Map<String, dynamic> json) =>
      _$WorkflowDescriptorFromJson(json);
}

/// A single step in a workflow
@freezed
class WorkflowStep with _$WorkflowStep {
  const factory WorkflowStep({
    /// Step identifier
    required String stepId,

    /// Step title
    required String title,

    /// Step description
    String? description,

    /// Icon
    String? icon,

    /// Step type
    @Default(WorkflowStepType.form) WorkflowStepType stepType,

    /// Components to render in this step
    List<ComponentDescriptor>? components,

    /// Schema for form input (if stepType is form)
    String? schemaRef,

    /// Actions available in this step
    List<ActionDescriptor>? actions,

    /// Validation rules for progressing to next step
    StepValidation? validation,

    /// Condition for showing this step
    StepCondition? condition,

    /// Permission check
    @Default(Permission(allowed: true)) Permission permission,

    /// Whether this step is optional
    @Default(false) bool optional,

    /// Estimated time to complete (in minutes)
    int? estimatedMinutes,
  }) = _WorkflowStep;

  factory WorkflowStep.fromJson(Map<String, dynamic> json) =>
      _$WorkflowStepFromJson(json);
}

/// Type of workflow step
enum WorkflowStepType {
  /// Form input step
  form,

  /// Review/confirmation step
  review,

  /// Information/instruction step
  info,

  /// Selection step
  selection,

  /// Custom step (requires plugin)
  custom,
}

/// Validation rules for progressing to next step
@freezed
class StepValidation with _$StepValidation {
  const factory StepValidation({
    /// Required fields that must be filled
    List<String>? requiredFields,

    /// Custom validation endpoint to call
    String? endpoint,

    /// Error message if validation fails
    String? errorMessage,
  }) = _StepValidation;

  factory StepValidation.fromJson(Map<String, dynamic> json) =>
      _$StepValidationFromJson(json);
}

/// Condition for showing a step
@freezed
class StepCondition with _$StepCondition {
  const factory StepCondition({
    /// Field to check
    required String field,

    /// Show step if field equals this value
    dynamic equals,

    /// Show step if field is one of these values
    List<dynamic>? oneOf,

    /// Show step if field is not empty
    @Default(false) bool notEmpty,
  }) = _StepCondition;

  factory StepCondition.fromJson(Map<String, dynamic> json) =>
      _$StepConditionFromJson(json);
}

/// Workflow execution state
@freezed
class WorkflowState with _$WorkflowState {
  const factory WorkflowState({
    /// Workflow instance ID
    required String instanceId,

    /// Workflow descriptor ID
    required String workflowId,

    /// Current step index
    required int currentStep,

    /// Accumulated state data
    @Default({}) Map<String, dynamic> data,

    /// Workflow status
    @Default(WorkflowStatus.inProgress) WorkflowStatus status,

    /// Start time
    DateTime? startedAt,

    /// Completion time
    DateTime? completedAt,

    /// Last update time
    DateTime? updatedAt,

    /// Error message (if status is error)
    String? errorMessage,
  }) = _WorkflowState;

  factory WorkflowState.fromJson(Map<String, dynamic> json) =>
      _$WorkflowStateFromJson(json);
}

/// Workflow execution status
enum WorkflowStatus {
  /// Workflow is in progress
  inProgress,

  /// Workflow completed successfully
  completed,

  /// Workflow was cancelled
  cancelled,

  /// Workflow encountered an error
  error,
}
