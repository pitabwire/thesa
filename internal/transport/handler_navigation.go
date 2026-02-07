package transport

import (
	"net/http"

	"github.com/pitabwire/thesa/internal/metadata"
	"github.com/pitabwire/thesa/model"
)

func handleNavigation(menu *metadata.MenuProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}
		caps := CapabilitiesFrom(r.Context())

		tree, err := menu.GetMenu(r.Context(), rctx, caps)
		if err != nil {
			WriteError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, tree)
	}
}
