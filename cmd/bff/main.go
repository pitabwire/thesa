// Package main is the entry point for the Thesa BFF server.
// It wires all dependencies together and starts the HTTP server.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

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
	os.Exit(run())
}

func run() int {
	// Step 1: Parse CLI flags.
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	// Step 2: Load configuration.
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		return 1
	}

	// Step 3: Initialize telemetry (logger, tracer, metrics).
	observability.Version = version
	observability.Commit = commit

	logger, err := observability.NewLogger(cfg.Observability)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger error: %v\n", err)
		return 1
	}
	defer logger.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	tracingShutdown, err := observability.InitTracing(ctx, cfg.Observability.Tracing, "thesa-bff", version)
	if err != nil {
		logger.Fatal("tracing initialization failed", zap.Error(err))
		return 1
	}

	metrics := observability.InitMetrics(prometheus.DefaultRegisterer)

	// Step 4: Load OpenAPI specs and build index.
	oaIndex := openapi.NewIndex()
	specSources := buildSpecSources(cfg.Specs)
	if err := oaIndex.Load(specSources); err != nil {
		logger.Fatal("OpenAPI index load failed", zap.Error(err))
		return 1
	}

	// Step 5: Load definitions, validate, build registry.
	loader := definition.NewLoader()
	defs, err := loader.LoadAll(cfg.Definitions.Directories)
	if err != nil {
		logger.Fatal("definition loading failed", zap.Error(err))
		return 1
	}

	validator := definition.NewValidator()
	verrs := validator.Validate(defs, oaIndex)
	if len(verrs) > 0 {
		for _, ve := range verrs {
			logger.Error("definition validation error", zap.String("error", ve.Error()))
		}
		logger.Fatal("definition validation failed", zap.Int("errors", len(verrs)))
		return 1
	}

	registry := definition.NewRegistry(defs)

	// Step 6: Initialize capability resolver.
	capResolver, err := buildCapabilityResolver(cfg.Capability)
	if err != nil {
		logger.Fatal("capability resolver initialization failed", zap.Error(err))
		return 1
	}

	// Step 7: Build invoker registry.
	sdkHandlers := invoker.NewSDKHandlerRegistry()
	invokerReg := invoker.NewRegistry()
	invokerReg.Register(invoker.NewOpenAPIOperationInvoker(oaIndex, cfg.Services))
	invokerReg.Register(invoker.NewSDKOperationInvoker(sdkHandlers))

	// Step 8: Build providers.
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

	// Step 9: Build HTTP router.
	jwks := transport.NewJWKSClient(cfg.Identity.JWKSURL, cfg.Identity.JWKSCacheTTL)

	specServiceIDs := make([]string, 0, len(specSources))
	for _, s := range specSources {
		specServiceIDs = append(specServiceIDs, s.ServiceID)
	}
	readinessChecks := observability.ReadinessChecks{
		DefinitionsLoaded: func() bool { return len(registry.AllDomains()) > 0 },
		OpenAPILoaded: func() bool {
			for _, svcID := range specServiceIDs {
				if len(oaIndex.AllOperationIDs(svcID)) > 0 {
					return true
				}
			}
			return len(specServiceIDs) == 0
		},
	}

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
		ReadyHandler:       observability.HandleReady(readinessChecks),
		MetricsHandler:     observability.Handler(),
		AppVersion:         version,
	})

	handler := metrics.MetricsMiddleware(observability.TracingMiddleware(router))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Step 10: Start HTTP server.
	logger.Info("server started",
		zap.Int("port", cfg.Server.Port),
		zap.String("version", version),
		zap.String("commit", commit),
		zap.Int("definitions", len(defs)),
	)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	// Wait for shutdown signal or server error.
	select {
	case <-ctx.Done():
		logger.Info("shutdown initiated")
	case err := <-errCh:
		logger.Error("server error", zap.Error(err))
		return 1
	}

	// Graceful shutdown.
	shutdownTimeout := cfg.Server.ShutdownTimeout
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	if err := tracingShutdown(shutdownCtx); err != nil {
		logger.Error("tracing shutdown error", zap.Error(err))
	}

	_ = metrics

	logger.Info("shutdown complete")
	return 0
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
