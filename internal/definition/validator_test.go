package definition

import (
	"testing"

	"github.com/pitabwire/thesa/internal/openapi"
	"github.com/pitabwire/thesa/model"
)

func validDomain() model.DomainDefinition {
	return model.DomainDefinition{
		Domain:  "orders",
		Version: "1.0.0",
		Navigation: model.NavigationDefinition{
			Label: "Orders",
			Icon:  "shopping_cart",
			Order: 10,
			Children: []model.NavigationChildDefinition{
				{Label: "All Orders", Route: "/orders", PageID: "orders.list", Order: 1},
			},
		},
		Pages: []model.PageDefinition{
			{
				ID:     "orders.list",
				Title:  "Orders",
				Route:  "/orders",
				Layout: "list",
				Capabilities: []string{"orders:list:view"},
				Table: &model.TableDefinition{
					DataSource: model.DataSourceDefinition{
						OperationID: "listOrders",
						ServiceID:   "orders-svc",
						Mapping:     model.ResponseMappingDefinition{ItemsPath: "data.orders"},
					},
					Columns: []model.ColumnDefinition{
						{Field: "id", Label: "ID", Type: "text"},
					},
					PageSize: 25,
				},
			},
		},
		Commands: []model.CommandDefinition{
			{
				ID:        "orders.update",
				Operation: model.OperationBinding{Type: "openapi", OperationID: "updateOrder", ServiceID: "orders-svc"},
				Input:     model.InputMapping{BodyMapping: "passthrough"},
				Output:    model.OutputMapping{Type: "passthrough"},
			},
		},
		Forms: []model.FormDefinition{
			{
				ID:            "orders.edit_form",
				Title:         "Edit Order",
				SubmitCommand: "orders.update",
				Sections: []model.SectionDefinition{
					{ID: "header", Title: "Info", Layout: "grid", Fields: []model.FieldDefinition{{Field: "name", Label: "Name", Type: "text"}}},
				},
			},
		},
	}
}

func loadTestOAPIIndex(t *testing.T) *openapi.Index {
	t.Helper()
	idx := openapi.NewIndex()
	err := idx.Load([]openapi.SpecSource{
		{ServiceID: "orders-svc", SpecPath: "../openapi/testdata/orders-svc.yaml"},
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	return idx
}

func TestValidator_valid(t *testing.T) {
	v := NewValidator()
	idx := loadTestOAPIIndex(t)
	errs := v.Validate([]model.DomainDefinition{validDomain()}, idx)
	if len(errs) > 0 {
		for _, e := range errs {
			t.Logf("  %s", e)
		}
		t.Fatalf("Validate() returned %d errors, want 0", len(errs))
	}
}

func TestValidator_missing_domain(t *testing.T) {
	v := NewValidator()
	def := validDomain()
	def.Domain = ""
	errs := v.Validate([]model.DomainDefinition{def}, nil)
	if !hasCode(errs, "REQUIRED") {
		t.Error("expected REQUIRED error for missing domain")
	}
}

func TestValidator_missing_version(t *testing.T) {
	v := NewValidator()
	def := validDomain()
	def.Version = ""
	errs := v.Validate([]model.DomainDefinition{def}, nil)
	if !hasCode(errs, "REQUIRED") {
		t.Error("expected REQUIRED error for missing version")
	}
}

func TestValidator_invalid_layout(t *testing.T) {
	v := NewValidator()
	def := validDomain()
	def.Pages[0].Layout = "unknown"
	errs := v.Validate([]model.DomainDefinition{def}, nil)
	if !hasCode(errs, "INVALID_ENUM") {
		t.Error("expected INVALID_ENUM error for invalid layout")
	}
}

func TestValidator_list_without_table(t *testing.T) {
	v := NewValidator()
	def := validDomain()
	def.Pages[0].Table = nil
	errs := v.Validate([]model.DomainDefinition{def}, nil)
	if !hasCode(errs, "REQUIRED") {
		t.Error("expected REQUIRED error for missing table on list layout")
	}
}

func TestValidator_missing_columns(t *testing.T) {
	v := NewValidator()
	def := validDomain()
	def.Pages[0].Table.Columns = nil
	errs := v.Validate([]model.DomainDefinition{def}, nil)
	if !hasCode(errs, "REQUIRED") {
		t.Error("expected REQUIRED error for missing columns")
	}
}

func TestValidator_invalid_page_size(t *testing.T) {
	v := NewValidator()
	def := validDomain()
	def.Pages[0].Table.PageSize = 300
	errs := v.Validate([]model.DomainDefinition{def}, nil)
	if !hasCode(errs, "RANGE") {
		t.Error("expected RANGE error for page_size > 200")
	}
}

func TestValidator_operation_not_found(t *testing.T) {
	v := NewValidator()
	idx := loadTestOAPIIndex(t)
	def := validDomain()
	def.Pages[0].Table.DataSource.OperationID = "nonexistent"
	errs := v.Validate([]model.DomainDefinition{def}, idx)
	if !hasCode(errs, "OPERATION_NOT_FOUND") {
		t.Error("expected OPERATION_NOT_FOUND error")
	}
}

func TestValidator_command_operation_not_found(t *testing.T) {
	v := NewValidator()
	idx := loadTestOAPIIndex(t)
	def := validDomain()
	def.Commands[0].Operation.OperationID = "nonexistent"
	errs := v.Validate([]model.DomainDefinition{def}, idx)
	if !hasCode(errs, "OPERATION_NOT_FOUND") {
		t.Error("expected OPERATION_NOT_FOUND error for command")
	}
}

func TestValidator_form_missing_command(t *testing.T) {
	v := NewValidator()
	def := validDomain()
	def.Forms[0].SubmitCommand = "nonexistent.command"
	errs := v.Validate([]model.DomainDefinition{def}, nil)
	if !hasCode(errs, "REF_NOT_FOUND") {
		t.Error("expected REF_NOT_FOUND error for form submit_command")
	}
}

func TestValidator_workflow_validation(t *testing.T) {
	v := NewValidator()
	def := validDomain()
	def.Workflows = []model.WorkflowDefinition{
		{
			ID:          "orders.approval",
			Name:        "Order Approval",
			InitialStep: "review",
			Steps: []model.StepDefinition{
				{ID: "review", Name: "Review", Type: "approval"},
				{ID: "done", Name: "Done", Type: "terminal"},
			},
			Transitions: []model.TransitionDefinition{
				{From: "review", To: "done", Event: "approved"},
			},
		},
	}
	errs := v.Validate([]model.DomainDefinition{def}, nil)
	for _, e := range errs {
		if e.Path == "definitions[0].workflows[0]" {
			t.Errorf("unexpected workflow error: %s", e)
		}
	}
}

func TestValidator_workflow_bad_initial_step(t *testing.T) {
	v := NewValidator()
	def := validDomain()
	def.Workflows = []model.WorkflowDefinition{
		{
			ID:          "orders.approval",
			Name:        "Order Approval",
			InitialStep: "nonexistent",
			Steps: []model.StepDefinition{
				{ID: "review", Name: "Review", Type: "approval"},
				{ID: "done", Name: "Done", Type: "terminal"},
			},
		},
	}
	errs := v.Validate([]model.DomainDefinition{def}, nil)
	if !hasCode(errs, "REF_NOT_FOUND") {
		t.Error("expected REF_NOT_FOUND for bad initial_step")
	}
}

func TestValidator_workflow_bad_step_type(t *testing.T) {
	v := NewValidator()
	def := validDomain()
	def.Workflows = []model.WorkflowDefinition{
		{
			ID:          "w1",
			Name:        "W1",
			InitialStep: "s1",
			Steps: []model.StepDefinition{
				{ID: "s1", Name: "S1", Type: "badtype"},
				{ID: "s2", Name: "S2", Type: "terminal"},
			},
		},
	}
	errs := v.Validate([]model.DomainDefinition{def}, nil)
	if !hasCode(errs, "INVALID_ENUM") {
		t.Error("expected INVALID_ENUM for bad step type")
	}
}

func TestValidator_capability_namespace_mismatch(t *testing.T) {
	v := NewValidator()
	def := validDomain()
	def.Pages[0].Capabilities = []string{"inventory:list:view"}
	errs := v.Validate([]model.DomainDefinition{def}, nil)
	if !hasCode(errs, "NAMESPACE_MISMATCH") {
		t.Error("expected NAMESPACE_MISMATCH error")
	}
}

func hasCode(errs []VError, code string) bool {
	for _, e := range errs {
		if e.Code == code {
			return true
		}
	}
	return false
}
