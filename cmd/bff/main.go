// Package main is the entry point for the Thesa BFF server.
// It wires all dependencies together and starts the HTTP server using Frame.
package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/pitabwire/frame"
	"github.com/pitabwire/util"

	"github.com/pitabwire/thesa/internal/capability"
	"github.com/pitabwire/thesa/internal/command"
	"github.com/pitabwire/thesa/internal/config"
	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/internal/metadata"
	"github.com/pitabwire/thesa/internal/observability"
	"github.com/pitabwire/thesa/internal/openapi"
	"github.com/pitabwire/thesa/internal/search"
	"github.com/pitabwire/thesa/internal/transport"
)

// Build-time variables set via ldflags:
//
//	go build -ldflags "-X main.version=1.0.0 -X main.commit=abc1234"
var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	ctx := context.Background()
	log := util.Log(ctx)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.WithError(err).Fatal("configuration error")
	}

	// Load OpenAPI specs.
	oaIndex := openapi.NewIndex()
	specSources := buildSpecSources(cfg.Specs)
	if err := oaIndex.Load(specSources); err != nil {
		log.WithError(err).Fatal("OpenAPI index load failed")
	}

	// Load definitions.
	loader := definition.NewLoader()
	defs, err := loader.LoadAll(cfg.Definitions.Directories)
	if err != nil {
		log.WithError(err).Fatal("definition loading failed")
	}

	validator := definition.NewValidator()
	verrs := validator.Validate(defs, oaIndex)
	if len(verrs) > 0 {
		for _, ve := range verrs {
			log.Error("definition validation error", "error", ve.Error())
		}
		log.Fatal("definition validation failed", "errors", len(verrs))
	}

	registry := definition.NewRegistry(defs)

	// Capability resolver.
	capResolver, err := buildCapabilityResolver(cfg.Capability)
	if err != nil {
		log.WithError(err).Fatal("capability resolver initialization failed")
	}

	// Create Frame service first (for its HTTP client).
	ctx, svc := frame.NewServiceWithContext(ctx,
		frame.WithConfig(cfg),
	)

	// Now use Frame's HTTP client for the invoker.
	httpClient := svc.HTTPClientManager().Client(ctx)

	// Build invoker registry.
	sdkHandlers := invoker.NewSDKHandlerRegistry()
	invokerReg := invoker.NewRegistry()
	invokerReg.Register(invoker.NewOpenAPIOperationInvoker(oaIndex, cfg.Services, httpClient))
	invokerReg.Register(invoker.NewSDKOperationInvoker(sdkHandlers))

	// Build providers.
	cmdExecutor := command.NewCommandExecutor(registry, invokerReg, oaIndex)

	actionProvider := metadata.NewActionProvider()
	menuProvider := metadata.NewMenuProvider(registry, invokerReg)
	pageProvider := metadata.NewPageProvider(registry, invokerReg, actionProvider)
	formProvider := metadata.NewFormProvider(registry, invokerReg, actionProvider)
	schemaProvider := metadata.NewSchemaProvider(registry)
	resourceProvider := metadata.NewResourceProvider(registry, invokerReg, oaIndex)
	searchProvider := search.NewSearchProvider(
		registry, invokerReg,
		cfg.Search.TimeoutPerProvider,
		cfg.Search.MaxResultsPerProvider,
	)
	lookupProvider := search.NewLookupProvider(
		registry, invokerReg,
		cfg.Lookup.Cache.TTL,
		cfg.Lookup.Cache.MaxEntries,
	)

	// Build HTTP router.
	jwks := transport.NewJWKSClient(cfg.Identity.JWKSURL, cfg.Identity.JWKSCacheTTL, httpClient)

	specServiceIDs := make([]string, 0, len(specSources))
	for _, s := range specSources {
		specServiceIDs = append(specServiceIDs, s.ServiceID)
	}

	observability.Version = version
	observability.Commit = commit

	router := transport.NewRouter(transport.Dependencies{
		Config:             cfg,
		Authenticate:       transport.JWTAuthenticator(cfg.Identity, jwks),
		CapabilityResolver: capResolver,
		Registry:           registry,
		MenuProvider:       menuProvider,
		PageProvider:       pageProvider,
		FormProvider:       formProvider,
		SchemaProvider:     schemaProvider,
		ResourceProvider:   resourceProvider,
		CommandExecutor:    cmdExecutor,
		SearchProvider:     searchProvider,
		LookupProvider:     lookupProvider,
		HealthHandler:      observability.HandleHealth(),
		ReadyHandler: observability.HandleReady(observability.ReadinessChecks{
			DefinitionsLoaded: func() bool { return len(registry.AllDomains()) > 0 },
			OpenAPILoaded: func() bool {
				for _, svcID := range specServiceIDs {
					if len(oaIndex.AllOperationIDs(svcID)) > 0 {
						return true
					}
				}
				return len(specServiceIDs) == 0
			},
		}),
		AppVersion: version,
	})

	// Set the handler on the service.
	svc.Init(ctx,
		frame.WithHTTPHandler(router),
		frame.WithHealthCheckPath("/ui/health"),
	)

	// Add health checks.
	svc.AddHealthCheck(frame.CheckerFunc(func() error {
		if len(registry.AllDomains()) == 0 {
			return fmt.Errorf("no definitions loaded")
		}
		return nil
	}))

	log = util.Log(ctx)
	log.Info("server starting",
		"version", version,
		"commit", commit,
		"definitions", len(defs),
	)

	serverPort := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := svc.Run(ctx, serverPort); err != nil {
		log.WithError(err).Fatal("server failed")
	}
}

// buildSpecSources converts config spec sources to openapi.SpecSource.
func buildSpecSources(specsCfg config.SpecsConfig) []openapi.SpecSource {
	sources := make([]openapi.SpecSource, len(specsCfg.Sources))
	for i, s := range specsCfg.Sources {
		specPath := s.SpecFile
		if specsCfg.Directory != "" && !filepath.IsAbs(specPath) {
			specPath = filepath.Join(specsCfg.Directory, specPath)
		}
		sources[i] = openapi.SpecSource{
			ServiceID: s.ServiceID,
			SpecPath:  specPath,
		}
	}
	return sources
}

// buildCapabilityResolver creates the appropriate resolver based on config.
func buildCapabilityResolver(cfg config.CapabilityConfig) (*capability.Resolver, error) {
	switch cfg.Evaluator {
	case "static", "":
		evaluator, err := capability.NewStaticPolicyEvaluator(cfg.StaticPolicyFile)
		if err != nil {
			return nil, fmt.Errorf("static policy: %w", err)
		}
		return capability.NewResolver(evaluator, cfg.Cache.TTL), nil
	default:
		return nil, fmt.Errorf("unsupported capability evaluator: %q", cfg.Evaluator)
	}
}
