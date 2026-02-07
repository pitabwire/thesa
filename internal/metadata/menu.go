package metadata

import (
	"context"
	"log/slog"
	"sort"

	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/model"
)

// MenuProvider builds a NavigationTree from definitions filtered by capabilities.
type MenuProvider struct {
	registry *definition.Registry
	invokers *invoker.Registry
}

// NewMenuProvider creates a MenuProvider backed by the given definition registry
// and invoker registry. The invoker registry is used for optional badge
// resolution and may be nil if badges are not needed.
func NewMenuProvider(registry *definition.Registry, invokers *invoker.Registry) *MenuProvider {
	return &MenuProvider{
		registry: registry,
		invokers: invokers,
	}
}

// GetMenu builds the navigation tree from all domain definitions, filtering
// items by the given capability set. Badge counts are resolved via the invoker
// registry on a best-effort basis (failures are logged and badges omitted).
func (p *MenuProvider) GetMenu(ctx context.Context, rctx *model.RequestContext, caps model.CapabilitySet) (model.NavigationTree, error) {
	domains := p.registry.AllDomains()

	var nodes []model.NavigationNode
	for _, domain := range domains {
		nav := domain.Navigation

		// Check domain-level capabilities.
		if len(nav.Capabilities) > 0 && !caps.HasAll(nav.Capabilities...) {
			continue
		}

		node := model.NavigationNode{
			ID:    domain.Domain,
			Label: nav.Label,
			Icon:  nav.Icon,
		}

		// Build children, filtering by capabilities and sorting by order.
		var children []orderedChild
		for _, child := range nav.Children {
			if len(child.Capabilities) > 0 && !caps.HasAll(child.Capabilities...) {
				continue
			}

			childNode := model.NavigationNode{
				ID:    child.PageID,
				Label: child.Label,
				Icon:  child.Icon,
				Route: child.Route,
			}

			// Resolve badge if configured.
			if child.Badge != nil && child.Badge.OperationID != "" {
				badge := p.resolveBadge(ctx, rctx, domain, child)
				if badge != nil {
					childNode.Badge = badge
				}
			}

			children = append(children, orderedChild{
				order: child.Order,
				node:  childNode,
			})
		}

		// Sort children by their order field.
		sort.Slice(children, func(i, j int) bool {
			return children[i].order < children[j].order
		})

		node.Children = make([]model.NavigationNode, len(children))
		for i, c := range children {
			node.Children[i] = c.node
		}

		nodes = append(nodes, node)
	}

	// Sort top-level nodes by their navigation order.
	sort.Slice(nodes, func(i, j int) bool {
		// Look up the order from the domain definitions.
		orderI := p.domainOrder(domains, nodes[i].ID)
		orderJ := p.domainOrder(domains, nodes[j].ID)
		return orderI < orderJ
	})

	return model.NavigationTree{Items: nodes}, nil
}

// orderedChild pairs a navigation node with its sort order.
type orderedChild struct {
	order int
	node  model.NavigationNode
}

// domainOrder returns the navigation order for the domain with the given ID.
func (p *MenuProvider) domainOrder(domains []model.DomainDefinition, domainID string) int {
	for _, d := range domains {
		if d.Domain == domainID {
			return d.Navigation.Order
		}
	}
	return 0
}

// resolveBadge attempts to fetch a badge count by invoking the badge operation.
// Returns nil if the invoker is not configured or if the invocation fails.
func (p *MenuProvider) resolveBadge(
	ctx context.Context,
	rctx *model.RequestContext,
	domain model.DomainDefinition,
	child model.NavigationChildDefinition,
) *model.BadgeDescriptor {
	if p.invokers == nil {
		return nil
	}

	badge := child.Badge
	binding := model.OperationBinding{
		Type:        "openapi",
		ServiceID:   domain.Domain,
		OperationID: badge.OperationID,
	}

	result, err := p.invokers.Invoke(ctx, rctx, binding, model.InvocationInput{})
	if err != nil {
		slog.Debug("menu: badge resolution failed",
			"domain", domain.Domain,
			"operation", badge.OperationID,
			"error", err,
		)
		return nil
	}

	// Extract the count from the response body.
	count := extractBadgeCount(result.Body, badge.Field)
	if count <= 0 {
		return nil
	}

	return &model.BadgeDescriptor{
		Count: count,
		Style: badge.Style,
	}
}

// extractBadgeCount extracts an integer count from a response body using the
// given field name. Returns 0 if the field is not found or not a number.
func extractBadgeCount(body any, field string) int {
	if field == "" || body == nil {
		return 0
	}
	m, ok := body.(map[string]any)
	if !ok {
		return 0
	}
	val, exists := m[field]
	if !exists {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 0
	}
}
