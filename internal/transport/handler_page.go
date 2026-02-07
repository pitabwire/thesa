package transport

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/pitabwire/thesa/internal/metadata"
	"github.com/pitabwire/thesa/model"
)

func handleGetPage(pages *metadata.PageProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}
		caps := CapabilitiesFrom(r.Context())
		pageID := chi.URLParam(r, "pageId")

		desc, err := pages.GetPage(r.Context(), rctx, caps, pageID)
		if err != nil {
			WriteError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, desc)
	}
}

func handleGetPageData(pages *metadata.PageProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}
		caps := CapabilitiesFrom(r.Context())
		pageID := chi.URLParam(r, "pageId")

		params := model.DataParams{
			Page:     queryInt(r, "page", 1),
			PageSize: queryInt(r, "page_size", 25),
			Sort:     r.URL.Query().Get("sort"),
			SortDir:  r.URL.Query().Get("sort_dir"),
			Filters:  queryMap(r, "filter"),
			Query:    r.URL.Query().Get("q"),
		}

		data, err := pages.GetPageData(r.Context(), rctx, caps, pageID, params)
		if err != nil {
			WriteError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, data)
	}
}

// queryInt extracts an integer query param with a default.
func queryInt(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// queryMap extracts all query params with a given prefix as a map.
// e.g., filter[status]=active → {"status": "active"}
func queryMap(r *http.Request, prefix string) map[string]string {
	result := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(key) > len(prefix)+2 && key[:len(prefix)+1] == prefix+"[" && key[len(key)-1] == ']' {
			field := key[len(prefix)+1 : len(key)-1]
			if len(values) > 0 {
				result[field] = values[0]
			}
		}
	}
	return result
}
