// Package capability resolves and caches user capabilities, and evaluates
// authorization policies using static configuration or OPA.
package capability

import (
	"sync"
	"time"

	"github.com/pitabwire/thesa/model"
)

type cacheEntry struct {
	caps    model.CapabilitySet
	expires time.Time
}

// Resolver implements model.CapabilityResolver with an in-memory cache.
type Resolver struct {
	evaluator model.PolicyEvaluator
	ttl       time.Duration
	mu        sync.RWMutex
	cache     map[string]cacheEntry
}

// NewResolver creates a new Resolver with the given evaluator and cache TTL.
func NewResolver(evaluator model.PolicyEvaluator, ttl time.Duration) *Resolver {
	return &Resolver{
		evaluator: evaluator,
		ttl:       ttl,
		cache:     make(map[string]cacheEntry),
	}
}

func cacheKey(rctx *model.RequestContext) string {
	return rctx.SubjectID + ":" + rctx.TenantID + ":" + rctx.PartitionID
}

// Resolve returns the full capability set for the given context. Results are
// cached for the configured TTL.
func (r *Resolver) Resolve(rctx *model.RequestContext) (model.CapabilitySet, error) {
	key := cacheKey(rctx)

	r.mu.RLock()
	if entry, ok := r.cache[key]; ok && time.Now().Before(entry.expires) {
		r.mu.RUnlock()
		return entry.caps, nil
	}
	r.mu.RUnlock()

	caps, err := r.evaluator.ResolveCapabilities(rctx)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.cache[key] = cacheEntry{caps: caps, expires: time.Now().Add(r.ttl)}
	r.mu.Unlock()

	return caps, nil
}

// Invalidate clears cached capabilities for the given user and tenant.
func (r *Resolver) Invalidate(subjectID, tenantID string) {
	prefix := subjectID + ":" + tenantID + ":"
	r.mu.Lock()
	for key := range r.cache {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(r.cache, key)
		}
	}
	r.mu.Unlock()
}
