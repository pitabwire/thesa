package search

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/model"
)

// SearchProvider orchestrates global search across all registered domains.
type SearchProvider struct {
	registry           *definition.Registry
	invokers           *invoker.Registry
	timeoutPerProvider time.Duration
	maxResultsDefault  int
}

// NewSearchProvider creates a new SearchProvider.
func NewSearchProvider(
	registry *definition.Registry,
	invokers *invoker.Registry,
	timeoutPerProvider time.Duration,
	maxResultsPerProvider int,
) *SearchProvider {
	if timeoutPerProvider <= 0 {
		timeoutPerProvider = 3 * time.Second
	}
	if maxResultsPerProvider <= 0 {
		maxResultsPerProvider = 50
	}
	return &SearchProvider{
		registry:           registry,
		invokers:           invokers,
		timeoutPerProvider: timeoutPerProvider,
		maxResultsDefault:  maxResultsPerProvider,
	}
}

// providerResult collects the outcome of a single search provider invocation.
type providerResult struct {
	ProviderID string
	Results    []model.SearchResult
	Status     string // "ok", "timeout", "error"
}

// Search executes a federated search across all eligible providers.
func (sp *SearchProvider) Search(
	ctx context.Context,
	rctx *model.RequestContext,
	caps model.CapabilitySet,
	query string,
	pagination model.Pagination,
) (model.SearchResponse, error) {
	// 1. Validate query.
	if len(query) < 2 {
		return model.SearchResponse{}, model.NewBadRequestError(
			"Search query must be at least 2 characters",
		)
	}

	// 2. Normalize pagination.
	if pagination.PageSize <= 0 {
		pagination.PageSize = 20
	}
	if pagination.PageSize > 50 {
		pagination.PageSize = 50
	}
	if pagination.Page <= 0 {
		pagination.Page = 1
	}

	// 3. Get all search definitions and filter by capability and domain.
	allDefs := sp.registry.AllSearches()
	var eligible []model.SearchDefinition
	for _, def := range allDefs {
		if pagination.Domain != "" && def.Domain != pagination.Domain {
			continue
		}
		if len(def.Capabilities) > 0 && !caps.HasAll(def.Capabilities...) {
			continue
		}
		eligible = append(eligible, def)
	}

	// 4. Execute providers in parallel.
	startTime := time.Now()
	results := sp.executeProviders(ctx, rctx, eligible, query)

	// 5. Merge all results.
	var merged []model.SearchResult
	providers := make(map[string]string, len(results))
	for _, r := range results {
		providers[r.ProviderID] = r.Status
		merged = append(merged, r.Results...)
	}

	// 6. Deduplicate by route + id (keep highest score).
	merged = deduplicate(merged)

	// 7. Sort by score descending.
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	totalCount := len(merged)

	// 8. Apply pagination.
	offset := (pagination.Page - 1) * pagination.PageSize
	if offset >= len(merged) {
		merged = nil
	} else {
		end := offset + pagination.PageSize
		if end > len(merged) {
			end = len(merged)
		}
		merged = merged[offset:end]
	}

	queryTimeMs := time.Since(startTime).Milliseconds()

	return model.SearchResponse{
		Data: model.SearchPayload{
			Results:    merged,
			TotalCount: totalCount,
			Query:      query,
		},
		Meta: map[string]any{
			"providers":     providers,
			"query_time_ms": queryTimeMs,
		},
	}, nil
}

// executeProviders runs all eligible providers concurrently and collects results.
func (sp *SearchProvider) executeProviders(
	ctx context.Context,
	rctx *model.RequestContext,
	defs []model.SearchDefinition,
	query string,
) []providerResult {
	if len(defs) == 0 {
		return nil
	}

	ch := make(chan providerResult, len(defs))
	var wg sync.WaitGroup

	for _, def := range defs {
		wg.Add(1)
		go func(d model.SearchDefinition) {
			defer wg.Done()
			ch <- sp.executeProvider(ctx, rctx, d, query)
		}(def)
	}

	// Close channel when all goroutines complete.
	go func() {
		wg.Wait()
		close(ch)
	}()

	var results []providerResult
	for r := range ch {
		results = append(results, r)
	}
	return results
}

// executeProvider runs a single search provider with a timeout.
func (sp *SearchProvider) executeProvider(
	ctx context.Context,
	rctx *model.RequestContext,
	def model.SearchDefinition,
	query string,
) providerResult {
	// Per-provider timeout.
	ctx, cancel := context.WithTimeout(ctx, sp.timeoutPerProvider)
	defer cancel()

	// Build invocation input with query.
	input := model.InvocationInput{
		QueryParams: map[string]string{"q": query},
	}

	// Invoke backend.
	result, err := sp.invokers.Invoke(ctx, rctx, def.Operation, input)
	if err != nil {
		status := "error"
		if ctx.Err() == context.DeadlineExceeded {
			status = "timeout"
		}
		return providerResult{ProviderID: def.ID, Status: status}
	}

	if result.StatusCode < 200 || result.StatusCode >= 300 {
		return providerResult{ProviderID: def.ID, Status: "error"}
	}

	// Map results.
	maxResults := def.MaxResults
	if maxResults <= 0 {
		maxResults = sp.maxResultsDefault
	}

	items := extractItems(result.Body, def.ResultMapping.ItemsPath)
	mapped := mapResults(items, def, maxResults)

	return providerResult{
		ProviderID: def.ID,
		Results:    mapped,
		Status:     "ok",
	}
}

// extractItems navigates a dot-separated path in the response body to find
// the results array.
func extractItems(body any, itemsPath string) []map[string]any {
	if itemsPath == "" {
		// Try the body directly as an array.
		if arr, ok := body.([]any); ok {
			return toMapSlice(arr)
		}
		return nil
	}

	current := body
	for _, part := range strings.Split(itemsPath, ".") {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[part]
	}

	arr, ok := current.([]any)
	if !ok {
		return nil
	}
	return toMapSlice(arr)
}

// toMapSlice converts []any to []map[string]any, skipping non-map items.
func toMapSlice(arr []any) []map[string]any {
	result := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			result = append(result, m)
		}
	}
	return result
}

// mapResults transforms raw backend items into scored SearchResults.
func mapResults(items []map[string]any, def model.SearchDefinition, maxResults int) []model.SearchResult {
	if len(items) > maxResults {
		items = items[:maxResults]
	}

	weight := def.Weight
	if weight <= 0 {
		weight = 1
	}

	total := len(items)
	results := make([]model.SearchResult, 0, total)
	for i, item := range items {
		// Position score: 1.0 at top, 0.5 at bottom.
		positionScore := 1.0
		if total > 1 {
			positionScore = 1.0 - (float64(i) / float64(total) * 0.5)
		}
		score := float64(weight) * positionScore

		id := getString(item, def.ResultMapping.IDField)
		route := resolveRoute(def.ResultMapping.Route, id)

		results = append(results, model.SearchResult{
			ID:       id,
			Title:    getString(item, def.ResultMapping.TitleField),
			Subtitle: getString(item, def.ResultMapping.SubtitleField),
			Category: def.Domain,
			Icon:     getString(item, def.ResultMapping.IconField),
			Route:    route,
			Score:    score,
		})
	}
	return results
}

// getString extracts a string value from a map, returning "" if not found.
func getString(m map[string]any, key string) string {
	if key == "" {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

// resolveRoute replaces {id} in the route template with the actual ID.
func resolveRoute(template, id string) string {
	return strings.ReplaceAll(template, "{id}", id)
}

// deduplicate removes duplicate results (same route + id), keeping the one
// with the highest score.
func deduplicate(results []model.SearchResult) []model.SearchResult {
	if len(results) == 0 {
		return results
	}

	seen := make(map[string]int, len(results)) // key → index in output
	var output []model.SearchResult

	for _, r := range results {
		key := r.Route + "|" + r.ID
		if idx, exists := seen[key]; exists {
			// Keep the one with the higher score.
			if r.Score > output[idx].Score {
				output[idx] = r
			}
		} else {
			seen[key] = len(output)
			output = append(output, r)
		}
	}
	return output
}
