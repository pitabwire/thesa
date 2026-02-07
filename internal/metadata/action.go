package metadata

import (
	"fmt"

	"github.com/pitabwire/thesa/model"
)

// ActionProvider resolves ActionDefinition lists into ActionDescriptor lists,
// filtering by capabilities and evaluating static conditions.
type ActionProvider struct{}

// NewActionProvider creates a new ActionProvider.
func NewActionProvider() *ActionProvider {
	return &ActionProvider{}
}

// ResolveActions resolves a list of action definitions into descriptors,
// filtering by capabilities. Static conditions are evaluated for
// enabled/visible state, while data-dependent conditions are passed through
// as ConditionDescriptor for client-side evaluation.
func (p *ActionProvider) ResolveActions(
	caps model.CapabilitySet,
	actions []model.ActionDefinition,
	resourceData map[string]any,
) []model.ActionDescriptor {
	var result []model.ActionDescriptor
	for _, action := range actions {
		// Check capabilities — omit if unauthorized.
		if len(action.Capabilities) > 0 && !caps.HasAll(action.Capabilities...) {
			continue
		}

		desc := model.ActionDescriptor{
			ID:         action.ID,
			Label:      action.Label,
			Icon:       action.Icon,
			Style:      action.Style,
			Type:       action.Type,
			Enabled:    true,
			Visible:    true,
			CommandID:  action.CommandID,
			NavigateTo: action.NavigateTo,
			WorkflowID: action.WorkflowID,
			FormID:     action.FormID,
			Params:     action.Params,
		}

		// Resolve confirmation descriptor.
		if action.Confirmation != nil {
			desc.Confirmation = &model.ConfirmationDescriptor{
				Title:   action.Confirmation.Title,
				Message: action.Confirmation.Message,
				Confirm: action.Confirmation.Confirm,
				Cancel:  action.Confirmation.Cancel,
				Style:   action.Confirmation.Style,
			}
		}

		// Evaluate conditions.
		var clientConditions []model.ConditionDescriptor
		for _, cond := range action.Conditions {
			if isStaticCondition(cond, resourceData) {
				// Evaluate static condition and apply its effect.
				met := evaluateStaticCondition(cond, resourceData)
				applyConditionEffect(&desc, cond.Effect, met)
			} else {
				// Pass data-dependent conditions through for client-side evaluation.
				clientConditions = append(clientConditions, model.ConditionDescriptor{
					Field:    cond.Field,
					Operator: cond.Operator,
					Value:    cond.Value,
					Effect:   cond.Effect,
				})
			}
		}
		if len(clientConditions) > 0 {
			desc.Conditions = clientConditions
		}

		result = append(result, desc)
	}

	if result == nil {
		result = []model.ActionDescriptor{}
	}

	return result
}

// isStaticCondition returns true if the condition can be evaluated server-side
// with the available resource data. A condition is static if the resource data
// contains the required field.
func isStaticCondition(cond model.ConditionDefinition, data map[string]any) bool {
	if data == nil {
		return false
	}
	_, exists := data[cond.Field]
	return exists
}

// evaluateStaticCondition evaluates a condition against resource data.
func evaluateStaticCondition(cond model.ConditionDefinition, data map[string]any) bool {
	if data == nil {
		return false
	}
	fieldVal, exists := data[cond.Field]
	if !exists {
		return false
	}

	condValue := cond.Value

	switch cond.Operator {
	case "eq", "equals", "==":
		return fmt.Sprint(fieldVal) == fmt.Sprint(condValue)
	case "neq", "not_equals", "!=":
		return fmt.Sprint(fieldVal) != fmt.Sprint(condValue)
	case "in":
		return valueInSlice(fieldVal, condValue)
	case "not_in":
		return !valueInSlice(fieldVal, condValue)
	case "exists":
		return exists
	case "not_exists":
		return !exists
	default:
		return false
	}
}

// applyConditionEffect applies the result of a condition evaluation to the descriptor.
func applyConditionEffect(desc *model.ActionDescriptor, effect string, conditionMet bool) {
	switch effect {
	case "hide":
		if conditionMet {
			desc.Visible = false
		}
	case "show":
		if !conditionMet {
			desc.Visible = false
		}
	case "disable":
		if conditionMet {
			desc.Enabled = false
		}
	case "enable":
		if !conditionMet {
			desc.Enabled = false
		}
	}
}

// valueInSlice checks if fieldVal matches any value in condValue (expected to be a slice-like value).
func valueInSlice(fieldVal, condValue any) bool {
	// condValue could be a comma-separated string or a slice.
	fieldStr := fmt.Sprint(fieldVal)

	switch cv := condValue.(type) {
	case []any:
		for _, v := range cv {
			if fmt.Sprint(v) == fieldStr {
				return true
			}
		}
	case string:
		// Treat as comma-separated list.
		for _, v := range splitComma(cv) {
			if v == fieldStr {
				return true
			}
		}
	}
	return false
}

// splitComma splits a string by commas and trims whitespace.
func splitComma(s string) []string {
	if s == "" {
		return nil
	}
	var parts []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := trimSpace(s[start:i])
			if part != "" {
				parts = append(parts, part)
			}
			start = i + 1
		}
	}
	return parts
}

// trimSpace trims leading and trailing spaces.
func trimSpace(s string) string {
	i := 0
	for i < len(s) && s[i] == ' ' {
		i++
	}
	j := len(s)
	for j > i && s[j-1] == ' ' {
		j--
	}
	return s[i:j]
}
