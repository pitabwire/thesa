package search

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/model"
)

// LookupProvider resolves LookupDefinitions to option lists with caching.
type LookupProvider struct {
	registry   *definition.Registry
	invokers   *invoker.Registry
	defaultTTL time.Duration
	maxEntries int

	mu    sync.RWMutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	options   []model.OptionDescriptor
	expiresAt time.Time
}

// NewLookupProvider creates a new LookupProvider.
func NewLookupProvider(
	registry *definition.Registry,
	invokers *invoker.Registry,
	defaultTTL time.Duration,
	maxEntries int,
) *LookupProvider {
	if defaultTTL <= 0 {
		defaultTTL = 5 * time.Minute
	}
	if maxEntries <= 0 {
		maxEntries = 1000
	}
	return &LookupProvider{
		registry:   registry,
		invokers:   invokers,
		defaultTTL: defaultTTL,
		maxEntries: maxEntries,
		cache:      make(map[string]cacheEntry),
	}
}

// GetLookup resolves a lookup definition to an option list.
func (lp *LookupProvider) GetLookup(
	ctx context.Context,
	rctx *model.RequestContext,
	lookupID string,
	query string,
) (model.LookupResponse, error) {
	def, ok := lp.registry.GetLookup(lookupID)
	if !ok {
		return model.LookupResponse{}, model.NewNotFoundError(
			fmt.Sprintf("lookup %q not found", lookupID),
		)
	}

	// Build cache key based on scope.
	cacheKey := lp.buildCacheKey(def, rctx)

	// Check cache.
	if options, hit := lp.getFromCache(cacheKey); hit {
		filtered := filterOptions(options, query)
		return model.LookupResponse{
			Data: model.LookupPayload{Options: filtered},
			Meta: map[string]any{"cached": true},
		}, nil
	}

	// Cache miss: invoke backend.
	options, err := lp.fetchFromBackend(ctx, rctx, def)
	if err != nil {
		return model.LookupResponse{}, err
	}

	// Determine TTL.
	ttl := lp.defaultTTL
	if def.Cache != nil && def.Cache.TTL != "" {
		if parsed, parseErr := time.ParseDuration(def.Cache.TTL); parseErr == nil {
			ttl = parsed
		}
	}

	// Store in cache.
	lp.putInCache(cacheKey, options, ttl)

	// Apply query filter.
	filtered := filterOptions(options, query)

	return model.LookupResponse{
		Data: model.LookupPayload{Options: filtered},
		Meta: map[string]any{"cached": false},
	}, nil
}

// buildCacheKey constructs a cache key scoped to the lookup and tenant context.
func (lp *LookupProvider) buildCacheKey(def model.LookupDefinition, rctx *model.RequestContext) string {
	scope := "global"
	if def.Cache != nil && def.Cache.Scope != "" {
		scope = def.Cache.Scope
	}

	switch scope {
	case "tenant":
		return fmt.Sprintf("lookup:%s:%s", def.ID, rctx.TenantID)
	case "partition":
		return fmt.Sprintf("lookup:%s:%s:%s", def.ID, rctx.TenantID, rctx.PartitionID)
	default: // "global"
		return fmt.Sprintf("lookup:%s", def.ID)
	}
}

// getFromCache returns cached options if the entry exists and hasn't expired.
func (lp *LookupProvider) getFromCache(key string) ([]model.OptionDescriptor, bool) {
	lp.mu.RLock()
	defer lp.mu.RUnlock()

	entry, exists := lp.cache[key]
	if !exists || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.options, true
}

// putInCache stores options in the cache with TTL.
func (lp *LookupProvider) putInCache(key string, options []model.OptionDescriptor, ttl time.Duration) {
	lp.mu.Lock()
	defer lp.mu.Unlock()

	// Evict expired entries if at capacity.
	if len(lp.cache) >= lp.maxEntries {
		lp.evictExpired()
	}

	lp.cache[key] = cacheEntry{
		options:   options,
		expiresAt: time.Now().Add(ttl),
	}
}

// evictExpired removes expired entries. Must be called with mu held.
func (lp *LookupProvider) evictExpired() {
	now := time.Now()
	for k, v := range lp.cache {
		if now.After(v.expiresAt) {
			delete(lp.cache, k)
		}
	}
}

// Invalidate removes a specific cache entry.
func (lp *LookupProvider) Invalidate(lookupID, tenantID string) {
	lp.mu.Lock()
	defer lp.mu.Unlock()

	// Remove all matching entries (global, tenant, partition scopes).
	for k := range lp.cache {
		if strings.HasPrefix(k, "lookup:"+lookupID) {
			if tenantID == "" || strings.Contains(k, tenantID) {
				delete(lp.cache, k)
			}
		}
	}
}

// CacheLen returns the number of entries in the cache. For testing.
func (lp *LookupProvider) CacheLen() int {
	lp.mu.RLock()
	defer lp.mu.RUnlock()
	return len(lp.cache)
}

// fetchFromBackend invokes the lookup operation and maps results.
func (lp *LookupProvider) fetchFromBackend(
	ctx context.Context,
	rctx *model.RequestContext,
	def model.LookupDefinition,
) ([]model.OptionDescriptor, error) {
	result, err := lp.invokers.Invoke(ctx, rctx, def.Operation, model.InvocationInput{})
	if err != nil {
		return nil, fmt.Errorf("lookup %q: %w", def.ID, err)
	}

	if result.StatusCode < 200 || result.StatusCode >= 300 {
		return nil, fmt.Errorf("lookup %q: backend returned status %d", def.ID, result.StatusCode)
	}

	return mapLookupResults(result.Body, def), nil
}

// mapLookupResults transforms the backend response into OptionDescriptors.
func mapLookupResults(body any, def model.LookupDefinition) []model.OptionDescriptor {
	// Try body as a slice directly.
	items := extractLookupItems(body)
	if items == nil {
		return nil
	}

	options := make([]model.OptionDescriptor, 0, len(items))
	for _, item := range items {
		label := getString(item, def.LabelField)
		value := getString(item, def.ValueField)
		if label == "" && value == "" {
			continue
		}
		options = append(options, model.OptionDescriptor{
			Label: label,
			Value: value,
		})
	}
	return options
}

// extractLookupItems tries to extract a slice of maps from the response body.
func extractLookupItems(body any) []map[string]any {
	// Direct array.
	if arr, ok := body.([]any); ok {
		return toMapSlice(arr)
	}
	// Object with "data" array.
	if m, ok := body.(map[string]any); ok {
		if data, exists := m["data"]; exists {
			if arr, ok := data.([]any); ok {
				return toMapSlice(arr)
			}
		}
		// Object with "items" array.
		if items, exists := m["items"]; exists {
			if arr, ok := items.([]any); ok {
				return toMapSlice(arr)
			}
		}
	}
	return nil
}

// filterOptions filters options by query (case-insensitive match on label).
func filterOptions(options []model.OptionDescriptor, query string) []model.OptionDescriptor {
	if query == "" {
		return options
	}

	q := strings.ToLower(query)
	var filtered []model.OptionDescriptor
	for _, opt := range options {
		if strings.Contains(strings.ToLower(opt.Label), q) {
			filtered = append(filtered, opt)
		}
	}
	return filtered
}
