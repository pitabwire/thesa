package transport

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/pitabwire/thesa/internal/workflow"
	"github.com/pitabwire/thesa/model"
)

func handleWorkflowStart(engine *workflow.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}
		workflowID := chi.URLParam(r, "workflowId")

		var body struct {
			Input map[string]any `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteError(w, model.NewBadRequestError("invalid JSON body"))
			return
		}

		inst, err := engine.Start(r.Context(), rctx, workflowID, body.Input)
		if err != nil {
			WriteError(w, err)
			return
		}
		WriteJSON(w, http.StatusCreated, inst)
	}
}

func handleWorkflowAdvance(engine *workflow.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}
		instanceID := chi.URLParam(r, "instanceId")

		var body struct {
			Event string         `json:"event"`
			Input map[string]any `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteError(w, model.NewBadRequestError("invalid JSON body"))
			return
		}

		inst, err := engine.Advance(r.Context(), rctx, instanceID, body.Event, body.Input)
		if err != nil {
			WriteError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, inst)
	}
}

func handleWorkflowGet(engine *workflow.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}
		instanceID := chi.URLParam(r, "instanceId")

		desc, err := engine.Get(r.Context(), rctx, instanceID)
		if err != nil {
			WriteError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, desc)
	}
}

func handleWorkflowCancel(engine *workflow.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}
		instanceID := chi.URLParam(r, "instanceId")

		var body struct {
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteError(w, model.NewBadRequestError("invalid JSON body"))
			return
		}

		if err := engine.Cancel(r.Context(), rctx, instanceID, body.Reason); err != nil {
			WriteError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
	}
}

func handleWorkflowList(engine *workflow.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}

		filters := model.WorkflowFilters{
			Status:     r.URL.Query().Get("status"),
			WorkflowID: r.URL.Query().Get("workflow_id"),
			SubjectID:  r.URL.Query().Get("subject_id"),
			Page:       queryInt(r, "page", 1),
			PageSize:   queryInt(r, "page_size", 20),
		}

		summaries, totalCount, err := engine.List(r.Context(), rctx, filters)
		if err != nil {
			WriteError(w, err)
			return
		}

		WriteJSON(w, http.StatusOK, map[string]any{
			"data":        summaries,
			"total_count": totalCount,
			"page":        filters.Page,
			"page_size":   filters.PageSize,
		})
	}
}
