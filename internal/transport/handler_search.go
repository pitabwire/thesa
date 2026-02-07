package transport

import (
	"net/http"

	"github.com/pitabwire/thesa/internal/search"
	"github.com/pitabwire/thesa/model"
)

func handleSearch(provider *search.SearchProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}
		caps := CapabilitiesFrom(r.Context())

		query := r.URL.Query().Get("q")
		pagination := model.Pagination{
			Page:     queryInt(r, "page", 1),
			PageSize: queryInt(r, "page_size", 20),
			Domain:   r.URL.Query().Get("domain"),
		}

		resp, err := provider.Search(r.Context(), rctx, caps, query, pagination)
		if err != nil {
			WriteError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, resp)
	}
}
