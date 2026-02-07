package definition

import (
	"fmt"
	"strings"

	"github.com/pitabwire/thesa/internal/openapi"
	"github.com/pitabwire/thesa/model"
)

// VError describes a single validation error in a definition.
type VError struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e VError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// Validator validates definitions structurally, referentially, and against OpenAPI specs.
type Validator struct{}

// NewValidator creates a new Validator.
func NewValidator() *Validator {
	return &Validator{}
}

// Validate checks all definitions. The index may be nil to skip OpenAPI checks.
func (v *Validator) Validate(defs []model.DomainDefinition, index *openapi.Index) []VError {
	var errs []VError
	for i, def := range defs {
		prefix := fmt.Sprintf("definitions[%d]", i)
		errs = append(errs, v.validateDomain(prefix, def, index)...)
	}
	return errs
}

func (v *Validator) validateDomain(prefix string, def model.DomainDefinition, index *openapi.Index) []VError {
	var errs []VError

	if def.Domain == "" {
		errs = append(errs, VError{Path: prefix + ".domain", Code: "REQUIRED", Message: "domain is required"})
	}
	if def.Version == "" {
		errs = append(errs, VError{Path: prefix + ".version", Code: "REQUIRED", Message: "version is required"})
	}
	if def.Navigation.Label == "" {
		errs = append(errs, VError{Path: prefix + ".navigation.label", Code: "REQUIRED", Message: "navigation.label is required"})
	}
	if len(def.Navigation.Children) == 0 {
		errs = append(errs, VError{Path: prefix + ".navigation.children", Code: "REQUIRED", Message: "at least one navigation child is required"})
	}

	// Build lookup sets for referential validation.
	formIDs := make(map[string]bool)
	for _, f := range def.Forms {
		formIDs[f.ID] = true
	}
	commandIDs := make(map[string]bool)
	for _, c := range def.Commands {
		commandIDs[c.ID] = true
	}
	workflowIDs := make(map[string]bool)
	for _, w := range def.Workflows {
		workflowIDs[w.ID] = true
	}
	lookupIDs := make(map[string]bool)
	for _, l := range def.Lookups {
		lookupIDs[l.ID] = true
	}

	for i, p := range def.Pages {
		pp := fmt.Sprintf("%s.pages[%d]", prefix, i)
		errs = append(errs, v.validatePage(pp, p, def.Domain, index)...)
	}
	for i, f := range def.Forms {
		fp := fmt.Sprintf("%s.forms[%d]", prefix, i)
		errs = append(errs, v.validateForm(fp, f, commandIDs)...)
	}
	for i, c := range def.Commands {
		cp := fmt.Sprintf("%s.commands[%d]", prefix, i)
		errs = append(errs, v.validateCommand(cp, c, def.Domain, index)...)
	}
	for i, w := range def.Workflows {
		wp := fmt.Sprintf("%s.workflows[%d]", prefix, i)
		errs = append(errs, v.validateWorkflow(wp, w)...)
	}
	for i, a := range def.Pages {
		for j, action := range a.Actions {
			ap := fmt.Sprintf("%s.pages[%d].actions[%d]", prefix, i, j)
			errs = append(errs, v.validateActionRef(ap, action, formIDs, commandIDs, workflowIDs)...)
		}
	}

	// Validate capability namespaces match domain ID.
	if def.Domain != "" {
		for i, p := range def.Pages {
			for _, cap := range p.Capabilities {
				if !strings.HasPrefix(cap, def.Domain+":") && cap != "*" {
					errs = append(errs, VError{
						Path:    fmt.Sprintf("%s.pages[%d].capabilities", prefix, i),
						Code:    "NAMESPACE_MISMATCH",
						Message: fmt.Sprintf("capability %q does not match domain %q", cap, def.Domain),
					})
				}
			}
		}
	}

	return errs
}

var validPageLayouts = map[string]bool{
	"list": true, "detail": true, "dashboard": true, "custom": true,
}

func (v *Validator) validatePage(prefix string, p model.PageDefinition, domain string, index *openapi.Index) []VError {
	var errs []VError

	if p.ID == "" {
		errs = append(errs, VError{Path: prefix + ".id", Code: "REQUIRED", Message: "id is required"})
	}
	if p.Title == "" {
		errs = append(errs, VError{Path: prefix + ".title", Code: "REQUIRED", Message: "title is required"})
	}
	if p.Layout == "" {
		errs = append(errs, VError{Path: prefix + ".layout", Code: "REQUIRED", Message: "layout is required"})
	} else if !validPageLayouts[p.Layout] {
		errs = append(errs, VError{Path: prefix + ".layout", Code: "INVALID_ENUM", Message: fmt.Sprintf("invalid layout %q", p.Layout)})
	}

	if p.Layout == "list" && p.Table == nil {
		errs = append(errs, VError{Path: prefix + ".table", Code: "REQUIRED", Message: "table is required for list layout"})
	}

	if p.Table != nil {
		errs = append(errs, v.validateTable(prefix+".table", *p.Table, domain, index)...)
	}

	return errs
}

func (v *Validator) validateTable(prefix string, t model.TableDefinition, domain string, index *openapi.Index) []VError {
	var errs []VError

	if len(t.Columns) == 0 {
		errs = append(errs, VError{Path: prefix + ".columns", Code: "REQUIRED", Message: "at least one column is required"})
	}

	if t.PageSize < 0 || t.PageSize > 200 {
		errs = append(errs, VError{Path: prefix + ".page_size", Code: "RANGE", Message: "page_size must be 0-200"})
	}

	// Validate operation_id against OpenAPI index.
	if index != nil && t.DataSource.OperationID != "" {
		serviceID := t.DataSource.ServiceID
		if serviceID == "" {
			serviceID = domain + "-svc"
		}
		if _, ok := index.GetOperation(serviceID, t.DataSource.OperationID); !ok {
			errs = append(errs, VError{
				Path:    prefix + ".data_source.operation_id",
				Code:    "OPERATION_NOT_FOUND",
				Message: fmt.Sprintf("operation %q not found in service %q", t.DataSource.OperationID, serviceID),
			})
		}
	}

	return errs
}

func (v *Validator) validateForm(prefix string, f model.FormDefinition, commandIDs map[string]bool) []VError {
	var errs []VError

	if f.ID == "" {
		errs = append(errs, VError{Path: prefix + ".id", Code: "REQUIRED", Message: "id is required"})
	}
	if f.Title == "" {
		errs = append(errs, VError{Path: prefix + ".title", Code: "REQUIRED", Message: "title is required"})
	}
	if f.SubmitCommand == "" {
		errs = append(errs, VError{Path: prefix + ".submit_command", Code: "REQUIRED", Message: "submit_command is required"})
	} else if !commandIDs[f.SubmitCommand] {
		errs = append(errs, VError{
			Path:    prefix + ".submit_command",
			Code:    "REF_NOT_FOUND",
			Message: fmt.Sprintf("command %q not found in domain", f.SubmitCommand),
		})
	}
	if len(f.Sections) == 0 {
		errs = append(errs, VError{Path: prefix + ".sections", Code: "REQUIRED", Message: "at least one section is required"})
	}

	return errs
}

func (v *Validator) validateCommand(prefix string, c model.CommandDefinition, domain string, index *openapi.Index) []VError {
	var errs []VError

	if c.ID == "" {
		errs = append(errs, VError{Path: prefix + ".id", Code: "REQUIRED", Message: "id is required"})
	}

	opType := c.Operation.Type
	if opType == "" {
		errs = append(errs, VError{Path: prefix + ".operation.type", Code: "REQUIRED", Message: "operation.type is required"})
	} else if opType != "openapi" && opType != "sdk" {
		errs = append(errs, VError{Path: prefix + ".operation.type", Code: "INVALID_ENUM", Message: fmt.Sprintf("invalid operation type %q", opType)})
	}

	if opType == "openapi" && c.Operation.OperationID == "" {
		errs = append(errs, VError{Path: prefix + ".operation.operation_id", Code: "REQUIRED", Message: "operation_id required for openapi type"})
	}
	if opType == "sdk" && c.Operation.Handler == "" {
		errs = append(errs, VError{Path: prefix + ".operation.handler", Code: "REQUIRED", Message: "handler required for sdk type"})
	}

	// Validate against OpenAPI index.
	if index != nil && opType == "openapi" && c.Operation.OperationID != "" {
		serviceID := c.Operation.ServiceID
		if serviceID == "" {
			serviceID = domain + "-svc"
		}
		if _, ok := index.GetOperation(serviceID, c.Operation.OperationID); !ok {
			errs = append(errs, VError{
				Path:    prefix + ".operation.operation_id",
				Code:    "OPERATION_NOT_FOUND",
				Message: fmt.Sprintf("operation %q not found in service %q", c.Operation.OperationID, serviceID),
			})
		}
	}

	return errs
}

var validStepTypes = map[string]bool{
	"action": true, "approval": true, "system": true,
	"wait": true, "notification": true, "terminal": true,
}

func (v *Validator) validateWorkflow(prefix string, w model.WorkflowDefinition) []VError {
	var errs []VError

	if w.ID == "" {
		errs = append(errs, VError{Path: prefix + ".id", Code: "REQUIRED", Message: "id is required"})
	}
	if w.Name == "" {
		errs = append(errs, VError{Path: prefix + ".name", Code: "REQUIRED", Message: "name is required"})
	}
	if w.InitialStep == "" {
		errs = append(errs, VError{Path: prefix + ".initial_step", Code: "REQUIRED", Message: "initial_step is required"})
	}
	if len(w.Steps) < 2 {
		errs = append(errs, VError{Path: prefix + ".steps", Code: "REQUIRED", Message: "at least two steps required (initial + terminal)"})
	}

	stepIDs := make(map[string]bool)
	for i, s := range w.Steps {
		sp := fmt.Sprintf("%s.steps[%d]", prefix, i)
		if s.ID == "" {
			errs = append(errs, VError{Path: sp + ".id", Code: "REQUIRED", Message: "step id is required"})
		}
		stepIDs[s.ID] = true
		if s.Type == "" {
			errs = append(errs, VError{Path: sp + ".type", Code: "REQUIRED", Message: "step type is required"})
		} else if !validStepTypes[s.Type] {
			errs = append(errs, VError{Path: sp + ".type", Code: "INVALID_ENUM", Message: fmt.Sprintf("invalid step type %q", s.Type)})
		}
	}

	// Validate initial_step references a valid step.
	if w.InitialStep != "" && !stepIDs[w.InitialStep] {
		errs = append(errs, VError{
			Path:    prefix + ".initial_step",
			Code:    "REF_NOT_FOUND",
			Message: fmt.Sprintf("initial_step %q not found in steps", w.InitialStep),
		})
	}

	// Validate transitions reference valid steps.
	for i, tr := range w.Transitions {
		tp := fmt.Sprintf("%s.transitions[%d]", prefix, i)
		if tr.From != "" && !stepIDs[tr.From] {
			errs = append(errs, VError{Path: tp + ".from", Code: "REF_NOT_FOUND", Message: fmt.Sprintf("step %q not found", tr.From)})
		}
		if tr.To != "" && !stepIDs[tr.To] {
			errs = append(errs, VError{Path: tp + ".to", Code: "REF_NOT_FOUND", Message: fmt.Sprintf("step %q not found", tr.To)})
		}
		if tr.Event == "" {
			errs = append(errs, VError{Path: tp + ".event", Code: "REQUIRED", Message: "transition event is required"})
		}
	}

	return errs
}

func (v *Validator) validateActionRef(prefix string, a model.ActionDefinition, formIDs, commandIDs, workflowIDs map[string]bool) []VError {
	var errs []VError

	switch a.Type {
	case "form":
		if a.FormID != "" && !formIDs[a.FormID] {
			errs = append(errs, VError{Path: prefix + ".form_id", Code: "REF_NOT_FOUND", Message: fmt.Sprintf("form %q not found", a.FormID)})
		}
	case "command", "confirm":
		if a.CommandID != "" && !commandIDs[a.CommandID] {
			errs = append(errs, VError{Path: prefix + ".command_id", Code: "REF_NOT_FOUND", Message: fmt.Sprintf("command %q not found", a.CommandID)})
		}
	case "workflow":
		if a.WorkflowID != "" && !workflowIDs[a.WorkflowID] {
			errs = append(errs, VError{Path: prefix + ".workflow_id", Code: "REF_NOT_FOUND", Message: fmt.Sprintf("workflow %q not found", a.WorkflowID)})
		}
	}

	return errs
}
