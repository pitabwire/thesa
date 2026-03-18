package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pitabwire/thesa/model"
)

// PgWorkflowStore is a PostgreSQL-backed WorkflowStore using pgx/v5.
type PgWorkflowStore struct {
	pool *pgxpool.Pool
}

// NewPgWorkflowStore creates a new PostgreSQL workflow store.
func NewPgWorkflowStore(pool *pgxpool.Pool) *PgWorkflowStore {
	return &PgWorkflowStore{pool: pool}
}

// EnsureSchema creates the workflow tables if they do not already exist.
func (s *PgWorkflowStore) EnsureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS workflow_instances (
			id              TEXT PRIMARY KEY,
			workflow_id     TEXT NOT NULL,
			tenant_id       TEXT NOT NULL,
			partition_id    TEXT NOT NULL DEFAULT '',
			subject_id      TEXT NOT NULL,
			current_step    TEXT NOT NULL,
			status          TEXT NOT NULL DEFAULT 'active',
			state           JSONB DEFAULT '{}',
			version         INTEGER NOT NULL DEFAULT 1,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			expires_at      TIMESTAMPTZ,
			idempotency_key TEXT
		)`)
	if err != nil {
		return fmt.Errorf("create workflow_instances table: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS workflow_events (
			id                    TEXT PRIMARY KEY,
			workflow_instance_id  TEXT NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,
			step_id               TEXT NOT NULL,
			event                 TEXT NOT NULL,
			actor_id              TEXT NOT NULL DEFAULT '',
			data                  JSONB DEFAULT '{}',
			comment               TEXT DEFAULT '',
			created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`)
	if err != nil {
		return fmt.Errorf("create workflow_events table: %w", err)
	}

	// Create indexes for common query patterns.
	for _, ddl := range []string{
		`CREATE INDEX IF NOT EXISTS idx_wf_inst_tenant_status ON workflow_instances (tenant_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_wf_inst_workflow_id ON workflow_instances (workflow_id)`,
		`CREATE INDEX IF NOT EXISTS idx_wf_inst_expires ON workflow_instances (expires_at) WHERE status = 'active' AND expires_at IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_wf_inst_idempotency ON workflow_instances (idempotency_key) WHERE idempotency_key IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_wf_events_instance ON workflow_events (workflow_instance_id, created_at)`,
	} {
		if _, err := s.pool.Exec(ctx, ddl); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}

	return nil
}

// HealthCheck verifies the database connection is alive.
func (s *PgWorkflowStore) HealthCheck(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Create inserts a new workflow instance.
func (s *PgWorkflowStore) Create(ctx context.Context, inst model.WorkflowInstance) error {
	stateJSON, err := json.Marshal(inst.State)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO workflow_instances (
			id, workflow_id, tenant_id, partition_id, subject_id,
			current_step, status, state, version,
			created_at, updated_at, expires_at, idempotency_key
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13
		)`,
		inst.ID, inst.WorkflowID, inst.TenantID, inst.PartitionID, inst.SubjectID,
		inst.CurrentStep, inst.Status, stateJSON, inst.Version,
		inst.CreatedAt, inst.UpdatedAt, inst.ExpiresAt, inst.IdempotencyKey,
	)
	if err != nil {
		return fmt.Errorf("insert workflow instance: %w", err)
	}
	return nil
}

// Get retrieves a workflow instance by ID, scoped to tenant.
func (s *PgWorkflowStore) Get(ctx context.Context, tenantID, instanceID string) (model.WorkflowInstance, error) {
	var inst model.WorkflowInstance
	var stateJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT id, workflow_id, tenant_id, partition_id, subject_id,
		       current_step, status, state, version,
		       created_at, updated_at, expires_at, idempotency_key
		FROM workflow_instances
		WHERE id = $1 AND tenant_id = $2`,
		instanceID, tenantID,
	).Scan(
		&inst.ID, &inst.WorkflowID, &inst.TenantID, &inst.PartitionID, &inst.SubjectID,
		&inst.CurrentStep, &inst.Status, &stateJSON, &inst.Version,
		&inst.CreatedAt, &inst.UpdatedAt, &inst.ExpiresAt, &inst.IdempotencyKey,
	)
	if err == pgx.ErrNoRows {
		return model.WorkflowInstance{}, model.NewNotFoundError(
			fmt.Sprintf("workflow instance %q not found", instanceID),
		)
	}
	if err != nil {
		return model.WorkflowInstance{}, fmt.Errorf("query workflow instance: %w", err)
	}

	if stateJSON != nil {
		if err := json.Unmarshal(stateJSON, &inst.State); err != nil {
			return model.WorkflowInstance{}, fmt.Errorf("unmarshal state: %w", err)
		}
	}

	return inst, nil
}

// Update persists an updated instance with optimistic locking.
func (s *PgWorkflowStore) Update(ctx context.Context, inst model.WorkflowInstance) error {
	stateJSON, err := json.Marshal(inst.State)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	tag, err := s.pool.Exec(ctx, `
		UPDATE workflow_instances SET
			current_step = $1,
			status = $2,
			state = $3,
			version = $4,
			updated_at = $5,
			expires_at = $6
		WHERE id = $7 AND version = $8`,
		inst.CurrentStep, inst.Status, stateJSON, inst.Version+1,
		time.Now().UTC(), inst.ExpiresAt,
		inst.ID, inst.Version,
	)
	if err != nil {
		return fmt.Errorf("update workflow instance: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.NewConflictError(
			fmt.Sprintf("workflow instance %q version conflict (expected %d)", inst.ID, inst.Version),
		)
	}
	return nil
}

// AppendEvent adds an event to the workflow audit trail.
func (s *PgWorkflowStore) AppendEvent(ctx context.Context, event model.WorkflowEvent) error {
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO workflow_events (
			id, workflow_instance_id, step_id, event, actor_id, data, comment, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		event.ID, event.WorkflowInstanceID, event.StepID, event.Event,
		event.ActorID, dataJSON, event.Comment, event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert workflow event: %w", err)
	}
	return nil
}

// GetEvents retrieves all events for a workflow instance.
func (s *PgWorkflowStore) GetEvents(ctx context.Context, tenantID, instanceID string) ([]model.WorkflowEvent, error) {
	// Verify tenant access.
	_, err := s.Get(ctx, tenantID, instanceID)
	if err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, workflow_instance_id, step_id, event, actor_id, data, comment, created_at
		FROM workflow_events
		WHERE workflow_instance_id = $1
		ORDER BY created_at ASC`,
		instanceID,
	)
	if err != nil {
		return nil, fmt.Errorf("query workflow events: %w", err)
	}
	defer rows.Close()

	var events []model.WorkflowEvent
	for rows.Next() {
		var evt model.WorkflowEvent
		var dataJSON []byte
		if err := rows.Scan(
			&evt.ID, &evt.WorkflowInstanceID, &evt.StepID, &evt.Event,
			&evt.ActorID, &dataJSON, &evt.Comment, &evt.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("scan workflow event: %w", err)
		}
		if dataJSON != nil {
			_ = json.Unmarshal(dataJSON, &evt.Data)
		}
		events = append(events, evt)
	}
	return events, rows.Err()
}

// FindActive returns active workflow instances for a tenant.
func (s *PgWorkflowStore) FindActive(ctx context.Context, tenantID string, filters WorkflowFilters) ([]model.WorkflowInstance, error) {
	query := `SELECT id, workflow_id, tenant_id, partition_id, subject_id,
	                 current_step, status, state, version,
	                 created_at, updated_at, expires_at, idempotency_key
	          FROM workflow_instances
	          WHERE tenant_id = $1 AND status = 'active'`
	args := []any{tenantID}
	argIdx := 2

	if filters.WorkflowID != "" {
		query += fmt.Sprintf(" AND workflow_id = $%d", argIdx)
		args = append(args, filters.WorkflowID)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filters.Limit)
		argIdx++
	}
	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filters.Offset)
	}

	return s.queryInstances(ctx, query, args...)
}

// FindExpired returns active instances past their expiration time.
func (s *PgWorkflowStore) FindExpired(ctx context.Context, cutoff time.Time) ([]model.WorkflowInstance, error) {
	query := `SELECT id, workflow_id, tenant_id, partition_id, subject_id,
	                 current_step, status, state, version,
	                 created_at, updated_at, expires_at, idempotency_key
	          FROM workflow_instances
	          WHERE status = 'active' AND expires_at IS NOT NULL AND expires_at < $1
	          ORDER BY expires_at ASC`
	return s.queryInstances(ctx, query, cutoff)
}

// Delete removes a workflow instance and its events.
func (s *PgWorkflowStore) Delete(ctx context.Context, tenantID, instanceID string) error {
	// Delete events first (foreign key).
	_, err := s.pool.Exec(ctx, `
		DELETE FROM workflow_events
		WHERE workflow_instance_id = $1
		AND workflow_instance_id IN (SELECT id FROM workflow_instances WHERE tenant_id = $2)`,
		instanceID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("delete workflow events: %w", err)
	}

	tag, err := s.pool.Exec(ctx, `
		DELETE FROM workflow_instances
		WHERE id = $1 AND tenant_id = $2`,
		instanceID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("delete workflow instance: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.NewNotFoundError(
			fmt.Sprintf("workflow instance %q not found", instanceID),
		)
	}
	return nil
}

// queryInstances executes a query and returns workflow instances.
func (s *PgWorkflowStore) queryInstances(ctx context.Context, query string, args ...any) ([]model.WorkflowInstance, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query workflow instances: %w", err)
	}
	defer rows.Close()

	var instances []model.WorkflowInstance
	for rows.Next() {
		var inst model.WorkflowInstance
		var stateJSON []byte
		if err := rows.Scan(
			&inst.ID, &inst.WorkflowID, &inst.TenantID, &inst.PartitionID, &inst.SubjectID,
			&inst.CurrentStep, &inst.Status, &stateJSON, &inst.Version,
			&inst.CreatedAt, &inst.UpdatedAt, &inst.ExpiresAt, &inst.IdempotencyKey,
		); err != nil {
			return nil, fmt.Errorf("scan workflow instance: %w", err)
		}
		if stateJSON != nil {
			_ = json.Unmarshal(stateJSON, &inst.State)
		}
		instances = append(instances, inst)
	}
	return instances, rows.Err()
}
