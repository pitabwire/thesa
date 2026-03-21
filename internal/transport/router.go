package transport

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/pitabwire/thesa/internal/command"
	"github.com/pitabwire/thesa/internal/config"
	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/metadata"
	"github.com/pitabwire/thesa/internal/search"
	"github.com/pitabwire/thesa/model"
)

// Dependencies holds all injected dependencies for the HTTP transport layer.
type Dependencies struct {
	Config             *config.Config
	Authenticate       func(http.Handler) http.Handler
	CapabilityResolver model.CapabilityResolver
	Registry           *definition.Registry
	MenuProvider       *metadata.MenuProvider
	PageProvider       *metadata.PageProvider
	FormProvider       *metadata.FormProvider
	SchemaProvider     *metadata.SchemaProvider
	ResourceProvider   *metadata.ResourceProvider
	CommandExecutor    *command.CommandExecutor
	SearchProvider     *search.SearchProvider
	LookupProvider     *search.LookupProvider
	HealthHandler      http.HandlerFunc
	ReadyHandler       http.HandlerFunc
	MetricsHandler     http.Handler
	AppVersion         string
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
	if deps.HealthHandler != nil {
		r.Get("/ui/health", deps.HealthHandler)
	}
	if deps.ReadyHandler != nil {
		r.Get("/ui/ready", deps.ReadyHandler)
	}
	if deps.MetricsHandler != nil {
		r.Method(http.MethodGet, "/metrics", deps.MetricsHandler)
	}

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

		// Capabilities
		r.Get("/ui/capabilities", handleCapabilities(deps.CapabilityResolver, deps.AppVersion))

		// Navigation & Pages
		r.Get("/ui/navigation", handleNavigation(deps.MenuProvider))
		r.Get("/ui/pages/{pageId}", handleGetPage(deps.PageProvider))
		r.Get("/ui/pages/{pageId}/data", handleGetPageData(deps.PageProvider))

		// Forms
		r.Get("/ui/forms/{formId}", handleGetForm(deps.FormProvider))
		r.Get("/ui/forms/{formId}/data", handleGetFormData(deps.FormProvider))

		// Schemas
		r.Get("/ui/schemas/{schemaId}", handleGetSchema(deps.SchemaProvider))

		// Commands & Actions
		r.Post("/ui/commands/{commandId}", handleCommand(deps.CommandExecutor))
		r.Post("/ui/actions/{actionId}", handleAction(deps.Registry, deps.CommandExecutor))

		// Resources
		r.Get("/ui/resources/{resourceType}/search", handleResourceSearch(deps.SearchProvider))
		r.Get("/ui/resources/{resourceType}/{id}", handleGetResourceItem(deps.ResourceProvider))
		r.Get("/ui/resources/{resourceType}", handleGetResource(deps.ResourceProvider))

		// Search & Lookups
		r.Get("/ui/search", handleSearch(deps.SearchProvider))
		r.Get("/ui/lookups/{lookupId}", handleLookup(deps.LookupProvider))

		// File operations (proxied to files-svc)
		filesSvc := deps.Config.Services["files-svc"]
		r.Post("/ui/upload", handleUpload(filesSvc))
		r.Get("/ui/download/{fileId}", handleDownload(filesSvc))
	})

	return r
}
