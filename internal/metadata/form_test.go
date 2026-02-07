package metadata

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/model"
)

func testFormDefinitions() []model.DomainDefinition {
	minLen := 2
	maxLen := 100
	minVal := 0.0
	maxVal := 999.99

	return []model.DomainDefinition{
		{
			Domain: "users",
			Forms: []model.FormDefinition{
				{
					ID:           "create-user",
					Title:        "Create User",
					SubmitCommand: "create-user-cmd",
					SuccessRoute: "/users/{id}",
					SuccessMessage: "User created successfully",
					Sections: []model.SectionDefinition{
						{
							ID:     "basic-info",
							Title:  "Basic Information",
							Layout: "grid",
							Columns: 2,
							Fields: []model.FieldDefinition{
								{
									Field:       "first_name",
									Label:       "First Name",
									Type:        "text",
									Required:    true,
									Placeholder: "Enter first name",
									HelpText:    "Your legal first name",
									Validation: &model.ValidationDefinition{
										MinLength: &minLen,
										MaxLength: &maxLen,
										Pattern:   "^[a-zA-Z]+$",
										Message:   "Only letters allowed",
									},
								},
								{
									Field:    "last_name",
									Label:    "Last Name",
									Type:     "text",
									Required: true,
								},
								{
									Field:    "email",
									Label:    "Email",
									Type:     "email",
									Required: true,
									Format:   "email",
									Span:     2,
								},
								{
									Field:    "role",
									Label:    "Role",
									Type:     "select",
									Lookup: &model.LookupRefDefinition{
										Static: []model.StaticOption{
											{Label: "Admin", Value: "admin"},
											{Label: "Editor", Value: "editor"},
											{Label: "Viewer", Value: "viewer"},
										},
									},
								},
								{
									Field:      "internal_id",
									Label:      "Internal ID",
									Type:       "text",
									Visibility: "hidden",
								},
							},
						},
						{
							ID:          "advanced",
							Title:       "Advanced Settings",
							Layout:      "form",
							Collapsible: true,
							Collapsed:   true,
							Fields: []model.FieldDefinition{
								{
									Field: "salary",
									Label: "Salary",
									Type:  "number",
									Validation: &model.ValidationDefinition{
										Min: &minVal,
										Max: &maxVal,
									},
								},
								{
									Field: "department",
									Label: "Department",
									Type:  "select",
									DependsOn: []model.FieldDependency{
										{Field: "role", Condition: "eq", Value: "admin"},
									},
								},
							},
						},
					},
				},
				{
					ID:            "edit-user",
					Title:         "Edit User",
					Capabilities:  []string{"users:edit"},
					SubmitCommand: "update-user-cmd",
					LoadSource: &model.DataSourceDefinition{
						OperationID: "getUser",
						ServiceID:   "user-svc",
						Mapping: model.ResponseMappingDefinition{
							FieldMap: map[string]string{
								"user_name": "name",
								"user_email": "email",
							},
						},
					},
					Sections: []model.SectionDefinition{
						{
							ID:     "user-info",
							Title:  "User Info",
							Layout: "form",
							Fields: []model.FieldDefinition{
								{
									Field:    "name",
									Label:    "Name",
									Type:     "text",
									ReadOnly: "true",
								},
								{
									Field: "email",
									Label: "Email",
									Type:  "email",
								},
								{
									Field:    "status",
									Label:    "Status",
									Type:     "text",
									ReadOnly: "always",
								},
							},
						},
						{
							ID:           "admin-section",
							Title:        "Admin Settings",
							Layout:       "form",
							Capabilities: []string{"users:admin"},
							Fields: []model.FieldDefinition{
								{Field: "is_admin", Label: "Admin", Type: "checkbox"},
							},
						},
					},
				},
				{
					ID:            "sdk-form",
					Title:         "SDK Form",
					SubmitCommand: "sdk-cmd",
					LoadSource: &model.DataSourceDefinition{
						Handler: "myHandler",
					},
					Sections: []model.SectionDefinition{
						{
							ID:     "sdk-section",
							Title:  "SDK Section",
							Layout: "form",
							Fields: []model.FieldDefinition{
								{Field: "field_a", Label: "Field A", Type: "text"},
								{Field: "field_b", Label: "Field B", Type: "text"},
							},
						},
					},
				},
			},
		},
	}
}

func newTestFormProvider(invokeFn func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error)) *FormProvider {
	reg := definition.NewRegistry(testFormDefinitions())
	ap := NewActionProvider()

	invokerReg := invoker.NewRegistry()
	if invokeFn != nil {
		invokerReg.Register(&mockInvokerForMenu{invokeFn: invokeFn})
	}

	return NewFormProvider(reg, invokerReg, ap)
}

// --- GetForm ---

func TestFormProvider_GetForm_success(t *testing.T) {
	p := newTestFormProvider(nil)

	desc, err := p.GetForm(context.Background(), nil, model.CapabilitySet{}, "create-user")
	if err != nil {
		t.Fatalf("GetForm error: %v", err)
	}
	if desc.ID != "create-user" {
		t.Errorf("ID = %q, want create-user", desc.ID)
	}
	if desc.Title != "Create User" {
		t.Errorf("Title = %q, want Create User", desc.Title)
	}
	if desc.SuccessRoute != "/users/{id}" {
		t.Errorf("SuccessRoute = %q, want /users/{id}", desc.SuccessRoute)
	}
	if desc.SuccessMessage != "User created successfully" {
		t.Errorf("SuccessMessage = %q", desc.SuccessMessage)
	}
}

func TestFormProvider_GetForm_submitEndpoint(t *testing.T) {
	p := newTestFormProvider(nil)

	desc, err := p.GetForm(context.Background(), nil, model.CapabilitySet{}, "create-user")
	if err != nil {
		t.Fatalf("GetForm error: %v", err)
	}
	want := "/api/commands/create-user-cmd"
	if desc.SubmitEndpoint != want {
		t.Errorf("SubmitEndpoint = %q, want %q", desc.SubmitEndpoint, want)
	}
}

func TestFormProvider_GetForm_notFound(t *testing.T) {
	p := newTestFormProvider(nil)

	_, err := p.GetForm(context.Background(), nil, model.CapabilitySet{}, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent form")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T, want *model.ErrorEnvelope", err)
	}
	if envErr.Code != model.ErrNotFound {
		t.Errorf("error code = %s, want %s", envErr.Code, model.ErrNotFound)
	}
}

func TestFormProvider_GetForm_forbidden(t *testing.T) {
	p := newTestFormProvider(nil)

	// edit-user requires "users:edit" capability.
	_, err := p.GetForm(context.Background(), nil, model.CapabilitySet{}, "edit-user")
	if err == nil {
		t.Fatal("expected error for insufficient capabilities")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T, want *model.ErrorEnvelope", err)
	}
	if envErr.Code != model.ErrForbidden {
		t.Errorf("error code = %s, want %s", envErr.Code, model.ErrForbidden)
	}
}

func TestFormProvider_GetForm_noCapabilitiesRequired(t *testing.T) {
	p := newTestFormProvider(nil)

	// create-user has no capabilities requirement.
	desc, err := p.GetForm(context.Background(), nil, model.CapabilitySet{}, "create-user")
	if err != nil {
		t.Fatalf("GetForm error: %v", err)
	}
	if desc.ID != "create-user" {
		t.Errorf("ID = %q, want create-user", desc.ID)
	}
}

func TestFormProvider_GetForm_sections(t *testing.T) {
	p := newTestFormProvider(nil)

	desc, err := p.GetForm(context.Background(), nil, model.CapabilitySet{}, "create-user")
	if err != nil {
		t.Fatalf("GetForm error: %v", err)
	}
	if len(desc.Sections) != 2 {
		t.Fatalf("len(Sections) = %d, want 2", len(desc.Sections))
	}

	basic := desc.Sections[0]
	if basic.ID != "basic-info" {
		t.Errorf("Sections[0].ID = %q, want basic-info", basic.ID)
	}
	if basic.Title != "Basic Information" {
		t.Errorf("Sections[0].Title = %q", basic.Title)
	}
	if basic.Layout != "grid" {
		t.Errorf("Sections[0].Layout = %q, want grid", basic.Layout)
	}
	if basic.Columns != 2 {
		t.Errorf("Sections[0].Columns = %d, want 2", basic.Columns)
	}

	advanced := desc.Sections[1]
	if !advanced.Collapsible {
		t.Error("Sections[1].Collapsible = false, want true")
	}
	if !advanced.Collapsed {
		t.Error("Sections[1].Collapsed = false, want true")
	}
}

func TestFormProvider_GetForm_sectionsFilteredByCapability(t *testing.T) {
	p := newTestFormProvider(nil)

	// edit-user has admin-section requiring "users:admin".
	caps := model.CapabilitySet{"users:edit": true}
	desc, err := p.GetForm(context.Background(), nil, caps, "edit-user")
	if err != nil {
		t.Fatalf("GetForm error: %v", err)
	}
	// Only user-info section (admin-section filtered out).
	if len(desc.Sections) != 1 {
		t.Fatalf("len(Sections) = %d, want 1 (admin-section filtered)", len(desc.Sections))
	}
	if desc.Sections[0].ID != "user-info" {
		t.Errorf("Sections[0].ID = %q, want user-info", desc.Sections[0].ID)
	}

	// With admin capability.
	caps = model.CapabilitySet{"users:edit": true, "users:admin": true}
	desc, err = p.GetForm(context.Background(), nil, caps, "edit-user")
	if err != nil {
		t.Fatalf("GetForm error: %v", err)
	}
	if len(desc.Sections) != 2 {
		t.Fatalf("len(Sections) = %d, want 2", len(desc.Sections))
	}
}

func TestFormProvider_GetForm_hiddenFieldsOmitted(t *testing.T) {
	p := newTestFormProvider(nil)

	desc, err := p.GetForm(context.Background(), nil, model.CapabilitySet{}, "create-user")
	if err != nil {
		t.Fatalf("GetForm error: %v", err)
	}
	// basic-info has 5 fields, but internal_id is hidden → 4 visible.
	fields := desc.Sections[0].Fields
	if len(fields) != 4 {
		t.Fatalf("len(Fields) = %d, want 4 (hidden field omitted)", len(fields))
	}
	for _, f := range fields {
		if f.Field == "internal_id" {
			t.Error("internal_id should be hidden and omitted")
		}
	}
}

func TestFormProvider_GetForm_fieldProperties(t *testing.T) {
	p := newTestFormProvider(nil)

	desc, err := p.GetForm(context.Background(), nil, model.CapabilitySet{}, "create-user")
	if err != nil {
		t.Fatalf("GetForm error: %v", err)
	}
	fields := desc.Sections[0].Fields

	// first_name
	fn := fields[0]
	if fn.Field != "first_name" {
		t.Errorf("Field = %q, want first_name", fn.Field)
	}
	if fn.Label != "First Name" {
		t.Errorf("Label = %q, want First Name", fn.Label)
	}
	if fn.Type != "text" {
		t.Errorf("Type = %q, want text", fn.Type)
	}
	if !fn.Required {
		t.Error("Required = false, want true")
	}
	if fn.Placeholder != "Enter first name" {
		t.Errorf("Placeholder = %q", fn.Placeholder)
	}
	if fn.HelpText != "Your legal first name" {
		t.Errorf("HelpText = %q", fn.HelpText)
	}

	// email (span)
	email := fields[2]
	if email.Span != 2 {
		t.Errorf("email.Span = %d, want 2", email.Span)
	}
	if email.Format != "email" {
		t.Errorf("email.Format = %q, want email", email.Format)
	}
}

func TestFormProvider_GetForm_validation(t *testing.T) {
	p := newTestFormProvider(nil)

	desc, err := p.GetForm(context.Background(), nil, model.CapabilitySet{}, "create-user")
	if err != nil {
		t.Fatalf("GetForm error: %v", err)
	}

	// first_name has validation.
	fn := desc.Sections[0].Fields[0]
	if fn.Validation == nil {
		t.Fatal("Validation is nil")
	}
	if *fn.Validation.MinLength != 2 {
		t.Errorf("MinLength = %d, want 2", *fn.Validation.MinLength)
	}
	if *fn.Validation.MaxLength != 100 {
		t.Errorf("MaxLength = %d, want 100", *fn.Validation.MaxLength)
	}
	if fn.Validation.Pattern != "^[a-zA-Z]+$" {
		t.Errorf("Pattern = %q", fn.Validation.Pattern)
	}
	if fn.Validation.Message != "Only letters allowed" {
		t.Errorf("Message = %q", fn.Validation.Message)
	}

	// salary has min/max.
	salary := desc.Sections[1].Fields[0]
	if salary.Validation == nil {
		t.Fatal("salary.Validation is nil")
	}
	if *salary.Validation.Min != 0.0 {
		t.Errorf("Min = %f, want 0", *salary.Validation.Min)
	}
	if *salary.Validation.Max != 999.99 {
		t.Errorf("Max = %f, want 999.99", *salary.Validation.Max)
	}
}

func TestFormProvider_GetForm_lookupStaticOptions(t *testing.T) {
	p := newTestFormProvider(nil)

	desc, err := p.GetForm(context.Background(), nil, model.CapabilitySet{}, "create-user")
	if err != nil {
		t.Fatalf("GetForm error: %v", err)
	}

	// role field has static lookup options.
	role := desc.Sections[0].Fields[3]
	if role.Field != "role" {
		t.Fatalf("Field = %q, want role", role.Field)
	}
	if len(role.Options) != 3 {
		t.Fatalf("len(Options) = %d, want 3", len(role.Options))
	}
	if role.Options[0].Label != "Admin" || role.Options[0].Value != "admin" {
		t.Errorf("Options[0] = %+v", role.Options[0])
	}
	if role.Options[2].Label != "Viewer" || role.Options[2].Value != "viewer" {
		t.Errorf("Options[2] = %+v", role.Options[2])
	}
}

func TestFormProvider_GetForm_dependsOn(t *testing.T) {
	p := newTestFormProvider(nil)

	desc, err := p.GetForm(context.Background(), nil, model.CapabilitySet{}, "create-user")
	if err != nil {
		t.Fatalf("GetForm error: %v", err)
	}

	// department field in advanced section has depends_on.
	dept := desc.Sections[1].Fields[1]
	if dept.Field != "department" {
		t.Fatalf("Field = %q, want department", dept.Field)
	}
	if len(dept.DependsOn) != 1 {
		t.Fatalf("len(DependsOn) = %d, want 1", len(dept.DependsOn))
	}
	dep := dept.DependsOn[0]
	if dep.Field != "role" {
		t.Errorf("DependsOn[0].Field = %q, want role", dep.Field)
	}
	if dep.Condition != "eq" {
		t.Errorf("DependsOn[0].Condition = %q, want eq", dep.Condition)
	}
	if dep.Value != "admin" {
		t.Errorf("DependsOn[0].Value = %q, want admin", dep.Value)
	}
}

func TestFormProvider_GetForm_readOnly(t *testing.T) {
	p := newTestFormProvider(nil)

	caps := model.CapabilitySet{"users:edit": true}
	desc, err := p.GetForm(context.Background(), nil, caps, "edit-user")
	if err != nil {
		t.Fatalf("GetForm error: %v", err)
	}

	fields := desc.Sections[0].Fields
	// name has read_only="true" → ReadOnly=true.
	if !fields[0].ReadOnly {
		t.Error("name.ReadOnly = false, want true")
	}
	// email has no read_only → ReadOnly=false.
	if fields[1].ReadOnly {
		t.Error("email.ReadOnly = true, want false")
	}
	// status has read_only="always" → ReadOnly=true.
	if !fields[2].ReadOnly {
		t.Error("status.ReadOnly = false, want true (always)")
	}
}

// --- GetFormData ---

func TestFormProvider_GetFormData_success(t *testing.T) {
	p := newTestFormProvider(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		if binding.OperationID != "getUser" {
			return model.InvocationResult{}, fmt.Errorf("unexpected operation: %s", binding.OperationID)
		}
		if binding.ServiceID != "user-svc" {
			return model.InvocationResult{}, fmt.Errorf("unexpected service: %s", binding.ServiceID)
		}
		if binding.Type != "openapi" {
			return model.InvocationResult{}, fmt.Errorf("unexpected type: %s", binding.Type)
		}
		return model.InvocationResult{
			StatusCode: http.StatusOK,
			Body: map[string]any{
				"user_name":  "Alice",
				"user_email": "alice@example.com",
				"status":     "active",
				"extra_field": "should be excluded",
			},
		}, nil
	})

	caps := model.CapabilitySet{"users:edit": true}
	data, err := p.GetFormData(context.Background(), nil, caps, "edit-user", nil)
	if err != nil {
		t.Fatalf("GetFormData error: %v", err)
	}

	// field_map should rename user_name → name, user_email → email.
	if data["name"] != "Alice" {
		t.Errorf("data[name] = %v, want Alice", data["name"])
	}
	if data["email"] != "alice@example.com" {
		t.Errorf("data[email] = %v, want alice@example.com", data["email"])
	}
	if data["status"] != "active" {
		t.Errorf("data[status] = %v, want active", data["status"])
	}
	// extra_field is not in the form's fields → filtered out.
	if _, exists := data["extra_field"]; exists {
		t.Error("extra_field should be filtered out (not in form fields)")
	}
	// user_name/user_email should not appear (renamed).
	if _, exists := data["user_name"]; exists {
		t.Error("user_name should not appear (renamed to name)")
	}
}

func TestFormProvider_GetFormData_notFound(t *testing.T) {
	p := newTestFormProvider(nil)

	_, err := p.GetFormData(context.Background(), nil, model.CapabilitySet{}, "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent form")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T, want *model.ErrorEnvelope", err)
	}
	if envErr.Code != model.ErrNotFound {
		t.Errorf("error code = %s, want %s", envErr.Code, model.ErrNotFound)
	}
}

func TestFormProvider_GetFormData_forbidden(t *testing.T) {
	p := newTestFormProvider(nil)

	// edit-user requires "users:edit".
	_, err := p.GetFormData(context.Background(), nil, model.CapabilitySet{}, "edit-user", nil)
	if err == nil {
		t.Fatal("expected error for insufficient capabilities")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T, want *model.ErrorEnvelope", err)
	}
	if envErr.Code != model.ErrForbidden {
		t.Errorf("error code = %s, want %s", envErr.Code, model.ErrForbidden)
	}
}

func TestFormProvider_GetFormData_noLoadSource(t *testing.T) {
	p := newTestFormProvider(nil)

	// create-user has no load_source.
	data, err := p.GetFormData(context.Background(), nil, model.CapabilitySet{}, "create-user", nil)
	if err != nil {
		t.Fatalf("GetFormData error: %v", err)
	}
	if data != nil {
		t.Errorf("data = %v, want nil (no load source)", data)
	}
}

func TestFormProvider_GetFormData_passesPathParams(t *testing.T) {
	var capturedInput model.InvocationInput
	p := newTestFormProvider(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		capturedInput = input
		return model.InvocationResult{
			StatusCode: http.StatusOK,
			Body:       map[string]any{"name": "Alice", "email": "a@b.com", "status": "ok"},
		}, nil
	})

	caps := model.CapabilitySet{"users:edit": true}
	params := map[string]string{"id": "user-123"}

	_, err := p.GetFormData(context.Background(), nil, caps, "edit-user", params)
	if err != nil {
		t.Fatalf("GetFormData error: %v", err)
	}
	if capturedInput.PathParams["id"] != "user-123" {
		t.Errorf("PathParams[id] = %q, want user-123", capturedInput.PathParams["id"])
	}
}

func TestFormProvider_GetFormData_backendError(t *testing.T) {
	p := newTestFormProvider(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		return model.InvocationResult{}, fmt.Errorf("backend unavailable")
	})

	caps := model.CapabilitySet{"users:edit": true}
	_, err := p.GetFormData(context.Background(), nil, caps, "edit-user", nil)
	if err == nil {
		t.Fatal("expected error from backend")
	}
}

func TestFormProvider_GetFormData_nonMapResponseBody(t *testing.T) {
	p := newTestFormProvider(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		return model.InvocationResult{
			StatusCode: http.StatusOK,
			Body:       "not a map",
		}, nil
	})

	caps := model.CapabilitySet{"users:edit": true}
	data, err := p.GetFormData(context.Background(), nil, caps, "edit-user", nil)
	if err != nil {
		t.Fatalf("GetFormData error: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("data = %v, want empty map", data)
	}
}

func TestFormProvider_GetFormData_sdkBinding(t *testing.T) {
	var capturedBinding model.OperationBinding
	p := newTestFormProvider(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		capturedBinding = binding
		return model.InvocationResult{
			StatusCode: http.StatusOK,
			Body:       map[string]any{"field_a": "val_a", "field_b": "val_b"},
		}, nil
	})

	data, err := p.GetFormData(context.Background(), nil, model.CapabilitySet{}, "sdk-form", nil)
	if err != nil {
		t.Fatalf("GetFormData error: %v", err)
	}

	// SDK handler → binding type should be "sdk".
	if capturedBinding.Type != "sdk" {
		t.Errorf("binding.Type = %q, want sdk", capturedBinding.Type)
	}
	if capturedBinding.Handler != "myHandler" {
		t.Errorf("binding.Handler = %q, want myHandler", capturedBinding.Handler)
	}

	// Data should be returned, filtered to form fields.
	if data["field_a"] != "val_a" {
		t.Errorf("data[field_a] = %v, want val_a", data["field_a"])
	}
	if data["field_b"] != "val_b" {
		t.Errorf("data[field_b] = %v, want val_b", data["field_b"])
	}
}

func TestFormProvider_GetFormData_filtersToFormFields(t *testing.T) {
	p := newTestFormProvider(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		return model.InvocationResult{
			StatusCode: http.StatusOK,
			Body: map[string]any{
				"field_a":     "keep",
				"field_b":     "keep",
				"field_extra": "should be excluded",
			},
		}, nil
	})

	data, err := p.GetFormData(context.Background(), nil, model.CapabilitySet{}, "sdk-form", nil)
	if err != nil {
		t.Fatalf("GetFormData error: %v", err)
	}
	if data["field_a"] != "keep" {
		t.Errorf("data[field_a] = %v", data["field_a"])
	}
	if data["field_b"] != "keep" {
		t.Errorf("data[field_b] = %v", data["field_b"])
	}
	if _, exists := data["field_extra"]; exists {
		t.Error("field_extra should be filtered out")
	}
}

func TestFormProvider_GetFormData_noFieldMapWhenEmpty(t *testing.T) {
	p := newTestFormProvider(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		return model.InvocationResult{
			StatusCode: http.StatusOK,
			Body: map[string]any{
				"field_a": "val_a",
				"field_b": "val_b",
			},
		}, nil
	})

	// sdk-form has no field_map, so keys should pass through as-is.
	data, err := p.GetFormData(context.Background(), nil, model.CapabilitySet{}, "sdk-form", nil)
	if err != nil {
		t.Fatalf("GetFormData error: %v", err)
	}
	if data["field_a"] != "val_a" {
		t.Errorf("data[field_a] = %v, want val_a", data["field_a"])
	}
}

// --- Helper function tests ---

func TestCollectFormFields(t *testing.T) {
	sections := []model.SectionDefinition{
		{
			Fields: []model.FieldDefinition{
				{Field: "name"},
				{Field: "email"},
			},
		},
		{
			Fields: []model.FieldDefinition{
				{Field: "age"},
			},
		},
	}

	fields := collectFormFields(sections)
	if len(fields) != 3 {
		t.Fatalf("len(fields) = %d, want 3", len(fields))
	}
	for _, f := range []string{"name", "email", "age"} {
		if !fields[f] {
			t.Errorf("fields[%q] = false, want true", f)
		}
	}
}

func TestCollectFormFields_empty(t *testing.T) {
	fields := collectFormFields(nil)
	if len(fields) != 0 {
		t.Errorf("len(fields) = %d, want 0", len(fields))
	}
}

func TestFilterToFields(t *testing.T) {
	data := map[string]any{
		"name":  "Alice",
		"email": "alice@example.com",
		"extra": "should be excluded",
	}
	fields := map[string]bool{
		"name":  true,
		"email": true,
	}

	result := filterToFields(data, fields)
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	if result["name"] != "Alice" {
		t.Errorf("result[name] = %v, want Alice", result["name"])
	}
	if result["email"] != "alice@example.com" {
		t.Errorf("result[email] = %v", result["email"])
	}
	if _, exists := result["extra"]; exists {
		t.Error("extra should not be in result")
	}
}

func TestFilterToFields_emptyData(t *testing.T) {
	result := filterToFields(map[string]any{}, map[string]bool{"name": true})
	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}

func TestRenameFields(t *testing.T) {
	data := map[string]any{
		"old_name":  "Alice",
		"old_email": "alice@example.com",
		"age":       float64(30),
	}
	fieldMap := map[string]string{
		"old_name":  "name",
		"old_email": "email",
	}

	result := renameFields(data, fieldMap)
	if result["name"] != "Alice" {
		t.Errorf("result[name] = %v, want Alice", result["name"])
	}
	if result["email"] != "alice@example.com" {
		t.Errorf("result[email] = %v", result["email"])
	}
	if result["age"] != float64(30) {
		t.Errorf("result[age] = %v, want 30", result["age"])
	}
	// old_name should not exist (renamed).
	if _, exists := result["old_name"]; exists {
		t.Error("old_name should not exist after rename")
	}
}

func TestRenameFields_noMapping(t *testing.T) {
	data := map[string]any{"name": "Alice"}
	result := renameFields(data, map[string]string{})
	if result["name"] != "Alice" {
		t.Errorf("result[name] = %v, want Alice", result["name"])
	}
}

func TestRenameFields_emptyData(t *testing.T) {
	result := renameFields(map[string]any{}, map[string]string{"a": "b"})
	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}
