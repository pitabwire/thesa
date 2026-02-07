package metadata

import (
	"context"
	"fmt"

	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/model"
)

// FormProvider resolves FormDefinitions into FormDescriptors and fetches
// pre-populated form data from backends.
type FormProvider struct {
	registry *definition.Registry
	invokers *invoker.Registry
	actions  *ActionProvider
}

// NewFormProvider creates a FormProvider backed by the given registries.
func NewFormProvider(registry *definition.Registry, invokers *invoker.Registry, actions *ActionProvider) *FormProvider {
	return &FormProvider{
		registry: registry,
		invokers: invokers,
		actions:  actions,
	}
}

// GetForm resolves a FormDescriptor from the definition, filtering by
// capabilities. Returns an error with code NOT_FOUND or FORBIDDEN.
func (p *FormProvider) GetForm(
	ctx context.Context,
	rctx *model.RequestContext,
	caps model.CapabilitySet,
	formID string,
) (model.FormDescriptor, error) {
	formDef, ok := p.registry.GetForm(formID)
	if !ok {
		return model.FormDescriptor{}, model.NewNotFoundError(
			fmt.Sprintf("form %q not found", formID),
		)
	}

	if len(formDef.Capabilities) > 0 && !caps.HasAll(formDef.Capabilities...) {
		return model.FormDescriptor{}, model.NewForbiddenError(
			fmt.Sprintf("insufficient capabilities for form %q", formID),
		)
	}

	desc := model.FormDescriptor{
		ID:             formDef.ID,
		Title:          formDef.Title,
		SubmitEndpoint: fmt.Sprintf("/api/commands/%s", formDef.SubmitCommand),
		SuccessRoute:   formDef.SuccessRoute,
		SuccessMessage: formDef.SuccessMessage,
	}

	// Resolve sections.
	desc.Sections = p.resolveSections(caps, formDef.Sections)

	return desc, nil
}

// GetFormData fetches pre-populated data for a form by invoking the
// form's load_source operation and filtering the result to include
// only fields present in the form descriptor.
func (p *FormProvider) GetFormData(
	ctx context.Context,
	rctx *model.RequestContext,
	caps model.CapabilitySet,
	formID string,
	params map[string]string,
) (map[string]any, error) {
	formDef, ok := p.registry.GetForm(formID)
	if !ok {
		return nil, model.NewNotFoundError(
			fmt.Sprintf("form %q not found", formID),
		)
	}

	if len(formDef.Capabilities) > 0 && !caps.HasAll(formDef.Capabilities...) {
		return nil, model.NewForbiddenError(
			fmt.Sprintf("insufficient capabilities for form %q", formID),
		)
	}

	if formDef.LoadSource == nil {
		return nil, nil // No load source → empty form.
	}

	ds := formDef.LoadSource
	binding := model.OperationBinding{
		Type:        "openapi",
		ServiceID:   ds.ServiceID,
		OperationID: ds.OperationID,
		Handler:     ds.Handler,
	}
	if ds.Handler != "" {
		binding.Type = "sdk"
	}

	input := model.InvocationInput{
		PathParams: params,
	}

	result, err := p.invokers.Invoke(ctx, rctx, binding, input)
	if err != nil {
		return nil, err
	}

	body, ok := result.Body.(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}

	// Apply field_map if configured.
	if len(ds.Mapping.FieldMap) > 0 {
		body = renameFields(body, ds.Mapping.FieldMap)
	}

	// Filter to only include fields that appear in the resolved form.
	formFields := collectFormFields(formDef.Sections)
	return filterToFields(body, formFields), nil
}

// resolveSections builds SectionDescriptors from form SectionDefinitions,
// filtering by capabilities.
func (p *FormProvider) resolveSections(caps model.CapabilitySet, sections []model.SectionDefinition) []model.SectionDescriptor {
	var result []model.SectionDescriptor
	for _, sec := range sections {
		if len(sec.Capabilities) > 0 && !caps.HasAll(sec.Capabilities...) {
			continue
		}

		sd := model.SectionDescriptor{
			ID:          sec.ID,
			Title:       sec.Title,
			Layout:      sec.Layout,
			Columns:     sec.Columns,
			Collapsible: sec.Collapsible,
			Collapsed:   sec.Collapsed,
		}

		for _, field := range sec.Fields {
			if field.Visibility == "hidden" {
				continue
			}

			fd := model.FieldDescriptor{
				Field:       field.Field,
				Label:       field.Label,
				Type:        field.Type,
				ReadOnly:    field.ReadOnly == "true" || field.ReadOnly == "always",
				Required:    field.Required,
				Format:      field.Format,
				Placeholder: field.Placeholder,
				HelpText:    field.HelpText,
				Span:        field.Span,
			}

			if field.Validation != nil {
				fd.Validation = &model.ValidationDescriptor{
					MinLength: field.Validation.MinLength,
					MaxLength: field.Validation.MaxLength,
					Min:       field.Validation.Min,
					Max:       field.Validation.Max,
					Pattern:   field.Validation.Pattern,
					Message:   field.Validation.Message,
				}
			}

			// Resolve lookup inline options.
			if field.Lookup != nil && len(field.Lookup.Static) > 0 {
				for _, opt := range field.Lookup.Static {
					fd.Options = append(fd.Options, model.OptionDescriptor{
						Label: opt.Label,
						Value: opt.Value,
					})
				}
			}

			// Resolve depends_on.
			for _, dep := range field.DependsOn {
				fd.DependsOn = append(fd.DependsOn, model.FieldDependencyDescriptor{
					Field:     dep.Field,
					Condition: dep.Condition,
					Value:     dep.Value,
				})
			}

			sd.Fields = append(sd.Fields, fd)
		}

		result = append(result, sd)
	}
	return result
}

// collectFormFields returns a set of all field names across all sections.
func collectFormFields(sections []model.SectionDefinition) map[string]bool {
	fields := make(map[string]bool)
	for _, sec := range sections {
		for _, f := range sec.Fields {
			fields[f.Field] = true
		}
	}
	return fields
}

// filterToFields returns only the keys from data that are in the fields set.
func filterToFields(data map[string]any, fields map[string]bool) map[string]any {
	result := make(map[string]any, len(fields))
	for k, v := range data {
		if fields[k] {
			result[k] = v
		}
	}
	return result
}

// renameFields renames keys in data according to the field map.
func renameFields(data map[string]any, fieldMap map[string]string) map[string]any {
	result := make(map[string]any, len(data))
	for k, v := range data {
		if newName, ok := fieldMap[k]; ok {
			result[newName] = v
		} else {
			result[k] = v
		}
	}
	return result
}
