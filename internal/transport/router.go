package transport

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/pitabwire/thesa/internal/command"
	"github.com/pitabwire/thesa/internal/config"
	"github.com/pitabwire/thesa/internal/metadata"
	"github.com/pitabwire/thesa/internal/search"
	"github.com/pitabwire/thesa/internal/workflow"
	"github.com/pitabwire/thesa/model"
)

// Dependencies holds all injected dependencies for the HTTP transport layer.
type Dependencies struct {
	Config             *config.Config
	Authenticate       func(http.Handler) http.Handler
	CapabilityResolver model.CapabilityResolver
	MenuProvider       *metadata.MenuProvider
	PageProvider       *metadata.PageProvider
	FormProvider       *metadata.FormProvider
	CommandExecutor    *command.CommandExecutor
	WorkflowEngine     *workflow.Engine
	SearchProvider     *search.SearchProvider
	LookupProvider     *search.LookupProvider
}

// NewRouter creates a chi.Router with the full middleware pipeline and all
// route registrations. Health, readiness, and metrics endpoints bypass the
// authentication middleware.
func NewRouter(deps Dependencies) chi.Router {
	r := chi.NewRouter()

	// Global middleware (layers 1-4): applied to all routes including health.
	r.Use(Recovery)
	r.Use(CORS(deps.Config.Server.CORS))
	r.Use(RequestID)
	r.Use(SecurityHeaders)

	// Public routes — bypass authentication.
	r.Get("/ui/health", handleHealth)
	r.Get("/ui/ready", handleReady)
	r.Get("/metrics", handleMetrics)

	// Authenticated routes — full middleware chain (layers 5-10).
	auth := deps.Authenticate
	if auth == nil {
		auth = func(next http.Handler) http.Handler { return next }
	}

	r.Group(func(r chi.Router) {
		r.Use(auth)
		r.Use(BuildRequestContextMiddleware(deps.Config.Identity.ClaimPaths))
		r.Use(ResolveCapabilities(deps.CapabilityResolver))
		r.Use(HandlerTimeout(deps.Config.Server.HandlerTimeout))
		r.Use(RequestLogging)
		r.Use(MetricsRecording)

		r.Get("/ui/navigation", handleNavigation(deps.MenuProvider))
		r.Get("/ui/pages/{pageId}", handleGetPage(deps.PageProvider))
		r.Get("/ui/pages/{pageId}/data", handleGetPageData(deps.PageProvider))
		r.Get("/ui/forms/{formId}", handleGetForm(deps.FormProvider))
		r.Get("/ui/forms/{formId}/data", handleGetFormData(deps.FormProvider))
		r.Post("/ui/commands/{commandId}", handleCommand(deps.CommandExecutor))
		r.Post("/ui/workflows/{workflowId}/start", handleWorkflowStart(deps.WorkflowEngine))
		r.Post("/ui/workflows/{instanceId}/advance", handleWorkflowAdvance(deps.WorkflowEngine))
		r.Get("/ui/workflows/{instanceId}", handleWorkflowGet(deps.WorkflowEngine))
		r.Post("/ui/workflows/{instanceId}/cancel", handleWorkflowCancel(deps.WorkflowEngine))
		r.Get("/ui/workflows", handleWorkflowList(deps.WorkflowEngine))
		r.Get("/ui/search", handleSearch(deps.SearchProvider))
		r.Get("/ui/lookups/{lookupId}", handleLookup(deps.LookupProvider))
	})

	return r
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleReady(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func handleMetrics(w http.ResponseWriter, _ *http.Request) {
	// Placeholder — will be replaced by Prometheus handler.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}
