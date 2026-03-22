package capability

import (
	"context"
	"strings"

	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/util"

	"github.com/pitabwire/thesa/internal/config"
	"github.com/pitabwire/thesa/model"
)

// CapabilityCheck pairs a capability string with the Keto namespace
// it should be checked against.
type CapabilityCheck struct {
	Capability string
	Namespace  string
}

// KetoPolicyEvaluator resolves capabilities by checking each known capability
// against the authorization service (Ory Keto) using BatchCheck. This ensures
// that OPL rules, role hierarchies, and computed permissions are fully evaluated
// — not just direct relation tuples.
type KetoPolicyEvaluator struct {
	authorizer security.Authorizer
	checks     []CapabilityCheck
}

// NewKetoPolicyEvaluator creates an evaluator that verifies capabilities
// through Keto's full permission evaluation engine. The checks are derived
// from definition files: each capability string is mapped to the authorization
// namespace of the service that owns it.
func NewKetoPolicyEvaluator(authorizer security.Authorizer, checks []CapabilityCheck) *KetoPolicyEvaluator {
	return &KetoPolicyEvaluator{
		authorizer: authorizer,
		checks:     checks,
	}
}

// ResolveCapabilities checks all known capabilities for the user via BatchCheck.
// Each check goes through Keto's full OPL evaluation, so role-based and
// computed permissions are correctly resolved.
//
// Capability strings use colons (e.g. "tenants:view") but Keto OPL relations
// use underscores (e.g. "tenants_view"). The evaluator handles this mapping
// transparently.
func (e *KetoPolicyEvaluator) ResolveCapabilities(ctx context.Context, rctx *model.RequestContext) (model.CapabilitySet, error) {
	log := util.Log(ctx)

	if len(e.checks) == 0 {
		log.Warn("capability: no checks configured, returning empty capabilities")
		return make(model.CapabilitySet), nil
	}

	subject := security.SubjectRef{
		Namespace: security.NamespaceProfile,
		ID:        rctx.SubjectID,
	}
	tenancyPath := rctx.TenantID + "/" + rctx.PartitionID

	requests := make([]security.CheckRequest, len(e.checks))
	for i, chk := range e.checks {
		requests[i] = security.CheckRequest{
			Object: security.ObjectRef{
				Namespace: chk.Namespace,
				ID:        tenancyPath,
			},
			Permission: CapabilityToPermission(chk.Capability),
			Subject:    subject,
		}
	}

	log.Debug("capability: checking permissions",
		"subject_id", rctx.SubjectID,
		"tenancy_path", tenancyPath,
		"check_count", len(requests),
	)

	results, err := e.authorizer.BatchCheck(ctx, requests)
	if err != nil {
		log.Error("capability: batch check failed, falling back to individual checks",
			"error", err,
			"subject_id", rctx.SubjectID,
		)
		return e.fallbackIndividualChecks(ctx, requests)
	}

	caps := make(model.CapabilitySet)
	for i, result := range results {
		if result.Allowed {
			caps[e.checks[i].Capability] = true
		}
	}

	log.Debug("capability: resolved permissions",
		"subject_id", rctx.SubjectID,
		"tenancy_path", tenancyPath,
		"granted", len(caps),
		"total", len(e.checks),
	)

	return caps, nil
}

// fallbackIndividualChecks tries each check individually when BatchCheck fails.
func (e *KetoPolicyEvaluator) fallbackIndividualChecks(ctx context.Context, requests []security.CheckRequest) (model.CapabilitySet, error) {
	log := util.Log(ctx)
	caps := make(model.CapabilitySet)

	for i, req := range requests {
		result, err := e.authorizer.Check(ctx, req)
		if err != nil {
			log.Warn("capability: individual check failed",
				"permission", req.Permission,
				"namespace", req.Object.Namespace,
				"error", err,
			)
			continue
		}
		if result.Allowed {
			caps[e.checks[i].Capability] = true
		}
	}

	return caps, nil
}

// CapabilityToPermission converts a colon-separated capability string
// to a Keto-compatible permission name (underscores).
// Example: "tenants:view" → "tenants_view"
func CapabilityToPermission(capability string) string {
	return strings.ReplaceAll(capability, ":", "_")
}

// PermissionToCapability converts a Keto permission name back to a
// colon-separated capability string.
// Example: "tenants_view" → "tenants:view"
func PermissionToCapability(permission string) string {
	return strings.ReplaceAll(permission, "_", ":")
}

// CollectCapabilityChecks extracts all unique capability strings from the
// loaded definitions and maps each to its service's authorization namespace.
func CollectCapabilityChecks(domains []model.DomainDefinition, services map[string]config.ServiceConfig) []CapabilityCheck {
	namespaceByService := make(map[string]string)
	for svcID, svc := range services {
		if svc.AuthorizationNamespace != "" {
			namespaceByService[svcID] = svc.AuthorizationNamespace
		}
	}

	seen := make(map[string]bool)
	var checks []CapabilityCheck

	for _, domain := range domains {
		primaryService := findDomainService(domain)
		namespace := namespaceByService[primaryService]
		if namespace == "" {
			continue
		}

		addCaps := func(caps []string) {
			for _, cap := range caps {
				key := cap + "\x00" + namespace
				if !seen[key] {
					seen[key] = true
					checks = append(checks, CapabilityCheck{
						Capability: cap,
						Namespace:  namespace,
					})
				}
			}
		}

		addCaps(domain.Navigation.Capabilities)
		for _, child := range domain.Navigation.Children {
			addCaps(child.Capabilities)
		}

		for _, page := range domain.Pages {
			addCaps(page.Capabilities)
			for _, section := range page.Sections {
				addCaps(section.Capabilities)
			}
			for _, action := range page.Actions {
				addCaps(action.Capabilities)
			}
			if page.Table != nil {
				for _, action := range page.Table.RowActions {
					addCaps(action.Capabilities)
				}
				for _, action := range page.Table.BulkActions {
					addCaps(action.Capabilities)
				}
			}
		}

		for _, form := range domain.Forms {
			addCaps(form.Capabilities)
			for _, section := range form.Sections {
				addCaps(section.Capabilities)
			}
		}

		for _, cmd := range domain.Commands {
			addCaps(cmd.Capabilities)
		}

		for _, srch := range domain.Searches {
			addCaps(srch.Capabilities)
		}
	}

	return checks
}

// findDomainService determines the primary service_id for a domain by
// scanning its pages, commands, and searches for the first explicit
// service_id reference.
func findDomainService(domain model.DomainDefinition) string {
	for _, page := range domain.Pages {
		if page.Table != nil && page.Table.DataSource.ServiceID != "" {
			return page.Table.DataSource.ServiceID
		}
	}
	for _, cmd := range domain.Commands {
		if cmd.Operation.ServiceID != "" {
			return cmd.Operation.ServiceID
		}
	}
	for _, srch := range domain.Searches {
		if srch.Operation.ServiceID != "" {
			return srch.Operation.ServiceID
		}
	}
	for _, form := range domain.Forms {
		if form.LoadSource != nil && form.LoadSource.ServiceID != "" {
			return form.LoadSource.ServiceID
		}
	}
	return domain.Domain + "-svc"
}
