package metadata

import (
	"fmt"
	"strings"

	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/model"
)

// SchemaProvider derives schemas from form and page definitions. The schema ID
// can be a form ID (returns form field schema) or a page ID (returns table
// column schema).
type SchemaProvider struct {
	registry *definition.Registry
}

// NewSchemaProvider creates a SchemaProvider.
func NewSchemaProvider(registry *definition.Registry) *SchemaProvider {
	return &SchemaProvider{registry: registry}
}

// schemaField is a single field in the schema response, matching the
// Flutter UI's SchemaField model.
type schemaField struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Label       string            `json:"label,omitempty"`
	HelpText    string            `json:"helpText,omitempty"`
	Placeholder string            `json:"placeholder,omitempty"`
	Required    bool              `json:"required,omitempty"`
	Readonly    bool              `json:"readonly,omitempty"`
	Sortable    bool              `json:"sortable,omitempty"`
	Filterable  bool              `json:"filterable,omitempty"`
	Validation  *schemaValidation `json:"validation,omitempty"`
	Options     []schemaOption    `json:"options,omitempty"`
	Priority    int               `json:"priority,omitempty"`
	VisibleWhen *visibilityRule   `json:"visibleWhen,omitempty"`
	Format      string            `json:"format,omitempty"`
}

type schemaValidation struct {
	MinLength *int     `json:"minLength,omitempty"`
	MaxLength *int     `json:"maxLength,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Message   string   `json:"errorMessage,omitempty"`
}

type schemaOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type visibilityRule struct {
	Field  string `json:"field"`
	Equals string `json:"equals,omitempty"`
}

// schemaResponse is the JSON structure matching the Flutter UI's Schema model.
type schemaResponse struct {
	SchemaID string        `json:"schemaId"`
	Title    string        `json:"title,omitempty"`
	Fields   []schemaField `json:"fields"`
}

// GetSchema derives a schema from definitions. It first tries to find a
// matching form, then a page. Returns the schema or NOT_FOUND.
func (p *SchemaProvider) GetSchema(schemaID string) (schemaResponse, error) {
	// Try as a form ID first.
	if formDef, ok := p.registry.GetForm(schemaID); ok {
		return p.schemaFromForm(schemaID, formDef), nil
	}

	// Try as a page ID.
	if pageDef, ok := p.registry.GetPage(schemaID); ok {
		return p.schemaFromPage(schemaID, pageDef), nil
	}

	// Try fuzzy match: "tenants" might match "tenants.list" or "access-control.createTenant".
	if resp, ok := p.schemaByPrefix(schemaID); ok {
		return resp, nil
	}

	return schemaResponse{}, model.NewNotFoundError(
		fmt.Sprintf("schema %q not found", schemaID),
	)
}

// schemaFromForm builds a schema from a form definition's sections and fields.
func (p *SchemaProvider) schemaFromForm(id string, form model.FormDefinition) schemaResponse {
	resp := schemaResponse{
		SchemaID: id,
		Title:    form.Title,
		Fields:   []schemaField{},
	}

	for _, sec := range form.Sections {
		for _, f := range sec.Fields {
			sf := fieldDefToSchema(f)
			resp.Fields = append(resp.Fields, sf)
		}
	}

	return resp
}

// schemaFromPage builds a schema from a page's table columns or sections.
func (p *SchemaProvider) schemaFromPage(id string, page model.PageDefinition) schemaResponse {
	resp := schemaResponse{
		SchemaID: id,
		Title:    page.Title,
		Fields:   []schemaField{},
	}

	// If the page has a table, derive schema from columns.
	if page.Table != nil {
		for i, col := range page.Table.Columns {
			sf := schemaField{
				Name:       col.Field,
				Type:       mapColumnType(col.Type),
				Label:      col.Label,
				Sortable:   col.Sortable,
				Filterable: col.Filterable,
				Format:     col.Format,
				Priority:   priorityFromIndex(i),
			}
			resp.Fields = append(resp.Fields, sf)
		}
		return resp
	}

	// Fall back to page sections (detail pages).
	for _, sec := range page.Sections {
		for _, f := range sec.Fields {
			sf := fieldDefToSchema(f)
			resp.Fields = append(resp.Fields, sf)
		}
	}

	return resp
}

// schemaByPrefix tries to find a form or page whose ID starts with the given prefix.
func (p *SchemaProvider) schemaByPrefix(prefix string) (schemaResponse, bool) {
	for _, domain := range p.registry.AllDomains() {
		for _, form := range domain.Forms {
			if strings.HasPrefix(form.ID, prefix) {
				return p.schemaFromForm(form.ID, form), true
			}
		}
		for _, page := range domain.Pages {
			if strings.HasPrefix(page.ID, prefix) {
				return p.schemaFromPage(page.ID, page), true
			}
		}
	}
	return schemaResponse{}, false
}

// fieldDefToSchema converts a FieldDefinition to a schemaField.
func fieldDefToSchema(f model.FieldDefinition) schemaField {
	sf := schemaField{
		Name:        f.Field,
		Type:        mapFieldType(f.Type),
		Label:       f.Label,
		Required:    f.Required,
		Readonly:    f.ReadOnly == "true" || f.ReadOnly == "always",
		Placeholder: f.Placeholder,
		HelpText:    f.HelpText,
		Format:      f.Format,
	}

	if f.Validation != nil {
		sf.Validation = &schemaValidation{
			MinLength: f.Validation.MinLength,
			MaxLength: f.Validation.MaxLength,
			Min:       f.Validation.Min,
			Max:       f.Validation.Max,
			Pattern:   f.Validation.Pattern,
			Message:   f.Validation.Message,
		}
	}

	if f.Lookup != nil && len(f.Lookup.Static) > 0 {
		for _, opt := range f.Lookup.Static {
			sf.Options = append(sf.Options, schemaOption{
				Value: opt.Value,
				Label: opt.Label,
			})
		}
	}

	// Convert depends_on to visibility rule (simplified: first dependency only).
	if len(f.DependsOn) > 0 {
		dep := f.DependsOn[0]
		sf.VisibleWhen = &visibilityRule{
			Field:  dep.Field,
			Equals: dep.Value,
		}
	}

	return sf
}

// mapFieldType maps BFF field types to Flutter schema FieldType names.
func mapFieldType(t string) string {
	switch t {
	case "text":
		return "string"
	case "textarea":
		return "string"
	case "number":
		return "number"
	case "currency":
		return "money"
	case "select":
		return "enum"
	case "reference":
		return "reference"
	case "date_range":
		return "date"
	case "file":
		return "file"
	case "badge":
		return "enum"
	default:
		return t
	}
}

// mapColumnType maps table column types to schema field types.
func mapColumnType(t string) string {
	switch t {
	case "text":
		return "string"
	case "badge":
		return "enum"
	case "currency":
		return "money"
	case "datetime":
		return "datetime"
	case "date_range":
		return "date"
	default:
		return t
	}
}

// priorityFromIndex returns a priority (1-5) based on column position.
// First columns are higher priority (always visible in responsive tables).
func priorityFromIndex(i int) int {
	switch {
	case i < 2:
		return 1
	case i < 4:
		return 2
	case i < 6:
		return 3
	default:
		return 4
	}
}
