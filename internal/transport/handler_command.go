package transport

import (
	"encoding/json"
	"net/http"

	"github.com/pitabwire/thesa/internal/command"
	"github.com/pitabwire/thesa/model"
)

func handleCommand(executor *command.CommandExecutor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}
		caps := CapabilitiesFrom(r.Context())
		commandID := r.PathValue("commandId")

		var input model.CommandInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			WriteError(w, model.NewBadRequestError("invalid JSON body"))
			return
		}

		resp, err := executor.Execute(r.Context(), rctx, caps, commandID, input)
		if err != nil {
			WriteError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, resp)
	}
}
