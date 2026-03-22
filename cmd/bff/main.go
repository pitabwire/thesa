// Package main is the entry point for the Thesa BFF server.
// It wires all dependencies together and starts the HTTP server using Frame.
package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/pitabwire/frame"
	frameversion "github.com/pitabwire/frame/version"
	"github.com/pitabwire/util"

	"github.com/pitabwire/thesa/internal/capability"
	"github.com/pitabwire/thesa/internal/command"
	"github.com/pitabwire/thesa/internal/config"
	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/internal/metadata"
	"github.com/pitabwire/thesa/internal/openapi"
	"github.com/pitabwire/thesa/internal/search"
	"github.com/pitabwire/thesa/internal/transport"
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

	// Create Frame service (provides HTTP client, telemetry, lifecycle,
	// and SecurityManager with authorization service access).
	// Service name defaults to "service-thesa" but can be overridden
	// via SERVICE_NAME env var (standard for all antinvestor services).
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "service-thesa"
	}
	ctx, svc := frame.NewServiceWithContext(ctx,
		frame.WithName(serviceName),
		frame.WithConfig(cfg),
	)

	httpClient := svc.HTTPClientManager().Client(ctx)

	// Capability resolver — checks each known capability against the
	// authorization service (Keto) using BatchCheck, which evaluates
	// OPL rules, role hierarchies, and computed permissions.
	authorizer := svc.SecurityManager().GetAuthorizer(ctx)
	capChecks := capability.CollectCapabilityChecks(defs, cfg.Services)
	evaluator := capability.NewKetoPolicyEvaluator(authorizer, capChecks)
	capResolver := capability.NewResolver(evaluator, cfg.Capability.Cache.TTL)

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
		AppVersion:         frameversion.Version,
	})

	// Register handler and health checks with Frame.
	svc.Init(ctx, frame.WithHTTPHandler(router))

	svc.AddHealthCheck(frame.CheckerFunc(func() error {
		if len(registry.AllDomains()) == 0 {
			return fmt.Errorf("no definitions loaded")
		}
		return nil
	}))

	svc.AddHealthCheck(frame.CheckerFunc(func() error {
		for _, svcID := range buildSpecServiceIDs(specSources) {
			if len(oaIndex.AllOperationIDs(svcID)) > 0 {
				return nil
			}
		}
		if len(specSources) == 0 {
			return nil
		}
		return fmt.Errorf("no OpenAPI specs loaded")
	}))

	log = util.Log(ctx)
	log.Info("server starting",
		"version", frameversion.Version,
		"commit", frameversion.Commit,
		"definitions", len(defs),
	)

	serverPort := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := svc.Run(ctx, serverPort); err != nil {
		log.WithError(err).Fatal("server failed")
	}
}

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

func buildSpecServiceIDs(sources []openapi.SpecSource) []string {
	ids := make([]string, len(sources))
	for i, s := range sources {
		ids[i] = s.ServiceID
	}
	return ids
}
