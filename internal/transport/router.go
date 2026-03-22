package transport

import (
	"net/http"

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
	AppVersion         string
}

// chainMiddleware returns a function that wraps an http.Handler with the
// given middleware chain, applied in order (first middleware is outermost).
func chainMiddleware(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// NewRouter creates an http.Handler with the full middleware pipeline and all
// route registrations. Health checks are handled by Frame at /healthz.
func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	// Auth middleware chain for authenticated routes.
	auth := deps.Authenticate
	if auth == nil {
		auth = func(next http.Handler) http.Handler { return next }
	}

	authChain := chainMiddleware(
		auth,
		BuildRequestContextMiddleware(deps.Config.Identity.ClaimPaths),
		ResolveCapabilities(deps.CapabilityResolver),
		HandlerTimeout(deps.Config.Server.HandlerTimeout),
		RequestLogging,
	)

	// Capabilities
	mux.Handle("GET /ui/capabilities", authChain(handleCapabilities(deps.CapabilityResolver, deps.AppVersion)))

	// Navigation & Pages
	mux.Handle("GET /ui/navigation", authChain(handleNavigation(deps.MenuProvider)))
	mux.Handle("GET /ui/pages/{pageId}", authChain(handleGetPage(deps.PageProvider)))
	mux.Handle("GET /ui/pages/{pageId}/data", authChain(handleGetPageData(deps.PageProvider)))

	// Forms
	mux.Handle("GET /ui/forms/{formId}", authChain(handleGetForm(deps.FormProvider)))
	mux.Handle("GET /ui/forms/{formId}/data", authChain(handleGetFormData(deps.FormProvider)))

	// Schemas
	mux.Handle("GET /ui/schemas/{schemaId}", authChain(handleGetSchema(deps.SchemaProvider)))

	// Commands & Actions
	mux.Handle("POST /ui/commands/{commandId}", authChain(handleCommand(deps.CommandExecutor)))
	mux.Handle("POST /ui/actions/{actionId}", authChain(handleAction(deps.Registry, deps.CommandExecutor)))

	// Resources
	mux.Handle("GET /ui/resources/{resourceType}/search", authChain(handleResourceSearch(deps.SearchProvider)))
	mux.Handle("GET /ui/resources/{resourceType}/{id}", authChain(handleGetResourceItem(deps.ResourceProvider)))
	mux.Handle("GET /ui/resources/{resourceType}", authChain(handleGetResource(deps.ResourceProvider)))

	// Search & Lookups
	mux.Handle("GET /ui/search", authChain(handleSearch(deps.SearchProvider)))
	mux.Handle("GET /ui/lookups/{lookupId}", authChain(handleLookup(deps.LookupProvider)))

	// File operations (proxied to files-svc)
	filesSvc := deps.Config.Services["files-svc"]
	mux.Handle("POST /ui/upload", authChain(handleUpload(filesSvc)))
	mux.Handle("GET /ui/download/{fileId}", authChain(handleDownload(filesSvc)))

	// Global middleware: applied to all routes.
	// CORS is handled by the API gateway — not duplicated here.
	var handler http.Handler = mux
	handler = InjectTraceContext(handler)
	handler = SecurityHeaders(handler)
	handler = RequestID(handler)
	handler = Recovery(handler)

	return handler
}
