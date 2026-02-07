package transport

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/pitabwire/thesa/internal/metadata"
	"github.com/pitabwire/thesa/model"
)

func handleGetForm(forms *metadata.FormProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}
		caps := CapabilitiesFrom(r.Context())
		formID := chi.URLParam(r, "formId")

		desc, err := forms.GetForm(r.Context(), rctx, caps, formID)
		if err != nil {
			WriteError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, desc)
	}
}

func handleGetFormData(forms *metadata.FormProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}
		caps := CapabilitiesFrom(r.Context())
		formID := chi.URLParam(r, "formId")

		// Collect all query params as the params map for path/route param forwarding.
		params := make(map[string]string)
		for key, values := range r.URL.Query() {
			if len(values) > 0 {
				params[key] = values[0]
			}
		}

		data, err := forms.GetFormData(r.Context(), rctx, caps, formID, params)
		if err != nil {
			WriteError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, data)
	}
}
