package model

import "time"

// Workflow instance status constants.
const (
	WorkflowStatusActive    = "active"
	WorkflowStatusCompleted = "completed"
	WorkflowStatusFailed    = "failed"
	WorkflowStatusCancelled = "cancelled"
	WorkflowStatusSuspended = "suspended"
)

// Workflow step status constants.
const (
	StepStatusPending    = "pending"
	StepStatusInProgress = "in_progress"
	StepStatusCompleted  = "completed"
	StepStatusSkipped    = "skipped"
	StepStatusFailed     = "failed"
	StepStatusFuture     = "future"
)

// WorkflowInstance represents a running workflow instance.
type WorkflowInstance struct {
	ID             string         `json:"id"`
	WorkflowID     string         `json:"workflow_id"`
	TenantID       string         `json:"tenant_id"`
	PartitionID    string         `json:"partition_id"`
	SubjectID      string         `json:"subject_id"`
	CurrentStep    string         `json:"current_step"`
	Status         string         `json:"status"`
	State          map[string]any `json:"state,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	ExpiresAt      *time.Time     `json:"expires_at,omitempty"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	Version        int            `json:"version"`
}

// WorkflowSummary is a lightweight representation of a workflow instance
// used in list views.
type WorkflowSummary struct {
	ID          string    `json:"id"`
	WorkflowID  string    `json:"workflow_id"`
	Name        string    `json:"name"`
	CurrentStep string    `json:"current_step"`
	Status      string    `json:"status"`
	SubjectID   string    `json:"subject_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WorkflowEvent records an event in a workflow's audit trail.
type WorkflowEvent struct {
	ID                 string    `json:"id"`
	WorkflowInstanceID string    `json:"workflow_instance_id"`
	StepID             string    `json:"step_id"`
	Event              string    `json:"event"`
	ActorID            string    `json:"actor_id"`
	Data               map[string]any `json:"data,omitempty"`
	Comment            string    `json:"comment,omitempty"`
	Timestamp          time.Time `json:"timestamp"`
}
