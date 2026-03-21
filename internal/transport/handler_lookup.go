package transport

import (
	"net/http"

	"github.com/pitabwire/thesa/internal/search"
	"github.com/pitabwire/thesa/model"
)

func handleLookup(provider *search.LookupProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}
		lookupID := r.PathValue("lookupId")
		query := r.URL.Query().Get("q")

		resp, err := provider.GetLookup(r.Context(), rctx, lookupID, query)
		if err != nil {
			WriteError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, resp)
	}
}
