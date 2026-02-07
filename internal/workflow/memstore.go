package workflow

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/pitabwire/thesa/model"
)

// MemoryWorkflowStore is an in-memory WorkflowStore for testing.
type MemoryWorkflowStore struct {
	mu        sync.RWMutex
	instances map[string]model.WorkflowInstance // key: instance ID
	events    map[string][]model.WorkflowEvent  // key: instance ID
}

// NewMemoryWorkflowStore creates a new in-memory workflow store.
func NewMemoryWorkflowStore() *MemoryWorkflowStore {
	return &MemoryWorkflowStore{
		instances: make(map[string]model.WorkflowInstance),
		events:    make(map[string][]model.WorkflowEvent),
	}
}

// Create persists a new workflow instance.
func (s *MemoryWorkflowStore) Create(_ context.Context, inst model.WorkflowInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.instances[inst.ID]; exists {
		return model.NewConflictError(
			fmt.Sprintf("workflow instance %q already exists", inst.ID),
		)
	}

	s.instances[inst.ID] = inst
	return nil
}

// Get retrieves a workflow instance by ID, scoped to tenant.
func (s *MemoryWorkflowStore) Get(_ context.Context, tenantID, instanceID string) (model.WorkflowInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	inst, exists := s.instances[instanceID]
	if !exists || inst.TenantID != tenantID {
		return model.WorkflowInstance{}, model.NewNotFoundError(
			fmt.Sprintf("workflow instance %q not found", instanceID),
		)
	}
	return inst, nil
}

// Update persists an updated instance with optimistic locking.
func (s *MemoryWorkflowStore) Update(_ context.Context, inst model.WorkflowInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.instances[inst.ID]
	if !exists {
		return model.NewNotFoundError(
			fmt.Sprintf("workflow instance %q not found", inst.ID),
		)
	}

	// Optimistic lock check.
	if existing.Version != inst.Version {
		return model.NewConflictError(
			fmt.Sprintf("workflow instance %q version conflict (expected %d, got %d)", inst.ID, inst.Version, existing.Version),
		)
	}

	inst.Version++
	inst.UpdatedAt = time.Now().UTC()
	s.instances[inst.ID] = inst
	return nil
}

// AppendEvent adds an event to the workflow's audit trail.
func (s *MemoryWorkflowStore) AppendEvent(_ context.Context, event model.WorkflowEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events[event.WorkflowInstanceID] = append(s.events[event.WorkflowInstanceID], event)
	return nil
}

// GetEvents retrieves all events for a workflow instance, ordered by timestamp.
func (s *MemoryWorkflowStore) GetEvents(_ context.Context, tenantID, instanceID string) ([]model.WorkflowEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Verify tenant access.
	inst, exists := s.instances[instanceID]
	if !exists || inst.TenantID != tenantID {
		return nil, model.NewNotFoundError(
			fmt.Sprintf("workflow instance %q not found", instanceID),
		)
	}

	events := s.events[instanceID]
	// Return sorted copy.
	result := make([]model.WorkflowEvent, len(events))
	copy(result, events)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})
	return result, nil
}

// FindActive returns active workflow instances for a tenant.
func (s *MemoryWorkflowStore) FindActive(_ context.Context, tenantID string, filters WorkflowFilters) ([]model.WorkflowInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []model.WorkflowInstance
	for _, inst := range s.instances {
		if inst.TenantID != tenantID {
			continue
		}
		if inst.Status != model.WorkflowStatusActive {
			continue
		}
		if filters.WorkflowID != "" && inst.WorkflowID != filters.WorkflowID {
			continue
		}
		result = append(result, inst)
	}

	// Sort by created_at descending.
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	// Apply offset and limit.
	if filters.Offset > 0 {
		if filters.Offset >= len(result) {
			return []model.WorkflowInstance{}, nil
		}
		result = result[filters.Offset:]
	}
	if filters.Limit > 0 && filters.Limit < len(result) {
		result = result[:filters.Limit]
	}

	return result, nil
}

// FindExpired returns active instances past their expiration time.
func (s *MemoryWorkflowStore) FindExpired(_ context.Context, cutoff time.Time) ([]model.WorkflowInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []model.WorkflowInstance
	for _, inst := range s.instances {
		if inst.Status != model.WorkflowStatusActive {
			continue
		}
		if inst.ExpiresAt == nil || !inst.ExpiresAt.Before(cutoff) {
			continue
		}
		result = append(result, inst)
	}

	// Sort by expires_at ascending.
	sort.Slice(result, func(i, j int) bool {
		return result[i].ExpiresAt.Before(*result[j].ExpiresAt)
	})

	return result, nil
}

// Delete removes a workflow instance and its events.
func (s *MemoryWorkflowStore) Delete(_ context.Context, tenantID, instanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	inst, exists := s.instances[instanceID]
	if !exists || inst.TenantID != tenantID {
		return model.NewNotFoundError(
			fmt.Sprintf("workflow instance %q not found", instanceID),
		)
	}

	delete(s.instances, instanceID)
	delete(s.events, instanceID)
	return nil
}

// Len returns the total number of instances. For testing.
func (s *MemoryWorkflowStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.instances)
}
