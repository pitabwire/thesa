package model

import (
	"context"
	"strings"
)

// CapabilitySet is a set of capabilities granted to a user. Each key is a
// capability string (e.g. "orders:list:view") and may include wildcards
// (e.g. "orders:*").
type CapabilitySet map[string]bool

// Has returns true if the set contains the exact capability or a wildcard
// that matches it.
func (cs CapabilitySet) Has(cap string) bool {
	if cs[cap] {
		return true
	}
	// Check wildcard matches: "orders:*" matches "orders:list:view",
	// "*" matches everything.
	for pattern := range cs {
		if matchWildcard(pattern, cap) {
			return true
		}
	}
	return false
}

// HasAll returns true if the set matches all given capabilities (including
// via wildcards).
func (cs CapabilitySet) HasAll(caps ...string) bool {
	for _, cap := range caps {
		if !cs.Has(cap) {
			return false
		}
	}
	return true
}

// HasAny returns true if the set matches at least one of the given
// capabilities (including via wildcards).
func (cs CapabilitySet) HasAny(caps ...string) bool {
	for _, cap := range caps {
		if cs.Has(cap) {
			return true
		}
	}
	return false
}

// matchWildcard returns true if pattern (which may end in "*") matches cap.
// Examples:
//
//	"*"             matches anything
//	"orders:*"      matches "orders:list:view"
//	"orders:list:*" matches "orders:list:view"
//	"orders:list"   does NOT match "orders:list:view" (exact only, no wildcard)
func matchWildcard(pattern, cap string) bool {
	if pattern == "*" {
		return true
	}
	if !strings.HasSuffix(pattern, ":*") {
		return false
	}
	prefix := pattern[:len(pattern)-1] // "orders:*" → "orders:"
	return strings.HasPrefix(cap, prefix)
}

// CapabilityResolver resolves the full capability set for a request context.
type CapabilityResolver interface {
	// Resolve returns all capabilities for the given subject/tenant/partition.
	Resolve(ctx context.Context, rctx *RequestContext) (CapabilitySet, error)

	// Invalidate clears cached capabilities for the given user and tenant.
	Invalidate(subjectID, tenantID string)
}

// PolicyEvaluator is the backend implementation that resolves capabilities
// from the authorization service.
type PolicyEvaluator interface {
	// ResolveCapabilities returns the full capability set for the given context
	// by querying the authorization service.
	ResolveCapabilities(ctx context.Context, rctx *RequestContext) (CapabilitySet, error)
}
