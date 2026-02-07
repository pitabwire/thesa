package workflow

import (
	"context"
	"time"

	"github.com/pitabwire/thesa/model"
)

// WorkflowStore persists workflow instances and events.
type WorkflowStore interface {
	// Create persists a new workflow instance.
	Create(ctx context.Context, instance model.WorkflowInstance) error

	// Get retrieves a workflow instance by ID, scoped to a tenant.
	// Returns NOT_FOUND if the instance doesn't exist or belongs to a
	// different tenant.
	Get(ctx context.Context, tenantID, instanceID string) (model.WorkflowInstance, error)

	// Update persists an updated workflow instance with optimistic locking.
	// The version must match the current stored version. Returns CONFLICT if
	// the version has changed.
	Update(ctx context.Context, instance model.WorkflowInstance) error

	// AppendEvent adds an event to the workflow's audit trail.
	AppendEvent(ctx context.Context, event model.WorkflowEvent) error

	// GetEvents retrieves all events for a workflow instance, scoped to a
	// tenant.
	GetEvents(ctx context.Context, tenantID, instanceID string) ([]model.WorkflowEvent, error)

	// FindActive returns active workflow instances for a tenant, optionally
	// filtered by workflow ID.
	FindActive(ctx context.Context, tenantID string, filters WorkflowFilters) ([]model.WorkflowInstance, error)

	// FindExpired returns active instances whose expires_at is before the
	// given cutoff time.
	FindExpired(ctx context.Context, cutoff time.Time) ([]model.WorkflowInstance, error)

	// Delete removes a workflow instance and its events.
	Delete(ctx context.Context, tenantID, instanceID string) error
}

// WorkflowFilters are optional filters for listing workflow instances.
type WorkflowFilters struct {
	WorkflowID string
	Status     string
	Limit      int
	Offset     int
}
