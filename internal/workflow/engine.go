package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/pitabwire/thesa/internal/command"
	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/model"
)

const defaultChainLimit = 10

// Engine manages the lifecycle of workflow instances.
type Engine struct {
	registry   *definition.Registry
	store      WorkflowStore
	invokers   *invoker.Registry
	capResolver model.CapabilityResolver
	chainLimit int
}

// NewEngine creates a new workflow engine.
func NewEngine(
	registry *definition.Registry,
	store WorkflowStore,
	invokers *invoker.Registry,
	capResolver model.CapabilityResolver,
) *Engine {
	return &Engine{
		registry:    registry,
		store:       store,
		invokers:    invokers,
		capResolver: capResolver,
		chainLimit:  defaultChainLimit,
	}
}

// Start creates a new workflow instance and enters the initial step.
func (e *Engine) Start(
	ctx context.Context,
	rctx *model.RequestContext,
	workflowID string,
	input map[string]any,
) (model.WorkflowInstance, error) {
	// 1. Look up workflow definition.
	wfDef, ok := e.registry.GetWorkflow(workflowID)
	if !ok {
		return model.WorkflowInstance{}, model.NewNotFoundError(
			fmt.Sprintf("workflow %q not found", workflowID),
		)
	}

	// 2. Check workflow-level capabilities.
	if len(wfDef.Capabilities) > 0 {
		caps, err := e.capResolver.Resolve(rctx)
		if err != nil {
			return model.WorkflowInstance{}, fmt.Errorf("resolve capabilities: %w", err)
		}
		if !caps.HasAll(wfDef.Capabilities...) {
			return model.WorkflowInstance{}, model.NewForbiddenError(
				fmt.Sprintf("insufficient capabilities for workflow %q", workflowID),
			)
		}
	}

	// 3. Compute expiration.
	now := time.Now().UTC()
	var expiresAt *time.Time
	if wfDef.Timeout != "" {
		dur, err := time.ParseDuration(wfDef.Timeout)
		if err == nil {
			exp := now.Add(dur)
			expiresAt = &exp
		}
	}

	// 4. Build initial state from input.
	state := make(map[string]any)
	for k, v := range input {
		state[k] = v
	}

	// 5. Create instance.
	inst := model.WorkflowInstance{
		ID:          uuid.New().String(),
		WorkflowID:  workflowID,
		TenantID:    rctx.TenantID,
		PartitionID: rctx.PartitionID,
		SubjectID:   rctx.SubjectID,
		CurrentStep: wfDef.InitialStep,
		Status:      model.WorkflowStatusActive,
		State:       state,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   expiresAt,
	}

	// 6. Persist instance.
	if err := e.store.Create(ctx, inst); err != nil {
		return model.WorkflowInstance{}, err
	}

	// 7. Append "step_entered" event.
	if err := e.appendEvent(ctx, inst.ID, wfDef.InitialStep, "step_entered", rctx.SubjectID, nil, ""); err != nil {
		return model.WorkflowInstance{}, err
	}

	// 8. If initial step is system/notification, auto-execute.
	stepDef := findStep(wfDef, wfDef.InitialStep)
	if stepDef != nil && isAutoStep(stepDef.Type) {
		var err error
		inst, err = e.executeStepChain(ctx, rctx, inst, wfDef, 0)
		if err != nil {
			return model.WorkflowInstance{}, err
		}
	}

	return inst, nil
}

// Advance processes an event on the current step and moves the workflow forward.
func (e *Engine) Advance(
	ctx context.Context,
	rctx *model.RequestContext,
	instanceID string,
	event string,
	input map[string]any,
) (model.WorkflowInstance, error) {
	// 1. Load instance.
	inst, err := e.store.Get(ctx, rctx.TenantID, instanceID)
	if err != nil {
		return model.WorkflowInstance{}, err
	}

	// 2. Tenant isolation already checked by store.Get.

	// 3. Verify status.
	if inst.Status != model.WorkflowStatusActive {
		return model.WorkflowInstance{}, model.NewWorkflowNotActiveError(
			fmt.Sprintf("workflow instance %q is %s, not active", instanceID, inst.Status),
		)
	}

	// 4. Look up workflow and step definitions.
	wfDef, ok := e.registry.GetWorkflow(inst.WorkflowID)
	if !ok {
		return model.WorkflowInstance{}, model.NewNotFoundError(
			fmt.Sprintf("workflow definition %q not found", inst.WorkflowID),
		)
	}

	stepDef := findStep(wfDef, inst.CurrentStep)
	if stepDef == nil {
		return model.WorkflowInstance{}, model.NewNotFoundError(
			fmt.Sprintf("step %q not found in workflow %q", inst.CurrentStep, inst.WorkflowID),
		)
	}

	// 5. Check step-level capabilities.
	if len(stepDef.Capabilities) > 0 {
		caps, err := e.capResolver.Resolve(rctx)
		if err != nil {
			return model.WorkflowInstance{}, fmt.Errorf("resolve capabilities: %w", err)
		}
		if !caps.HasAll(stepDef.Capabilities...) {
			return model.WorkflowInstance{}, model.NewForbiddenError(
				fmt.Sprintf("insufficient capabilities for step %q", stepDef.ID),
			)
		}
	}

	// 6. Find matching transition.
	transition := findTransition(wfDef, inst.CurrentStep, event, inst.State)
	if transition == nil {
		return model.WorkflowInstance{}, model.NewInvalidTransitionError(
			fmt.Sprintf("no valid transition from step %q with event %q", inst.CurrentStep, event),
		)
	}

	// 6b. Check transition guard capability.
	if transition.Guard != "" {
		caps, err := e.capResolver.Resolve(rctx)
		if err != nil {
			return model.WorkflowInstance{}, fmt.Errorf("resolve capabilities: %w", err)
		}
		if !caps.Has(transition.Guard) {
			return model.WorkflowInstance{}, model.NewForbiddenError(
				fmt.Sprintf("insufficient capability %q for transition guard", transition.Guard),
			)
		}
	}

	// 7. Merge input into workflow state.
	if inst.State == nil {
		inst.State = make(map[string]any)
	}
	for k, v := range input {
		inst.State[k] = v
	}

	// 8. Append completion and event audit records.
	if err := e.appendEvent(ctx, inst.ID, inst.CurrentStep, "step_completed", rctx.SubjectID, nil, ""); err != nil {
		return model.WorkflowInstance{}, err
	}
	if err := e.appendEvent(ctx, inst.ID, inst.CurrentStep, event, rctx.SubjectID, nil, ""); err != nil {
		return model.WorkflowInstance{}, err
	}

	// 9. Transition to next step.
	inst.CurrentStep = transition.To
	inst.UpdatedAt = time.Now().UTC()

	// 10. Append "step_entered" for new step.
	if err := e.appendEvent(ctx, inst.ID, transition.To, "step_entered", "system", nil, ""); err != nil {
		return model.WorkflowInstance{}, err
	}

	// 11. Check new step type.
	nextStep := findStep(wfDef, transition.To)
	if nextStep != nil && nextStep.Type == "terminal" {
		inst.Status = model.WorkflowStatusCompleted
		if err := e.appendEvent(ctx, inst.ID, transition.To, "workflow_completed", "system", nil, ""); err != nil {
			return model.WorkflowInstance{}, err
		}
	}

	// 12. Persist with optimistic locking.
	if err := e.store.Update(ctx, inst); err != nil {
		return model.WorkflowInstance{}, err
	}

	// 13. If next step is auto-executable, run the chain.
	if nextStep != nil && isAutoStep(nextStep.Type) && inst.Status == model.WorkflowStatusActive {
		// Reload after Update (version incremented by store).
		inst, err = e.store.Get(ctx, rctx.TenantID, inst.ID)
		if err != nil {
			return model.WorkflowInstance{}, err
		}
		inst, err = e.executeStepChain(ctx, rctx, inst, wfDef, 0)
		if err != nil {
			return model.WorkflowInstance{}, err
		}
	}

	return inst, nil
}

// Get returns the workflow descriptor for the frontend.
func (e *Engine) Get(
	ctx context.Context,
	rctx *model.RequestContext,
	instanceID string,
) (model.WorkflowDescriptor, error) {
	inst, err := e.store.Get(ctx, rctx.TenantID, instanceID)
	if err != nil {
		return model.WorkflowDescriptor{}, err
	}

	wfDef, ok := e.registry.GetWorkflow(inst.WorkflowID)
	if !ok {
		return model.WorkflowDescriptor{}, model.NewNotFoundError(
			fmt.Sprintf("workflow definition %q not found", inst.WorkflowID),
		)
	}

	// Resolve capabilities for filtering.
	var caps model.CapabilitySet
	if e.capResolver != nil {
		caps, _ = e.capResolver.Resolve(rctx)
	}

	// Build step summaries.
	steps := make([]model.StepSummary, 0, len(wfDef.Steps))
	for _, s := range wfDef.Steps {
		// Filter out steps the user can't see (if they have capabilities).
		if len(s.Capabilities) > 0 && caps != nil && !caps.HasAny(s.Capabilities...) {
			continue
		}
		status := computeStepStatus(s.ID, inst)
		steps = append(steps, model.StepSummary{
			ID:     s.ID,
			Name:   s.Name,
			Type:   s.Type,
			Status: status,
		})
	}

	// Build current step descriptor.
	var currentStep *model.StepDescriptor
	stepDef := findStep(wfDef, inst.CurrentStep)
	if stepDef != nil {
		currentStep = &model.StepDescriptor{
			ID:     stepDef.ID,
			Name:   stepDef.Name,
			Type:   stepDef.Type,
			Status: model.StepStatusInProgress,
		}
	}

	// Build history from events.
	events, _ := e.store.GetEvents(ctx, rctx.TenantID, instanceID)
	history := make([]model.HistoryEntry, 0, len(events))
	for _, evt := range events {
		stepName := evt.StepID
		for _, s := range wfDef.Steps {
			if s.ID == evt.StepID {
				stepName = s.Name
				break
			}
		}
		history = append(history, model.HistoryEntry{
			StepName:  stepName,
			Event:     evt.Event,
			Actor:     evt.ActorID,
			Timestamp: evt.Timestamp.Format(time.RFC3339),
			Comment:   evt.Comment,
		})
	}

	return model.WorkflowDescriptor{
		ID:          inst.ID,
		WorkflowID:  inst.WorkflowID,
		Name:        wfDef.Name,
		Status:      inst.Status,
		CurrentStep: currentStep,
		Steps:       steps,
		History:     history,
	}, nil
}

// Cancel cancels an active workflow instance.
func (e *Engine) Cancel(
	ctx context.Context,
	rctx *model.RequestContext,
	instanceID string,
	reason string,
) error {
	inst, err := e.store.Get(ctx, rctx.TenantID, instanceID)
	if err != nil {
		return err
	}

	if inst.Status != model.WorkflowStatusActive && inst.Status != model.WorkflowStatusSuspended {
		return model.NewWorkflowNotActiveError(
			fmt.Sprintf("workflow instance %q is %s, cannot cancel", instanceID, inst.Status),
		)
	}

	inst.Status = model.WorkflowStatusCancelled
	inst.UpdatedAt = time.Now().UTC()

	if err := e.appendEvent(ctx, inst.ID, inst.CurrentStep, "cancelled", rctx.SubjectID, nil, reason); err != nil {
		return err
	}

	return e.store.Update(ctx, inst)
}

// List returns workflow summaries for the current tenant.
func (e *Engine) List(
	ctx context.Context,
	rctx *model.RequestContext,
	filters model.WorkflowFilters,
) ([]model.WorkflowSummary, int, error) {
	storeFilters := WorkflowFilters{
		WorkflowID: filters.WorkflowID,
		Status:     filters.Status,
		Limit:      filters.PageSize,
		Offset:     (filters.Page - 1) * filters.PageSize,
	}
	if storeFilters.Limit <= 0 {
		storeFilters.Limit = 20
	}
	if storeFilters.Offset < 0 {
		storeFilters.Offset = 0
	}

	instances, err := e.store.FindActive(ctx, rctx.TenantID, storeFilters)
	if err != nil {
		return nil, 0, err
	}

	// Get total count (FindActive with no pagination).
	allFilters := WorkflowFilters{
		WorkflowID: filters.WorkflowID,
		Status:     filters.Status,
	}
	all, err := e.store.FindActive(ctx, rctx.TenantID, allFilters)
	if err != nil {
		return nil, 0, err
	}
	totalCount := len(all)

	summaries := make([]model.WorkflowSummary, 0, len(instances))
	for _, inst := range instances {
		name := inst.WorkflowID
		if wfDef, ok := e.registry.GetWorkflow(inst.WorkflowID); ok {
			name = wfDef.Name
		}
		summaries = append(summaries, model.WorkflowSummary{
			ID:          inst.ID,
			WorkflowID:  inst.WorkflowID,
			Name:        name,
			CurrentStep: inst.CurrentStep,
			Status:      inst.Status,
			SubjectID:   inst.SubjectID,
			CreatedAt:   inst.CreatedAt,
			UpdatedAt:   inst.UpdatedAt,
		})
	}

	return summaries, totalCount, nil
}

// ProcessTimeouts finds expired workflow instances and processes their timeouts.
func (e *Engine) ProcessTimeouts(ctx context.Context) error {
	now := time.Now().UTC()
	expired, err := e.store.FindExpired(ctx, now)
	if err != nil {
		return fmt.Errorf("find expired workflows: %w", err)
	}

	for _, inst := range expired {
		if err := e.processTimeout(ctx, inst); err != nil {
			// Log and continue processing other instances.
			continue
		}
	}
	return nil
}

// processTimeout handles a single expired workflow instance.
func (e *Engine) processTimeout(ctx context.Context, inst model.WorkflowInstance) error {
	wfDef, ok := e.registry.GetWorkflow(inst.WorkflowID)
	if !ok {
		return fmt.Errorf("workflow definition %q not found", inst.WorkflowID)
	}

	// Check step-level on_timeout first.
	stepDef := findStep(wfDef, inst.CurrentStep)
	var targetStep string
	if stepDef != nil && stepDef.OnTimeout != "" {
		targetStep = stepDef.OnTimeout
	} else if wfDef.OnTimeout != "" {
		targetStep = wfDef.OnTimeout
	}

	if err := e.appendEvent(ctx, inst.ID, inst.CurrentStep, "timeout", "system", nil, ""); err != nil {
		return err
	}

	if targetStep != "" {
		// Transition to timeout step.
		inst.CurrentStep = targetStep
		inst.UpdatedAt = time.Now().UTC()

		if err := e.appendEvent(ctx, inst.ID, targetStep, "step_entered", "system", nil, ""); err != nil {
			return err
		}

		// Check if timeout target is terminal.
		target := findStep(wfDef, targetStep)
		if target != nil && target.Type == "terminal" {
			inst.Status = model.WorkflowStatusCompleted
			if err := e.appendEvent(ctx, inst.ID, targetStep, "workflow_completed", "system", nil, ""); err != nil {
				return err
			}
		}
	} else {
		// No timeout handler: fail the workflow.
		inst.Status = model.WorkflowStatusFailed
		if err := e.appendEvent(ctx, inst.ID, inst.CurrentStep, "workflow_failed", "system", nil, "timeout with no handler"); err != nil {
			return err
		}
	}

	return e.store.Update(ctx, inst)
}

// executeStepChain executes system/notification steps in a chain until
// a human step, terminal state, or chain limit is reached.
func (e *Engine) executeStepChain(
	ctx context.Context,
	rctx *model.RequestContext,
	inst model.WorkflowInstance,
	wfDef model.WorkflowDefinition,
	depth int,
) (model.WorkflowInstance, error) {
	if depth >= e.chainLimit {
		inst.Status = model.WorkflowStatusSuspended
		inst.UpdatedAt = time.Now().UTC()
		_ = e.appendEvent(ctx, inst.ID, inst.CurrentStep, "workflow_suspended", "system", nil, "chain limit reached")
		if err := e.store.Update(ctx, inst); err != nil {
			return inst, err
		}
		return inst, model.NewWorkflowChainLimitError()
	}

	stepDef := findStep(wfDef, inst.CurrentStep)
	if stepDef == nil {
		return inst, fmt.Errorf("step %q not found in workflow", inst.CurrentStep)
	}

	// Execute the system step.
	result, execErr := e.executeSystemStep(ctx, rctx, inst, stepDef)

	if execErr != nil {
		// Handle notification type: non-blocking, treat as completed.
		if stepDef.Type == "notification" {
			// Record error but continue.
			inst.State["_last_error"] = execErr.Error()
			_ = e.appendEvent(ctx, inst.ID, stepDef.ID, "step_failed", "system",
				map[string]any{"error": execErr.Error()}, "")
			_ = e.appendEvent(ctx, inst.ID, stepDef.ID, "step_completed", "system", nil, "notification best-effort")
		} else {
			// Record error in state.
			inst.State["_last_error"] = execErr.Error()
			_ = e.appendEvent(ctx, inst.ID, stepDef.ID, "step_failed", "system",
				map[string]any{"error": execErr.Error()}, "")

			// Try error transition.
			errorTransition := findTransition(wfDef, inst.CurrentStep, "error", inst.State)
			if errorTransition != nil {
				inst.CurrentStep = errorTransition.To
				inst.UpdatedAt = time.Now().UTC()
				_ = e.appendEvent(ctx, inst.ID, errorTransition.To, "step_entered", "system", nil, "")

				target := findStep(wfDef, errorTransition.To)
				if target != nil && target.Type == "terminal" {
					inst.Status = model.WorkflowStatusCompleted
					_ = e.appendEvent(ctx, inst.ID, errorTransition.To, "workflow_completed", "system", nil, "")
				}
				if err := e.store.Update(ctx, inst); err != nil {
					return inst, err
				}
				// Continue chain if error target is auto-executable.
				if target != nil && isAutoStep(target.Type) && inst.Status == model.WorkflowStatusActive {
					inst, _ = e.store.Get(ctx, rctx.TenantID, inst.ID)
					return e.executeStepChain(ctx, rctx, inst, wfDef, depth+1)
				}
				return inst, nil
			}

			// No error transition: suspend.
			inst.Status = model.WorkflowStatusSuspended
			inst.UpdatedAt = time.Now().UTC()
			_ = e.appendEvent(ctx, inst.ID, inst.CurrentStep, "workflow_suspended", "system", nil, "system step failed, no error transition")
			if err := e.store.Update(ctx, inst); err != nil {
				return inst, err
			}
			return inst, nil
		}
	} else {
		// Success: merge result into state and record completion.
		if result != nil {
			for k, v := range result {
				inst.State[k] = v
			}
		}
		_ = e.appendEvent(ctx, inst.ID, stepDef.ID, "step_completed", "system", nil, "")
	}

	// Find "completed" transition to next step.
	completedTransition := findTransition(wfDef, inst.CurrentStep, "completed", inst.State)
	if completedTransition == nil {
		// No completed transition: workflow is misconfigured, fail it.
		inst.Status = model.WorkflowStatusFailed
		inst.UpdatedAt = time.Now().UTC()
		_ = e.appendEvent(ctx, inst.ID, inst.CurrentStep, "workflow_failed", "system", nil, "no completed transition")
		if err := e.store.Update(ctx, inst); err != nil {
			return inst, err
		}
		return inst, nil
	}

	// Transition to next step.
	inst.CurrentStep = completedTransition.To
	inst.UpdatedAt = time.Now().UTC()
	_ = e.appendEvent(ctx, inst.ID, completedTransition.To, "step_entered", "system", nil, "")

	nextStep := findStep(wfDef, completedTransition.To)
	if nextStep != nil && nextStep.Type == "terminal" {
		inst.Status = model.WorkflowStatusCompleted
		_ = e.appendEvent(ctx, inst.ID, completedTransition.To, "workflow_completed", "system", nil, "")
	}

	// Persist.
	if err := e.store.Update(ctx, inst); err != nil {
		return inst, err
	}

	// Continue chain if next step is auto-executable.
	if nextStep != nil && isAutoStep(nextStep.Type) && inst.Status == model.WorkflowStatusActive {
		inst, _ = e.store.Get(ctx, rctx.TenantID, inst.ID)
		return e.executeStepChain(ctx, rctx, inst, wfDef, depth+1)
	}

	return inst, nil
}

// executeSystemStep invokes the backend operation for a system step and
// applies output mapping. Returns the merged state additions.
func (e *Engine) executeSystemStep(
	ctx context.Context,
	rctx *model.RequestContext,
	inst model.WorkflowInstance,
	stepDef *model.StepDefinition,
) (map[string]any, error) {
	if stepDef.Operation == nil {
		return nil, nil // No operation to invoke.
	}

	// Build invocation input from step input mapping.
	var invInput model.InvocationInput
	if stepDef.Input != nil {
		mapper := command.NewInputMapper()
		cmdInput := model.CommandInput{
			Input: inst.State,
		}
		var err error
		invInput, err = mapper.MapInput(*stepDef.Input, cmdInput, rctx, inst.State)
		if err != nil {
			return nil, fmt.Errorf("map input for step %q: %w", stepDef.ID, err)
		}
	}

	// Invoke backend.
	result, err := e.invokers.Invoke(ctx, rctx, *stepDef.Operation, invInput)
	if err != nil {
		return nil, fmt.Errorf("invoke step %q: %w", stepDef.ID, err)
	}

	if result.StatusCode < 200 || result.StatusCode >= 300 {
		return nil, fmt.Errorf("step %q returned status %d", stepDef.ID, result.StatusCode)
	}

	// Apply output mapping.
	return applyOutputMapping(stepDef.Output, result), nil
}

// applyOutputMapping extracts fields from the invocation result based on the
// output mapping definition.
func applyOutputMapping(output *model.OutputMapping, result model.InvocationResult) map[string]any {
	if output == nil || len(output.Fields) == 0 {
		// If no mapping, try to merge the body directly.
		if bodyMap, ok := result.Body.(map[string]any); ok {
			return bodyMap
		}
		return nil
	}

	bodyMap, ok := result.Body.(map[string]any)
	if !ok {
		return nil
	}

	merged := make(map[string]any, len(output.Fields))
	for stateKey, bodyPath := range output.Fields {
		if val, exists := bodyMap[bodyPath]; exists {
			merged[stateKey] = val
		}
	}
	return merged
}

// appendEvent is a convenience helper for creating and persisting events.
func (e *Engine) appendEvent(
	ctx context.Context,
	instanceID, stepID, event, actorID string,
	data map[string]any,
	comment string,
) error {
	return e.store.AppendEvent(ctx, model.WorkflowEvent{
		ID:                 uuid.New().String(),
		WorkflowInstanceID: instanceID,
		StepID:             stepID,
		Event:              event,
		ActorID:            actorID,
		Data:               data,
		Comment:            comment,
		Timestamp:          time.Now().UTC(),
	})
}

// findStep looks up a step definition by ID within a workflow.
func findStep(wfDef model.WorkflowDefinition, stepID string) *model.StepDefinition {
	for i := range wfDef.Steps {
		if wfDef.Steps[i].ID == stepID {
			return &wfDef.Steps[i]
		}
	}
	return nil
}

// findTransition finds the first matching transition from a step with the given event.
func findTransition(wfDef model.WorkflowDefinition, fromStep, event string, state map[string]any) *model.TransitionDefinition {
	for i := range wfDef.Transitions {
		t := &wfDef.Transitions[i]
		if t.From != fromStep || t.Event != event {
			continue
		}
		// Evaluate condition if present.
		if t.Condition != "" && !evaluateCondition(t.Condition, state) {
			continue
		}
		return t
	}
	return nil
}

// evaluateCondition evaluates a simple condition expression against workflow state.
// Supports: "field == 'value'" and "field != 'value'".
func evaluateCondition(condition string, state map[string]any) bool {
	// Simple equality check: "field == 'value'"
	if parts := splitCondition(condition, "=="); len(parts) == 2 {
		field := trimSpace(parts[0])
		expected := trimQuotes(trimSpace(parts[1]))
		actual := fmt.Sprint(state[field])
		return actual == expected
	}
	// Simple inequality check: "field != 'value'"
	if parts := splitCondition(condition, "!="); len(parts) == 2 {
		field := trimSpace(parts[0])
		expected := trimQuotes(trimSpace(parts[1]))
		actual := fmt.Sprint(state[field])
		return actual != expected
	}
	// If we can't parse the condition, treat as true (permissive).
	return true
}

// splitCondition splits a condition string by an operator, but only if the
// operator isn't part of a longer operator (e.g., != vs ==).
func splitCondition(s, op string) []string {
	idx := -1
	for i := 0; i <= len(s)-len(op); i++ {
		if s[i:i+len(op)] == op {
			// For "==", make sure it's not "!="
			if op == "==" && i > 0 && s[i-1] == '!' {
				continue
			}
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil
	}
	return []string{s[:idx], s[idx+len(op):]}
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func trimQuotes(s string) string {
	if len(s) >= 2 && ((s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"')) {
		return s[1 : len(s)-1]
	}
	return s
}

// isAutoStep returns true if the step type should be automatically executed.
func isAutoStep(stepType string) bool {
	return stepType == "system" || stepType == "notification"
}

// computeStepStatus determines a step's display status based on the workflow state.
func computeStepStatus(stepID string, inst model.WorkflowInstance) string {
	if inst.CurrentStep == stepID {
		if inst.Status == model.WorkflowStatusCompleted || inst.Status == model.WorkflowStatusFailed {
			return inst.Status
		}
		return model.StepStatusInProgress
	}
	// For a simple approximation: steps in the definition before the current
	// step are "completed", steps after are "future". This is a simplified
	// heuristic; a full implementation would track per-step status.
	return model.StepStatusFuture
}
