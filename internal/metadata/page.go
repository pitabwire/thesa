package metadata

import (
	"context"
	"fmt"
	"strings"

	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/model"
)

// PageProvider resolves PageDefinitions into PageDescriptors and fetches
// page data from backends.
type PageProvider struct {
	registry *definition.Registry
	invokers *invoker.Registry
	actions  *ActionProvider
}

// NewPageProvider creates a PageProvider backed by the given registries.
func NewPageProvider(registry *definition.Registry, invokers *invoker.Registry, actions *ActionProvider) *PageProvider {
	return &PageProvider{
		registry: registry,
		invokers: invokers,
		actions:  actions,
	}
}

// GetPage resolves a PageDescriptor from the definition, filtering by
// capabilities. Returns an error with code NOT_FOUND or FORBIDDEN.
func (p *PageProvider) GetPage(
	ctx context.Context,
	rctx *model.RequestContext,
	caps model.CapabilitySet,
	pageID string,
) (model.PageDescriptor, error) {
	pageDef, ok := p.registry.GetPage(pageID)
	if !ok {
		return model.PageDescriptor{}, model.NewNotFoundError(
			fmt.Sprintf("page %q not found", pageID),
		)
	}

	if len(pageDef.Capabilities) > 0 && !caps.HasAll(pageDef.Capabilities...) {
		return model.PageDescriptor{}, model.NewForbiddenError(
			fmt.Sprintf("insufficient capabilities for page %q", pageID),
		)
	}

	desc := model.PageDescriptor{
		ID:              pageDef.ID,
		Title:           pageDef.Title,
		Route:           pageDef.Route,
		Layout:          pageDef.Layout,
		RefreshInterval: pageDef.RefreshInterval,
	}

	// Resolve breadcrumb.
	for _, b := range pageDef.Breadcrumb {
		desc.Breadcrumb = append(desc.Breadcrumb, model.BreadcrumbDescriptor{
			Label: b.Label,
			Route: b.Route,
		})
	}

	// Resolve table.
	if pageDef.Table != nil {
		desc.Table = p.resolveTable(caps, pageDef.Table, pageDef.ID)
	}

	// Resolve sections.
	desc.Sections = p.resolveSections(caps, pageDef.Sections)

	// Resolve page-level actions.
	desc.Actions = p.actions.ResolveActions(caps, pageDef.Actions, nil)

	return desc, nil
}

// GetPageData fetches data for a list page by invoking the backend operation
// and applying response mapping.
func (p *PageProvider) GetPageData(
	ctx context.Context,
	rctx *model.RequestContext,
	caps model.CapabilitySet,
	pageID string,
	params model.DataParams,
) (model.DataResponse, error) {
	pageDef, ok := p.registry.GetPage(pageID)
	if !ok {
		return model.DataResponse{}, model.NewNotFoundError(
			fmt.Sprintf("page %q not found", pageID),
		)
	}

	if len(pageDef.Capabilities) > 0 && !caps.HasAll(pageDef.Capabilities...) {
		return model.DataResponse{}, model.NewForbiddenError(
			fmt.Sprintf("insufficient capabilities for page %q", pageID),
		)
	}

	if pageDef.Table == nil {
		return model.DataResponse{}, model.NewBadRequestError(
			fmt.Sprintf("page %q has no data source", pageID),
		)
	}

	ds := pageDef.Table.DataSource
	binding := model.OperationBinding{
		Type:        "openapi",
		ServiceID:   ds.ServiceID,
		OperationID: ds.OperationID,
		Handler:     ds.Handler,
	}
	if ds.Handler != "" {
		binding.Type = "sdk"
	}

	// Build invocation input from DataParams.
	input := buildDataInput(params)

	result, err := p.invokers.Invoke(ctx, rctx, binding, input)
	if err != nil {
		return model.DataResponse{}, err
	}

	// Apply response mapping.
	return applyResponseMapping(result, ds.Mapping, params), nil
}

// resolveTable builds a TableDescriptor from a TableDefinition, filtering
// by capabilities.
func (p *PageProvider) resolveTable(caps model.CapabilitySet, table *model.TableDefinition, pageID string) *model.TableDescriptor {
	desc := &model.TableDescriptor{
		DataEndpoint: fmt.Sprintf("/api/pages/%s/data", pageID),
		DefaultSort:  table.DefaultSort,
		SortDir:      table.SortDir,
		PageSize:     table.PageSize,
		Selectable:   table.Selectable,
	}

	if desc.PageSize <= 0 {
		desc.PageSize = 25
	}

	// Resolve columns (no capability filtering for columns—visibility is
	// handled client-side via the "visible" expression).
	for _, col := range table.Columns {
		desc.Columns = append(desc.Columns, model.ColumnDescriptor{
			Field:     col.Field,
			Label:     col.Label,
			Type:      col.Type,
			Sortable:  col.Sortable,
			Format:    col.Format,
			Width:     col.Width,
			StatusMap: col.StatusMap,
		})
		if col.Link != nil {
			desc.Columns[len(desc.Columns)-1].Link = &model.LinkDescriptor{
				Route:  col.Link.Route,
				Params: col.Link.Params,
			}
		}
	}

	// Resolve filters.
	for _, f := range table.Filters {
		fd := model.FilterDescriptor{
			Field:    f.Field,
			Label:    f.Label,
			Type:     f.Type,
			Operator: f.Operator,
			Default:  f.Default,
		}
		if f.Options != nil && len(f.Options.Static) > 0 {
			for _, opt := range f.Options.Static {
				fd.Options = append(fd.Options, model.OptionDescriptor{
					Label: opt.Label,
					Value: opt.Value,
				})
			}
		}
		desc.Filters = append(desc.Filters, fd)
	}

	// Resolve row actions filtered by capabilities.
	desc.RowActions = p.actions.ResolveActions(caps, table.RowActions, nil)

	// Resolve bulk actions filtered by capabilities.
	desc.BulkActions = p.actions.ResolveActions(caps, table.BulkActions, nil)

	return desc
}

// resolveSections builds SectionDescriptors from SectionDefinitions,
// filtering by capabilities.
func (p *PageProvider) resolveSections(caps model.CapabilitySet, sections []model.SectionDefinition) []model.SectionDescriptor {
	var result []model.SectionDescriptor
	for _, sec := range sections {
		// Check section-level capabilities.
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

		// Resolve fields, filtering by visibility expression.
		for _, field := range sec.Fields {
			if field.Visibility != "" && field.Visibility == "hidden" {
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

// buildDataInput constructs an InvocationInput from DataParams.
func buildDataInput(params model.DataParams) model.InvocationInput {
	query := make(map[string]string)
	if params.Page > 0 {
		query["page"] = fmt.Sprintf("%d", params.Page)
	}
	if params.PageSize > 0 {
		query["page_size"] = fmt.Sprintf("%d", params.PageSize)
	}
	if params.Sort != "" {
		query["sort"] = params.Sort
	}
	if params.SortDir != "" {
		query["sort_dir"] = params.SortDir
	}
	if params.Query != "" {
		query["q"] = params.Query
	}
	for k, v := range params.Filters {
		query[k] = v
	}
	return model.InvocationInput{QueryParams: query}
}

// applyResponseMapping extracts items and total from the backend response
// using the configured mapping paths.
func applyResponseMapping(result model.InvocationResult, mapping model.ResponseMappingDefinition, params model.DataParams) model.DataResponse {
	body, ok := result.Body.(map[string]any)
	if !ok {
		return model.DataResponse{
			Data: model.DataPayload{
				Items:    []map[string]any{},
				Page:     params.Page,
				PageSize: params.PageSize,
			},
		}
	}

	// Extract items using items_path.
	rawItems := extractPath(body, mapping.ItemsPath)
	items := toMapSlice(rawItems)

	// Apply field_map renaming.
	if len(mapping.FieldMap) > 0 {
		items = applyFieldMap(items, mapping.FieldMap)
	}

	// Extract total count.
	total := 0
	if mapping.TotalPath != "" {
		if v, ok := extractPath(body, mapping.TotalPath).(float64); ok {
			total = int(v)
		}
	}
	if total == 0 {
		total = len(items)
	}

	return model.DataResponse{
		Data: model.DataPayload{
			Items:      items,
			TotalCount: total,
			Page:       params.Page,
			PageSize:   params.PageSize,
		},
	}
}

// extractPath navigates a dot-separated path in a map.
func extractPath(data map[string]any, path string) any {
	if path == "" || data == nil {
		return nil
	}
	parts := strings.Split(path, ".")
	var current any = data
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[part]
	}
	return current
}

// toMapSlice converts an any (expected []any of map[string]any) to []map[string]any.
func toMapSlice(v any) []map[string]any {
	if v == nil {
		return []map[string]any{}
	}
	slice, ok := v.([]any)
	if !ok {
		return []map[string]any{}
	}
	result := make([]map[string]any, 0, len(slice))
	for _, item := range slice {
		if m, ok := item.(map[string]any); ok {
			result = append(result, m)
		}
	}
	return result
}

// applyFieldMap renames fields in each item according to the field_map.
func applyFieldMap(items []map[string]any, fieldMap map[string]string) []map[string]any {
	result := make([]map[string]any, len(items))
	for i, item := range items {
		mapped := make(map[string]any, len(item))
		for k, v := range item {
			if newName, ok := fieldMap[k]; ok {
				mapped[newName] = v
			} else {
				mapped[k] = v
			}
		}
		result[i] = mapped
	}
	return result
}
