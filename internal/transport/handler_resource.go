package transport

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/pitabwire/thesa/internal/metadata"
	"github.com/pitabwire/thesa/internal/search"
	"github.com/pitabwire/thesa/model"
)

// handleGetResource returns a paginated list of resources for the given type.
// The resource type maps to a domain's list page and its backend data source.
func handleGetResource(provider *metadata.ResourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}
		caps := CapabilitiesFrom(r.Context())
		resourceType := chi.URLParam(r, "resourceType")

		params := model.DataParams{
			Page:     queryInt(r, "page", 1),
			PageSize: queryInt(r, "page_size", 25),
			Sort:     r.URL.Query().Get("sort_field"),
			SortDir:  r.URL.Query().Get("sort_direction"),
			Filters:  queryMap(r, "filter"),
			Query:    r.URL.Query().Get("q"),
		}
		// Also accept "sort" / "sort_dir" as alternative parameter names.
		if params.Sort == "" {
			params.Sort = r.URL.Query().Get("sort")
		}
		if params.SortDir == "" {
			params.SortDir = r.URL.Query().Get("sort_dir")
		}

		data, err := provider.GetResourceList(r.Context(), rctx, caps, resourceType, params)
		if err != nil {
			WriteError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, data)
	}
}

// handleGetResourceItem returns a single resource item by type and ID.
func handleGetResourceItem(provider *metadata.ResourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}
		caps := CapabilitiesFrom(r.Context())
		resourceType := chi.URLParam(r, "resourceType")
		id := chi.URLParam(r, "id")

		item, err := provider.GetResourceItem(r.Context(), rctx, caps, resourceType, id)
		if err != nil {
			WriteError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, item)
	}
}

// handleResourceSearch searches resources of a specific type. Delegates to the
// search provider with a domain filter matching the resource type.
func handleResourceSearch(provider *search.SearchProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}
		caps := CapabilitiesFrom(r.Context())
		resourceType := chi.URLParam(r, "resourceType")

		query := r.URL.Query().Get("q")
		limit := queryInt(r, "limit", 20)

		pagination := model.Pagination{
			Page:     1,
			PageSize: limit,
			Domain:   resourceType,
		}

		resp, err := provider.Search(r.Context(), rctx, caps, query, pagination)
		if err != nil {
			WriteError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, resp)
	}
}

// handleGetSchema returns a schema derived from form or page definitions.
// The schema ID maps to a form ID (returns form fields) or page ID (returns
// table columns).
func handleGetSchema(provider *metadata.SchemaProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schemaID := chi.URLParam(r, "schemaId")

		schema, err := provider.GetSchema(schemaID)
		if err != nil {
			WriteError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, schema)
	}
}
