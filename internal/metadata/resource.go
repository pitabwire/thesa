package metadata

import (
	"context"
	"fmt"
	"strings"

	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/internal/openapi"
	"github.com/pitabwire/thesa/model"
)

// resourceIndex maps a resource type to its list page and detail operation.
type resourceIndex struct {
	listPage  *model.PageDefinition
	serviceID string
	// getOpID is the operation ID for fetching a single item (e.g., "getTenant").
	getOpID string
}

// ResourceProvider fetches resource data by type, delegating to the
// same backend invocation pipeline used by pages.
type ResourceProvider struct {
	registry *definition.Registry
	invokers *invoker.Registry
	oaIndex  *openapi.Index
}

// NewResourceProvider creates a ResourceProvider.
func NewResourceProvider(
	registry *definition.Registry,
	invokers *invoker.Registry,
	oaIndex *openapi.Index,
) *ResourceProvider {
	return &ResourceProvider{
		registry: registry,
		invokers: invokers,
		oaIndex:  oaIndex,
	}
}

// GetResourceList returns a paginated list of resources for the given type.
// The resource type is matched to a domain's list page (layout: table).
func (p *ResourceProvider) GetResourceList(
	ctx context.Context,
	rctx *model.RequestContext,
	caps model.CapabilitySet,
	resourceType string,
	params model.DataParams,
) (model.DataResponse, error) {
	idx := p.findResource(resourceType)
	if idx == nil || idx.listPage == nil {
		return model.DataResponse{}, model.NewNotFoundError(
			fmt.Sprintf("resource type %q not found", resourceType),
		)
	}

	page := idx.listPage
	if len(page.Capabilities) > 0 && !caps.HasAll(page.Capabilities...) {
		return model.DataResponse{}, model.NewForbiddenError(
			fmt.Sprintf("insufficient capabilities for resource %q", resourceType),
		)
	}

	if page.Table == nil {
		return model.DataResponse{}, model.NewBadRequestError(
			fmt.Sprintf("resource type %q has no data source", resourceType),
		)
	}

	ds := page.Table.DataSource
	binding := model.OperationBinding{
		Type:        "openapi",
		ServiceID:   ds.ServiceID,
		OperationID: ds.OperationID,
		Handler:     ds.Handler,
	}
	if ds.Handler != "" {
		binding.Type = "sdk"
	}

	input := buildDataInput(params)
	result, err := p.invokers.Invoke(ctx, rctx, binding, input)
	if err != nil {
		return model.DataResponse{}, err
	}

	return applyResponseMapping(result, ds.Mapping, params), nil
}

// GetResourceItem returns a single resource by type and ID.
// It finds the corresponding "get" operation for the resource's service
// and invokes it with the ID as a path parameter.
func (p *ResourceProvider) GetResourceItem(
	ctx context.Context,
	rctx *model.RequestContext,
	caps model.CapabilitySet,
	resourceType string,
	id string,
) (map[string]any, error) {
	idx := p.findResource(resourceType)
	if idx == nil {
		return nil, model.NewNotFoundError(
			fmt.Sprintf("resource type %q not found", resourceType),
		)
	}

	if idx.listPage != nil {
		if len(idx.listPage.Capabilities) > 0 && !caps.HasAll(idx.listPage.Capabilities...) {
			return nil, model.NewForbiddenError(
				fmt.Sprintf("insufficient capabilities for resource %q", resourceType),
			)
		}
	}

	if idx.getOpID == "" || idx.serviceID == "" {
		return nil, model.NewNotFoundError(
			fmt.Sprintf("no get operation found for resource type %q", resourceType),
		)
	}

	binding := model.OperationBinding{
		Type:        "openapi",
		ServiceID:   idx.serviceID,
		OperationID: idx.getOpID,
	}

	input := model.InvocationInput{
		Body: map[string]any{"id": id},
	}

	result, err := p.invokers.Invoke(ctx, rctx, binding, input)
	if err != nil {
		return nil, err
	}

	body, ok := result.Body.(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}

	return body, nil
}

// findResource searches the definition registry for a list page and get
// operation that match the given resource type. Resource types are matched
// by: exact page ID prefix (e.g., "tenants" matches "tenants.list"),
// or domain name.
func (p *ResourceProvider) findResource(resourceType string) *resourceIndex {
	for _, domain := range p.registry.AllDomains() {
		var listPage *model.PageDefinition
		var serviceID string

		for i := range domain.Pages {
			pg := &domain.Pages[i]

			// Match by page ID prefix (e.g., "tenants" matches "tenants.list").
			idPrefix := strings.SplitN(pg.ID, ".", 2)[0]
			domainMatch := domain.Domain == resourceType
			prefixMatch := idPrefix == resourceType

			if (domainMatch || prefixMatch) && pg.Table != nil && pg.Layout == "table" {
				listPage = pg
				serviceID = pg.Table.DataSource.ServiceID
				break
			}
		}

		if listPage == nil {
			continue
		}

		// Search for a "get" operation in the service's OpenAPI spec.
		getOpID := findGetOperation(p.oaIndex, serviceID, resourceType)

		return &resourceIndex{
			listPage:  listPage,
			serviceID: serviceID,
			getOpID:   getOpID,
		}
	}

	return nil
}

// findGetOperation searches a service's operations for one that looks like
// a "get single item" operation for the given resource type.
// It tries patterns like "getProfile", "getTenant", "getFile", etc.
func findGetOperation(idx *openapi.Index, serviceID, resourceType string) string {
	ops := idx.AllOperationIDs(serviceID)

	// Normalize resource type for matching: "profiles" → "profile", "tenants" → "tenant".
	singular := strings.TrimSuffix(resourceType, "s")

	for _, opID := range ops {
		lower := strings.ToLower(opID)
		// Match patterns like "getTenant", "getProfileById", "getFile".
		if strings.HasPrefix(lower, "get") && strings.Contains(lower, strings.ToLower(singular)) {
			return opID
		}
	}
	return ""
}
