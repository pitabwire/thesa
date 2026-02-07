package metadata

import (
	"testing"

	"github.com/pitabwire/thesa/model"
)

func TestActionProvider_ResolveActions_basic(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{ID: "create", Label: "Create", Type: "command", CommandID: "create-user"},
		{ID: "view", Label: "View", Type: "navigate", NavigateTo: "/users/{id}"},
	}

	result := ap.ResolveActions(model.CapabilitySet{"*": true}, actions, nil)
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	if result[0].ID != "create" {
		t.Errorf("result[0].ID = %q, want create", result[0].ID)
	}
	if result[0].CommandID != "create-user" {
		t.Errorf("result[0].CommandID = %q, want create-user", result[0].CommandID)
	}
	if result[1].NavigateTo != "/users/{id}" {
		t.Errorf("result[1].NavigateTo = %q", result[1].NavigateTo)
	}
}

func TestActionProvider_ResolveActions_filtersUncapable(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{ID: "admin-only", Label: "Admin", Type: "command", Capabilities: []string{"admin:access"}},
		{ID: "public", Label: "Public", Type: "navigate"},
	}

	caps := model.CapabilitySet{"users:view": true}
	result := ap.ResolveActions(caps, actions, nil)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1 (admin filtered)", len(result))
	}
	if result[0].ID != "public" {
		t.Errorf("result[0].ID = %q, want public", result[0].ID)
	}
}

func TestActionProvider_ResolveActions_requiresAllCapabilities(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{ID: "multi-cap", Label: "Multi", Type: "command", Capabilities: []string{"cap1", "cap2"}},
	}

	// Has only one of the two required.
	caps := model.CapabilitySet{"cap1": true}
	result := ap.ResolveActions(caps, actions, nil)

	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0 (missing cap2)", len(result))
	}
}

func TestActionProvider_ResolveActions_enabledAndVisibleByDefault(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{ID: "action", Label: "Action", Type: "command"},
	}

	result := ap.ResolveActions(model.CapabilitySet{}, actions, nil)
	if !result[0].Enabled {
		t.Error("Enabled = false, want true by default")
	}
	if !result[0].Visible {
		t.Error("Visible = false, want true by default")
	}
}

func TestActionProvider_ResolveActions_confirmation(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{
			ID:    "delete",
			Label: "Delete",
			Type:  "command",
			Confirmation: &model.ConfirmationDefinition{
				Title:   "Confirm Delete",
				Message: "Are you sure?",
				Confirm: "Yes, delete",
				Cancel:  "Cancel",
				Style:   "danger",
			},
		},
	}

	result := ap.ResolveActions(model.CapabilitySet{}, actions, nil)
	if result[0].Confirmation == nil {
		t.Fatal("Confirmation is nil")
	}
	if result[0].Confirmation.Title != "Confirm Delete" {
		t.Errorf("Confirmation.Title = %q", result[0].Confirmation.Title)
	}
	if result[0].Confirmation.Style != "danger" {
		t.Errorf("Confirmation.Style = %q", result[0].Confirmation.Style)
	}
}

func TestActionProvider_ResolveActions_noConfirmation(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{ID: "simple", Label: "Simple", Type: "command"},
	}

	result := ap.ResolveActions(model.CapabilitySet{}, actions, nil)
	if result[0].Confirmation != nil {
		t.Errorf("Confirmation = %+v, want nil", result[0].Confirmation)
	}
}

func TestActionProvider_ResolveActions_staticCondition_hide(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{
			ID:    "cancel",
			Label: "Cancel",
			Type:  "command",
			Conditions: []model.ConditionDefinition{
				{Field: "status", Operator: "eq", Value: "completed", Effect: "hide"},
			},
		},
	}

	// Status is "completed" → condition met → hide.
	data := map[string]any{"status": "completed"}
	result := ap.ResolveActions(model.CapabilitySet{}, actions, data)

	if result[0].Visible {
		t.Error("Visible = true, want false (status == completed → hide)")
	}
}

func TestActionProvider_ResolveActions_staticCondition_hideNotMet(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{
			ID:    "cancel",
			Label: "Cancel",
			Type:  "command",
			Conditions: []model.ConditionDefinition{
				{Field: "status", Operator: "eq", Value: "completed", Effect: "hide"},
			},
		},
	}

	// Status is "active" → condition not met → visible.
	data := map[string]any{"status": "active"}
	result := ap.ResolveActions(model.CapabilitySet{}, actions, data)

	if !result[0].Visible {
		t.Error("Visible = false, want true (status != completed)")
	}
}

func TestActionProvider_ResolveActions_staticCondition_show(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{
			ID:    "approve",
			Label: "Approve",
			Type:  "command",
			Conditions: []model.ConditionDefinition{
				{Field: "status", Operator: "eq", Value: "pending", Effect: "show"},
			},
		},
	}

	// status=pending → condition met → show stays true.
	data := map[string]any{"status": "pending"}
	result := ap.ResolveActions(model.CapabilitySet{}, actions, data)
	if !result[0].Visible {
		t.Error("Visible = false, want true (show condition met)")
	}

	// status=active → condition not met → show makes it invisible.
	data = map[string]any{"status": "active"}
	result = ap.ResolveActions(model.CapabilitySet{}, actions, data)
	if result[0].Visible {
		t.Error("Visible = true, want false (show condition not met)")
	}
}

func TestActionProvider_ResolveActions_staticCondition_disable(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{
			ID:    "edit",
			Label: "Edit",
			Type:  "command",
			Conditions: []model.ConditionDefinition{
				{Field: "locked", Operator: "eq", Value: "true", Effect: "disable"},
			},
		},
	}

	data := map[string]any{"locked": "true"}
	result := ap.ResolveActions(model.CapabilitySet{}, actions, data)
	if result[0].Enabled {
		t.Error("Enabled = true, want false (locked → disable)")
	}
}

func TestActionProvider_ResolveActions_staticCondition_enable(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{
			ID:    "submit",
			Label: "Submit",
			Type:  "command",
			Conditions: []model.ConditionDefinition{
				{Field: "complete", Operator: "eq", Value: "true", Effect: "enable"},
			},
		},
	}

	// complete=false → condition not met → enable makes it disabled.
	data := map[string]any{"complete": "false"}
	result := ap.ResolveActions(model.CapabilitySet{}, actions, data)
	if result[0].Enabled {
		t.Error("Enabled = true, want false (enable condition not met)")
	}

	// complete=true → condition met → stays enabled.
	data = map[string]any{"complete": "true"}
	result = ap.ResolveActions(model.CapabilitySet{}, actions, data)
	if !result[0].Enabled {
		t.Error("Enabled = false, want true (enable condition met)")
	}
}

func TestActionProvider_ResolveActions_staticCondition_neq(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{
			ID:    "cancel",
			Label: "Cancel",
			Type:  "command",
			Conditions: []model.ConditionDefinition{
				{Field: "status", Operator: "neq", Value: "active", Effect: "hide"},
			},
		},
	}

	// status != active → hide.
	data := map[string]any{"status": "completed"}
	result := ap.ResolveActions(model.CapabilitySet{}, actions, data)
	if result[0].Visible {
		t.Error("Visible = true, want false (neq met)")
	}

	// status == active → neq not met → visible.
	data = map[string]any{"status": "active"}
	result = ap.ResolveActions(model.CapabilitySet{}, actions, data)
	if !result[0].Visible {
		t.Error("Visible = false, want true (neq not met)")
	}
}

func TestActionProvider_ResolveActions_staticCondition_in(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{
			ID:    "reopen",
			Label: "Reopen",
			Type:  "command",
			Conditions: []model.ConditionDefinition{
				{Field: "status", Operator: "in", Value: "cancelled,completed", Effect: "show"},
			},
		},
	}

	data := map[string]any{"status": "cancelled"}
	result := ap.ResolveActions(model.CapabilitySet{}, actions, data)
	if !result[0].Visible {
		t.Error("Visible = false, want true (status in [cancelled,completed])")
	}

	data = map[string]any{"status": "active"}
	result = ap.ResolveActions(model.CapabilitySet{}, actions, data)
	if result[0].Visible {
		t.Error("Visible = true, want false (status not in list)")
	}
}

func TestActionProvider_ResolveActions_dataDependentConditionPassedThrough(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{
			ID:    "action",
			Label: "Action",
			Type:  "command",
			Conditions: []model.ConditionDefinition{
				{Field: "amount", Operator: "gt", Value: "100", Effect: "disable"},
			},
		},
	}

	// Field "amount" not in resource data → treated as client-side condition.
	result := ap.ResolveActions(model.CapabilitySet{}, actions, nil)
	if len(result[0].Conditions) != 1 {
		t.Fatalf("len(Conditions) = %d, want 1", len(result[0].Conditions))
	}
	cond := result[0].Conditions[0]
	if cond.Field != "amount" || cond.Operator != "gt" || cond.Effect != "disable" {
		t.Errorf("Condition = %+v, unexpected", cond)
	}
}

func TestActionProvider_ResolveActions_mixedConditions(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{
			ID:    "action",
			Label: "Action",
			Type:  "command",
			Conditions: []model.ConditionDefinition{
				// Static (field in data).
				{Field: "status", Operator: "eq", Value: "active", Effect: "show"},
				// Client-side (field not in data).
				{Field: "priority", Operator: "gt", Value: "5", Effect: "disable"},
			},
		},
	}

	data := map[string]any{"status": "active"}
	result := ap.ResolveActions(model.CapabilitySet{}, actions, data)

	// Static condition met → visible stays true.
	if !result[0].Visible {
		t.Error("Visible = false, want true")
	}
	// Client-side condition passed through.
	if len(result[0].Conditions) != 1 {
		t.Fatalf("len(Conditions) = %d, want 1", len(result[0].Conditions))
	}
	if result[0].Conditions[0].Field != "priority" {
		t.Errorf("Conditions[0].Field = %q, want priority", result[0].Conditions[0].Field)
	}
}

func TestActionProvider_ResolveActions_allActionTypes(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{ID: "cmd", Label: "Command", Type: "command", CommandID: "do-thing"},
		{ID: "nav", Label: "Navigate", Type: "navigate", NavigateTo: "/path"},
		{ID: "wf", Label: "Workflow", Type: "workflow", WorkflowID: "wf-1"},
		{ID: "frm", Label: "Form", Type: "modal", FormID: "form-1"},
		{ID: "lnk", Label: "Link", Type: "link", NavigateTo: "https://example.com"},
	}

	result := ap.ResolveActions(model.CapabilitySet{}, actions, nil)
	if len(result) != 5 {
		t.Fatalf("len(result) = %d, want 5", len(result))
	}

	// Verify each type is preserved.
	types := map[string]bool{}
	for _, a := range result {
		types[a.Type] = true
	}
	for _, want := range []string{"command", "navigate", "workflow", "modal", "link"} {
		if !types[want] {
			t.Errorf("type %q not found in result", want)
		}
	}
}

func TestActionProvider_ResolveActions_params(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{
			ID:    "view",
			Label: "View",
			Type:  "navigate",
			Params: map[string]string{
				"id":   "{row.id}",
				"mode": "detail",
			},
		},
	}

	result := ap.ResolveActions(model.CapabilitySet{}, actions, nil)
	if result[0].Params["id"] != "{row.id}" {
		t.Errorf("Params[id] = %q, want {row.id}", result[0].Params["id"])
	}
	if result[0].Params["mode"] != "detail" {
		t.Errorf("Params[mode] = %q, want detail", result[0].Params["mode"])
	}
}

func TestActionProvider_ResolveActions_emptyActions(t *testing.T) {
	ap := NewActionProvider()

	result := ap.ResolveActions(model.CapabilitySet{}, nil, nil)
	if result == nil {
		t.Fatal("result is nil, want empty slice")
	}
	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}

func TestActionProvider_ResolveActions_styleAndIcon(t *testing.T) {
	ap := NewActionProvider()

	actions := []model.ActionDefinition{
		{
			ID:    "delete",
			Label: "Delete",
			Type:  "command",
			Icon:  "trash",
			Style: "danger",
		},
	}

	result := ap.ResolveActions(model.CapabilitySet{}, actions, nil)
	if result[0].Icon != "trash" {
		t.Errorf("Icon = %q, want trash", result[0].Icon)
	}
	if result[0].Style != "danger" {
		t.Errorf("Style = %q, want danger", result[0].Style)
	}
}

// --- splitComma and trimSpace ---

func TestSplitComma(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"a, b , c", []string{"a", "b", "c"}},
		{"single", []string{"single"}},
		{"", nil},
	}
	for _, tt := range tests {
		got := splitComma(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("splitComma(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i, v := range got {
			if v != tt.want[i] {
				t.Errorf("splitComma(%q)[%d] = %q, want %q", tt.input, i, v, tt.want[i])
			}
		}
	}
}
