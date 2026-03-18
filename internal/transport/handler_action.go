package transport

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/pitabwire/thesa/internal/command"
	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/workflow"
	"github.com/pitabwire/thesa/model"
)

// handleAction looks up an action definition by ID and routes execution to
// the appropriate handler (command or workflow) based on the action type.
func handleAction(
	registry *definition.Registry,
	cmdExecutor *command.CommandExecutor,
	wfEngine *workflow.Engine,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}
		caps := CapabilitiesFrom(r.Context())
		actionID := chi.URLParam(r, "actionId")

		var body struct {
			Input map[string]any `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteError(w, model.NewBadRequestError("invalid JSON body"))
			return
		}

		// Find the action definition across all domains.
		actionDef := findActionInRegistry(registry, actionID)
		if actionDef == nil {
			WriteError(w, model.NewNotFoundError("action "+actionID+" not found"))
			return
		}

		switch actionDef.Type {
		case "command":
			if actionDef.CommandID == "" {
				WriteError(w, model.NewBadRequestError("action has no command_id"))
				return
			}
			cmdInput := model.CommandInput{
				Input: body.Input,
			}
			resp, err := cmdExecutor.Execute(r.Context(), rctx, caps, actionDef.CommandID, cmdInput)
			if err != nil {
				WriteError(w, err)
				return
			}
			WriteJSON(w, http.StatusOK, resp)

		case "workflow":
			if wfEngine == nil || actionDef.WorkflowID == "" {
				WriteError(w, model.NewBadRequestError("workflows not configured or action has no workflow_id"))
				return
			}
			inst, err := wfEngine.Start(r.Context(), rctx, actionDef.WorkflowID, body.Input)
			if err != nil {
				WriteError(w, err)
				return
			}
			WriteJSON(w, http.StatusCreated, inst)

		default:
			// For navigate, form, and custom actions, return the action metadata
			// for the client to handle.
			WriteJSON(w, http.StatusOK, map[string]any{
				"action_id":   actionDef.ID,
				"type":        actionDef.Type,
				"navigate_to": actionDef.NavigateTo,
				"form_id":     actionDef.FormID,
				"workflow_id": actionDef.WorkflowID,
				"params":      actionDef.Params,
			})
		}
	}
}

// findActionInRegistry searches all domain definitions for an action with the given ID.
func findActionInRegistry(registry *definition.Registry, actionID string) *model.ActionDefinition {
	for _, domain := range registry.AllDomains() {
		for i := range domain.Pages {
			for j := range domain.Pages[i].Actions {
				if domain.Pages[i].Actions[j].ID == actionID {
					return &domain.Pages[i].Actions[j]
				}
			}
		}
	}
	return nil
}
